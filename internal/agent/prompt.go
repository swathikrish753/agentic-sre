package agent

import (
	"fmt"
	"strings"
)

// systemPromptTemplate teaches the model the exact reply format the parser expects.
const systemPromptTemplate = `You are an SRE assistant that diagnoses problems in a Kubernetes cluster.

You can use these tools:
%s

Always reply in exactly one of these two formats.

To use a tool:
Thought: <your reasoning>
Action: <one tool name from the list above>
Action Input: <the input for the tool>

When you know the answer:
Thought: <your reasoning>
Final Answer: <your answer>

Use one Action per reply. Never write "Observation:" yourself; the tool result will be sent to you.`

// buildSystemPrompt fills the tool list into the template.
func buildSystemPrompt(toolNames []string) string {
	lines := make([]string, len(toolNames))
	for i, name := range toolNames {
		lines[i] = "- " + name
	}
	return fmt.Sprintf(systemPromptTemplate, strings.Join(lines, "\n"))
}

// formatReminder is sent back when the model's reply can't be parsed.
const formatReminder = `Your last reply did not follow the format. Reply with either
"Thought: ... / Action: ... / Action Input: ..." or "Thought: ... / Final Answer: ...".`
