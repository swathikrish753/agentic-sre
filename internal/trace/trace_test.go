package trace

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/swathikrish753/agentic-sre/internal/llm"
)

func TestTracerWritesJSONLines(t *testing.T) {
	var buf bytes.Buffer
	tr := New(&buf)
	tr.Step(Step{RunID: "r1", Step: 1, Action: "get_pods", PromptTokens: 120, CompletionTokens: 18, LLMMillis: 900})
	tr.Run(Run{RunID: "r1", Outcome: "final_answer", Steps: 1})

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("got %d lines: %q", len(lines), buf.String())
	}
	var step map[string]any
	if err := json.Unmarshal([]byte(lines[0]), &step); err != nil {
		t.Fatal(err)
	}
	if step["type"] != "step" || step["action"] != "get_pods" || step["prompt_tokens"] != 120.0 {
		t.Fatalf("step line = %s", lines[0])
	}
	if !strings.Contains(lines[1], `"type":"run"`) || !strings.Contains(lines[1], `"outcome":"final_answer"`) {
		t.Fatalf("run line = %s", lines[1])
	}
}

func TestNilTracerIsANoOp(t *testing.T) {
	var tr *Tracer
	tr.Step(Step{Step: 1})
	tr.Run(Run{})
	if err := tr.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestCost(t *testing.T) {
	u := llm.Usage{PromptTokens: 2_000_000, CompletionTokens: 500_000}

	if inr, ok := Cost("llama3.2:3b", u); !ok || inr != 0 {
		t.Fatalf("local model: %v, %v", inr, ok)
	}

	Prices["test-cloud"] = Price{InputPerMTok: 10, OutputPerMTok: 40}
	defer delete(Prices, "test-cloud")
	if inr, ok := Cost("test-cloud", u); !ok || inr != 40 { // 2×10 + 0.5×40
		t.Fatalf("cloud model: %v, %v", inr, ok)
	}

	if _, ok := Cost("unknown-model", u); ok {
		t.Fatal("unknown model must not report a price")
	}
}
