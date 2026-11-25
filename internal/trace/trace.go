// Package trace is the agent's diary: one JSON line per step, then one summary line.
package trace

import (
	"encoding/json"
	"io"
)

// Step is one diary line, written after each turn of the loop.
type Step struct {
	Step        int     `json:"step"`
	Action      string  `json:"action,omitempty"`
	Input       string  `json:"input,omitempty"`
	Observation string  `json:"observation,omitempty"`
	FinalAnswer string  `json:"final_answer,omitempty"`
	TokensIn    int     `json:"tokens_in"`
	TokensOut   int     `json:"tokens_out"`
	Millis      int64   `json:"ms"`
	CostINR     float64 `json:"cost_inr"`
}

// Summary is the last diary line, written when the run ends.
type Summary struct {
	RunID     string  `json:"run_id"`
	Model     string  `json:"model"`
	Outcome   string  `json:"outcome"` // final_answer | max_steps | cancelled | error
	Steps     int     `json:"steps"`
	TokensIn  int     `json:"tokens_in"`
	TokensOut int     `json:"tokens_out"`
	Millis    int64   `json:"ms"`
	CostINR   float64 `json:"cost_inr"`
}

// Write adds one line to the diary. No diary (w == nil) means do nothing.
func Write(w io.Writer, line any) {
	if w == nil {
		return
	}
	json.NewEncoder(w).Encode(line) // Encode adds the newline
}
