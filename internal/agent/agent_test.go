package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/swathikrish753/agentic-sre/internal/llm"
	"github.com/swathikrish753/agentic-sre/internal/trace"
	"github.com/swathikrish753/agentic-sre/pkg/logger"
)

// scriptedLLM returns fixed replies in order, so the loop is tested without Ollama.
type scriptedLLM struct {
	replies []string
	calls   int
	last    []llm.Message
}

// Each reply reports fixed token counts, so tests can check the trace totals.
func (s *scriptedLLM) Complete(_ context.Context, req llm.CompletionRequest) (llm.Response, error) {
	s.last = req.Messages
	if s.calls >= len(s.replies) {
		return llm.Response{}, errors.New("script exhausted")
	}
	r := s.replies[s.calls]
	s.calls++
	return llm.Response{Text: r, Usage: llm.Usage{PromptTokens: 100 * s.calls, CompletionTokens: 20}}, nil
}

func (s *scriptedLLM) Stream(ctx context.Context, req llm.CompletionRequest, onToken func(string)) (llm.Response, error) {
	return s.Complete(ctx, req)
}

func quietLogger() logger.Logger {
	l := logrus.New()
	l.SetOutput(io.Discard)
	return logrus.NewEntry(l)
}

func TestRunHappyPath(t *testing.T) {
	brain := &scriptedLLM{replies: []string{
		"Thought: check pods\nAction: get_pods\nAction Input: default",
		"Thought: cart is crashing\nAction: get_logs\nAction Input: cart-7d9f8b6c5-x2k4p",
		"Thought: redis refused\nFinal Answer: cart is in CrashLoopBackOff because redis:6379 refuses connections",
	}}
	a := New(brain, FakeActor{}, "test", 6, quietLogger())

	got, err := a.Run(context.Background(), "Why is the cart service down?")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "redis") {
		t.Fatalf("unexpected answer %q", got)
	}
	// The last call must have seen the logs as an observation.
	lastObs := brain.last[len(brain.last)-1]
	if lastObs.Role != llm.RoleUser || !strings.Contains(lastObs.Content, "Observation: panic") {
		t.Fatalf("last message = %+v", lastObs)
	}
}

func TestRunMaxSteps(t *testing.T) {
	loop := "Thought: again\nAction: get_pods\nAction Input: default"
	brain := &scriptedLLM{replies: []string{loop, loop, loop}}
	a := New(brain, FakeActor{}, "test", 3, quietLogger())

	if _, err := a.Run(context.Background(), "task"); !errors.Is(err, ErrMaxSteps) {
		t.Fatalf("err = %v, want ErrMaxSteps", err)
	}
}

func TestRunBadFormatAndToolError(t *testing.T) {
	brain := &scriptedLLM{replies: []string{
		"no idea what format you want",
		"Thought: try a tool\nAction: delete_cluster\nAction Input: now",
		"Final Answer: done",
	}}
	a := New(brain, FakeActor{}, "test", 6, quietLogger())

	if _, err := a.Run(context.Background(), "task"); err != nil {
		t.Fatal(err)
	}
	var sawReminder, sawToolErr bool
	for _, m := range brain.last {
		sawReminder = sawReminder || m.Content == formatReminder
		sawToolErr = sawToolErr || strings.Contains(m.Content, `Observation: error: unknown tool "delete_cluster"`)
	}
	if !sawReminder || !sawToolErr {
		t.Fatalf("reminder=%v toolErr=%v", sawReminder, sawToolErr)
	}
}

func TestRunCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	a := New(&scriptedLLM{}, FakeActor{}, "test", 6, quietLogger())
	if _, err := a.Run(ctx, "task"); !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want context.Canceled", err)
	}
}

func TestRunWritesTrace(t *testing.T) {
	brain := &scriptedLLM{replies: []string{
		"Thought: check pods\nAction: get_pods\nAction Input: default",
		"Final Answer: cart cannot reach redis",
	}}
	var buf bytes.Buffer
	a := New(brain, FakeActor{}, "llama3.2:3b", 6, quietLogger()).WithTracer(trace.New(&buf), "run-1")

	if _, err := a.Run(context.Background(), "Why is cart down?"); err != nil {
		t.Fatal(err)
	}

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 3 { // 2 steps + 1 summary
		t.Fatalf("got %d lines:\n%s", len(lines), buf.String())
	}
	var step1 trace.Step
	var sum trace.Run
	json.Unmarshal([]byte(lines[0]), &step1)
	json.Unmarshal([]byte(lines[2]), &sum)

	if step1.Action != "get_pods" || step1.PromptTokens != 100 || !strings.Contains(step1.Observation, "CrashLoopBackOff") {
		t.Fatalf("step 1 = %+v", step1)
	}
	// 100 + 200 prompt tokens, 20 + 20 completion tokens, local model so ₹0 and a known price.
	if sum.Outcome != "final_answer" || sum.Steps != 2 || sum.PromptTokens != 300 || sum.CompletionTokens != 40 ||
		sum.CostINR != 0 || !sum.CostKnown || sum.RunID != "run-1" {
		t.Fatalf("summary = %+v", sum)
	}
}

func TestRunTraceOutcomes(t *testing.T) {
	loop := "Thought: again\nAction: get_pods\nAction Input: default"
	tests := []struct {
		name    string
		replies []string
		ctx     func() context.Context
		want    string
	}{
		{"max steps", []string{loop, loop}, context.Background, "max_steps"},
		{"llm error", nil, context.Background, "error"},
		{"cancelled", nil, func() context.Context {
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			return ctx
		}, "cancelled"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			a := New(&scriptedLLM{replies: tt.replies}, FakeActor{}, "llama3.2:3b", 2, quietLogger()).WithTracer(trace.New(&buf), "r")
			if _, err := a.Run(tt.ctx(), "task"); err == nil {
				t.Fatal("expected an error")
			}
			lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
			var sum trace.Run
			json.Unmarshal([]byte(lines[len(lines)-1]), &sum)
			if sum.Type != "run" || sum.Outcome != tt.want {
				t.Fatalf("summary = %+v", sum)
			}
		})
	}
}
