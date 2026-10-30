// Package agent runs the ReAct loop: think (LLM), act (Actor), observe, repeat.
package agent

import (
	"context"
	"errors"
	"fmt"

	"github.com/swathikrish753/agentic-sre/internal/llm"
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
}

// New builds an Agent.
func New(client llm.Client, actor Actor, model string, maxSteps int, log logger.Logger) *Agent {
	return &Agent{llm: client, actor: actor, model: model, maxSteps: maxSteps, log: log}
}

// Run loops until a Final Answer, ctx is done, the LLM fails, or maxSteps is hit.
func (a *Agent) Run(ctx context.Context, task string) (string, error) {
	history := []llm.Message{
		{Role: llm.RoleSystem, Content: buildSystemPrompt(a.actor.ToolNames())},
		{Role: llm.RoleUser, Content: task},
	}

	for step := 1; step <= a.maxSteps; step++ {
		if err := ctx.Err(); err != nil {
			return "", err
		}

		// Think.
		reply, err := a.llm.Complete(ctx, llm.CompletionRequest{Model: a.model, Messages: history})
		if err != nil {
			return "", fmt.Errorf("agent: step %d: %w", step, err)
		}
		history = append(history, llm.Message{Role: llm.RoleAssistant, Content: reply})

		parsed, err := parseReply(reply)
		if errors.Is(err, ErrBadFormat) {
			a.log.WithField("step", step).Warn("bad format, sending reminder")
			history = append(history, llm.Message{Role: llm.RoleUser, Content: formatReminder})
			continue
		}

		if parsed.FinalAnswer != "" {
			a.log.WithField("step", step).Info("final answer")
			return parsed.FinalAnswer, nil
		}

		// Act.
		a.log.WithFields(map[string]any{"step": step, "tool": parsed.Action, "input": parsed.ActionInput}).Info("tool call")
		observation, err := a.actor.Act(ctx, parsed.Action, parsed.ActionInput)
		if err != nil {
			observation = "error: " + err.Error()
		}

		// Observe.
		history = append(history, llm.Message{Role: llm.RoleUser, Content: "Observation: " + observation})
	}
	return "", ErrMaxSteps
}
