// Package agent runs the ReAct loop: think (LLM), act (Actor), observe, repeat.
package agent

import (
	"context"
	"errors"
	"fmt"
	"io"
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
	diary    io.Writer // where trace lines go; nil means no diary
	runID    string
}

// New builds an Agent.
func New(client llm.Client, actor Actor, model string, maxSteps int, log logger.Logger) *Agent {
	return &Agent{llm: client, actor: actor, model: model, maxSteps: maxSteps, log: log}
}

// WithTrace makes Run write a diary line per step and a summary line at the end.
func (a *Agent) WithTrace(diary io.Writer, runID string) *Agent {
	a.diary, a.runID = diary, runID
	return a
}

// Run loops until a Final Answer, ctx is done, the LLM fails, or maxSteps is hit.
func (a *Agent) Run(ctx context.Context, task string) (answer string, err error) {
	start := time.Now()
	total := trace.Summary{RunID: a.runID, Model: a.model}

	// However Run ends, write the summary line last.
	defer func() {
		total.Outcome = outcome(err)
		total.Millis = time.Since(start).Milliseconds()
		trace.Write(a.diary, total)
		a.log.WithFields(map[string]any{
			"outcome": total.Outcome, "steps": total.Steps,
			"tokens_in": total.TokensIn, "tokens_out": total.TokensOut, "cost_inr": total.CostINR,
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
		stepStart := time.Now()
		line := trace.Step{Step: step}

		// Think.
		resp, err := a.llm.Complete(ctx, llm.CompletionRequest{Model: a.model, Messages: history})
		if err != nil {
			return "", fmt.Errorf("agent: step %d: %w", step, err)
		}
		history = append(history, llm.Message{Role: llm.RoleAssistant, Content: resp.Text})
		line.TokensIn, line.TokensOut = resp.Usage.PromptTokens, resp.Usage.CompletionTokens
		line.CostINR, _ = trace.Cost(a.model, line.TokensIn, line.TokensOut)

		parsed, err := parseReply(resp.Text)
		switch {
		case errors.Is(err, ErrBadFormat):
			a.log.WithField("step", step).Warn("bad format, sending reminder")
			history = append(history, llm.Message{Role: llm.RoleUser, Content: formatReminder})

		case parsed.FinalAnswer != "":
			a.log.WithField("step", step).Info("final answer")
			line.FinalAnswer = parsed.FinalAnswer
			answer = parsed.FinalAnswer

		default:
			// Act.
			observation, err := a.actor.Act(ctx, parsed.Action, parsed.ActionInput)
			if err != nil {
				observation = "error: " + err.Error()
			}
			a.log.WithFields(map[string]any{"step": step, "tool": parsed.Action, "input": parsed.ActionInput}).Info("tool call")
			line.Action, line.Input, line.Observation = parsed.Action, parsed.ActionInput, observation

			// Observe.
			history = append(history, llm.Message{Role: llm.RoleUser, Content: "Observation: " + observation})
		}

		// Write this step's diary line and add it to the totals.
		line.Millis = time.Since(stepStart).Milliseconds()
		trace.Write(a.diary, line)
		total.Steps = step
		total.TokensIn += line.TokensIn
		total.TokensOut += line.TokensOut
		total.CostINR += line.CostINR

		if answer != "" {
			return answer, nil
		}
	}
	return "", ErrMaxSteps
}

// outcome names how a run ended, for the summary line.
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
