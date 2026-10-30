package agent

import (
	"errors"
	"strings"
)

// ErrBadFormat means the reply had neither an Action nor a Final Answer.
var ErrBadFormat = errors.New("agent: reply does not follow the Thought/Action format")

// Step is one parsed model reply.
type Step struct {
	Thought     string
	Action      string
	ActionInput string
	FinalAnswer string
}

// parseReply turns the model's text into a Step.
func parseReply(text string) (Step, error) {
	// The model sometimes invents its own "Observation:". Everything after it is fiction.
	if i := strings.Index(text, "Observation:"); i >= 0 {
		text = text[:i]
	}

	var step Step
	var finalLines []string
	inFinal := false

	for _, line := range strings.Split(text, "\n") {
		trimmed := strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(trimmed, "Thought:"):
			step.Thought = strings.TrimSpace(strings.TrimPrefix(trimmed, "Thought:"))
			inFinal = false
		// "Action Input:" must be checked before "Action:" because it starts with "Action".
		case strings.HasPrefix(trimmed, "Action Input:"):
			step.ActionInput = strings.TrimSpace(strings.TrimPrefix(trimmed, "Action Input:"))
			inFinal = false
		case strings.HasPrefix(trimmed, "Action:"):
			step.Action = strings.TrimSpace(strings.TrimPrefix(trimmed, "Action:"))
			inFinal = false
		case strings.HasPrefix(trimmed, "Final Answer:"):
			finalLines = append(finalLines, strings.TrimSpace(strings.TrimPrefix(trimmed, "Final Answer:")))
			inFinal = true
		case inFinal:
			finalLines = append(finalLines, line)
		}
	}

	// If both appear, the Action wins: the model hasn't seen the tool result yet.
	if step.Action != "" {
		return step, nil
	}
	if len(finalLines) > 0 {
		step.FinalAnswer = strings.TrimSpace(strings.Join(finalLines, "\n"))
		if step.FinalAnswer != "" {
			return step, nil
		}
	}
	return Step{}, ErrBadFormat
}
