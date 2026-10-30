package agent

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/swathikrish753/agentic-sre/internal/llm"
	"github.com/swathikrish753/agentic-sre/pkg/logger"
)

// scriptedLLM returns fixed replies in order, so the loop is tested without Ollama.
type scriptedLLM struct {
	replies []string
	calls   int
	last    []llm.Message
}

func (s *scriptedLLM) Complete(_ context.Context, req llm.CompletionRequest) (string, error) {
	s.last = req.Messages
	if s.calls >= len(s.replies) {
		return "", errors.New("script exhausted")
	}
	r := s.replies[s.calls]
	s.calls++
	return r, nil
}

func (s *scriptedLLM) Stream(ctx context.Context, req llm.CompletionRequest, onToken func(string)) (string, error) {
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
