// Package agent runs the ReAct loop: think (LLM), act (Actor), observe, repeat.
package agent

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/swathikrish753/agentic-sre/internal/llm"
	"github.com/swathikrish753/agentic-sre/internal/trace"
	"github.com/swathikrish753/agentic-sre/pkg/logger"
)

// ErrMaxSteps means the agent ran out of steps without a Final Answer.
var ErrMaxSteps = errors.New("agent: reached max steps without a final answer")

// Agent wires a brain (llm.Client) to hands (Actor).
type Agent struct {
	llm      llm.Client
	actor    Actor
	model    string
	maxSteps int
	log      logger.Logger
	tracer   *trace.Tracer // nil means no trace file
	runID    string
}

// New builds an Agent.
func New(client llm.Client, actor Actor, model string, maxSteps int, log logger.Logger) *Agent {
	return &Agent{llm: client, actor: actor, model: model, maxSteps: maxSteps, log: log}
}

// WithTracer makes Run write one JSONL line per step and a summary line at the end.
func (a *Agent) WithTracer(t *trace.Tracer, runID string) *Agent {
	a.tracer, a.runID = t, runID
	return a
}

// Run loops until a Final Answer, ctx is done, the LLM fails, or maxSteps is hit.
func (a *Agent) Run(ctx context.Context, task string) (answer string, err error) {
	sum := trace.Run{RunID: a.runID, Model: a.model, Task: task, Started: time.Now(), CostKnown: true}
	// Whatever way Run ends, write the summary on the way out.
	defer func() {
		sum.TotalMillis = time.Since(sum.Started).Milliseconds()
		sum.Outcome = outcome(err)
		if err != nil {
			sum.Error = err.Error()
		}
		a.tracer.Run(sum)
		a.log.WithFields(map[string]any{
			"outcome": sum.Outcome, "steps": sum.Steps, "prompt_tokens": sum.PromptTokens,
			"completion_tokens": sum.CompletionTokens, "total_ms": sum.TotalMillis, "cost_inr": sum.CostINR,
		}).Info("run finished")
	}()

	history := []llm.Message{
		{Role: llm.RoleSystem, Content: buildSystemPrompt(a.actor.ToolNames())},
		{Role: llm.RoleUser, Content: task},
	}

	for step := 1; step <= a.maxSteps; step++ {
		if err := ctx.Err(); err != nil {
			return "", err
		}
		sum.Steps = step

		// Think.
		start := time.Now()
		resp, err := a.llm.Complete(ctx, llm.CompletionRequest{Model: a.model, Messages: history})
		if err != nil {
			return "", fmt.Errorf("agent: step %d: %w", step, err)
		}
		rec := trace.Step{
			RunID:            a.runID,
			Step:             step,
			PromptTokens:     resp.Usage.PromptTokens,
			CompletionTokens: resp.Usage.CompletionTokens,
			LLMMillis:        time.Since(start).Milliseconds(),
		}
		cost, known := trace.Cost(a.model, resp.Usage)
		rec.CostINR = cost
		sum.PromptTokens += rec.PromptTokens
		sum.CompletionTokens += rec.CompletionTokens
		sum.CostINR += cost
		sum.CostKnown = sum.CostKnown && known

		log := a.log.WithFields(map[string]any{
			"step": step, "prompt_tokens": rec.PromptTokens, "completion_tokens": rec.CompletionTokens, "llm_ms": rec.LLMMillis,
		})
		history = append(history, llm.Message{Role: llm.RoleAssistant, Content: resp.Text})

		parsed, err := parseReply(resp.Text)
		if errors.Is(err, ErrBadFormat) {
			log.Warn("bad format, sending reminder")
			rec.BadFormat = true
			a.tracer.Step(rec)
			history = append(history, llm.Message{Role: llm.RoleUser, Content: formatReminder})
			continue
		}

		if parsed.FinalAnswer != "" {
			log.Info("final answer")
			rec.FinalAnswer = parsed.FinalAnswer
			a.tracer.Step(rec)
			return parsed.FinalAnswer, nil
		}

		// Act.
		toolStart := time.Now()
		observation, err := a.actor.Act(ctx, parsed.Action, parsed.ActionInput)
		if err != nil {
			observation = "error: " + err.Error()
		}
		rec.Action, rec.ActionInput, rec.Observation = parsed.Action, parsed.ActionInput, observation
		rec.ToolMillis = time.Since(toolStart).Milliseconds()
		log.WithFields(map[string]any{"tool": parsed.Action, "input": parsed.ActionInput, "tool_ms": rec.ToolMillis}).Info("tool call")
		a.tracer.Step(rec)

		// Observe.
		history = append(history, llm.Message{Role: llm.RoleUser, Content: "Observation: " + observation})
	}
	return "", ErrMaxSteps
}

// outcome names how a run ended, for the trace summary.
func outcome(err error) string {
	switch {
	case err == nil:
		return "final_answer"
	case errors.Is(err, ErrMaxSteps):
		return "max_steps"
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		return "cancelled"
	default:
		return "error"
	}
}
