// Package trace writes one JSONL file per agent run: one line per step, then one summary line.
package trace

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Step is one think-act-observe turn of the loop.
type Step struct {
	Type             string  `json:"type"` // always "step"
	RunID            string  `json:"run_id"`
	Step             int     `json:"step"`
	Action           string  `json:"action,omitempty"`
	ActionInput      string  `json:"action_input,omitempty"`
	Observation      string  `json:"observation,omitempty"`
	FinalAnswer      string  `json:"final_answer,omitempty"`
	BadFormat        bool    `json:"bad_format,omitempty"`
	PromptTokens     int     `json:"prompt_tokens"`
	CompletionTokens int     `json:"completion_tokens"`
	LLMMillis        int64   `json:"llm_ms"`
	ToolMillis       int64   `json:"tool_ms,omitempty"`
	CostINR          float64 `json:"cost_inr"`
}

// Run is the summary line written when a run ends, however it ends.
type Run struct {
	Type             string    `json:"type"` // always "run"
	RunID            string    `json:"run_id"`
	Model            string    `json:"model"`
	Task             string    `json:"task"`
	Outcome          string    `json:"outcome"` // final_answer | max_steps | cancelled | error
	Error            string    `json:"error,omitempty"`
	Steps            int       `json:"steps"`
	PromptTokens     int       `json:"prompt_tokens"`
	CompletionTokens int       `json:"completion_tokens"`
	TotalMillis      int64     `json:"total_ms"`
	CostINR          float64   `json:"cost_inr"`
	CostKnown        bool      `json:"cost_known"`
	Started          time.Time `json:"started"`
}

// Tracer appends events as JSON lines. A nil *Tracer is valid and records nothing,
// so tracing can be switched off without if-statements everywhere.
type Tracer struct {
	mu  sync.Mutex
	enc *json.Encoder
	c   io.Closer
}

// New writes to any io.Writer (a file, a buffer in tests).
func New(w io.Writer) *Tracer {
	t := &Tracer{enc: json.NewEncoder(w)}
	if c, ok := w.(io.Closer); ok {
		t.c = c
	}
	return t
}

// NewFile creates dir/<runID>.jsonl.
func NewFile(dir, runID string) (*Tracer, string, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, "", fmt.Errorf("trace: %w", err)
	}
	path := filepath.Join(dir, runID+".jsonl")
	f, err := os.Create(path)
	if err != nil {
		return nil, "", fmt.Errorf("trace: %w", err)
	}
	return New(f), path, nil
}

// Step records one turn.
func (t *Tracer) Step(s Step) {
	s.Type = "step"
	t.write(s)
}

// Run records the summary.
func (t *Tracer) Run(r Run) {
	r.Type = "run"
	t.write(r)
}

func (t *Tracer) write(v any) {
	if t == nil {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	_ = t.enc.Encode(v) // a trace write must never break a run
}

// Close flushes and closes the file, if there is one.
func (t *Tracer) Close() error {
	if t == nil || t.c == nil {
		return nil
	}
	return t.c.Close()
}

// NewRunID returns a sortable id like 20261125-142209.123.
func NewRunID(now time.Time) string {
	return now.UTC().Format("20060102-150405.000")
}
