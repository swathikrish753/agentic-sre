package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// MCPActor is the agent's real hands: it calls tools on an MCP server.
type MCPActor struct {
	session *mcp.ClientSession
	tools   map[string]*mcp.Tool
	names   []string
}

var _ Actor = (*MCPActor)(nil)

// NewMCPActor asks the server which tools it has, once, at startup.
func NewMCPActor(ctx context.Context, session *mcp.ClientSession) (*MCPActor, error) {
	res, err := session.ListTools(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("mcp: list tools: %w", err)
	}
	a := &MCPActor{session: session, tools: make(map[string]*mcp.Tool, len(res.Tools))}
	for _, t := range res.Tools {
		a.tools[t.Name] = t
		a.names = append(a.names, t.Name)
	}
	return a, nil
}

func (a *MCPActor) ToolNames() []string { return a.names }

// Act turns the model's text input into MCP arguments, calls the tool, and returns its text.
func (a *MCPActor) Act(ctx context.Context, tool, input string) (string, error) {
	t, ok := a.tools[tool]
	if !ok {
		return "", fmt.Errorf("unknown tool %q", tool)
	}
	args, err := toArguments(t, input)
	if err != nil {
		return "", err
	}

	res, err := a.session.CallTool(ctx, &mcp.CallToolParams{Name: tool, Arguments: args})
	if err != nil {
		return "", fmt.Errorf("mcp: call %s: %w", tool, err)
	}

	var out strings.Builder
	for _, c := range res.Content {
		if tc, ok := c.(*mcp.TextContent); ok {
			out.WriteString(tc.Text)
		}
	}
	// IsError means the tool ran but failed (bad input, pod not found). The model should see why.
	if res.IsError {
		return "", errors.New(out.String())
	}
	return out.String(), nil
}

// toArguments accepts either a JSON object ({"pod":"cart-1"}) or a plain value (cart-1).
// A plain value is put into the tool's first property, which covers one-argument tools.
func toArguments(t *mcp.Tool, input string) (map[string]any, error) {
	input = strings.TrimSpace(input)
	if strings.HasPrefix(input, "{") {
		var args map[string]any
		if err := json.Unmarshal([]byte(input), &args); err != nil {
			return nil, fmt.Errorf("action input is not valid JSON: %w", err)
		}
		return args, nil
	}
	args := map[string]any{}
	if input == "" {
		return args, nil
	}
	prop := firstProperty(t)
	if prop == "" {
		return nil, fmt.Errorf("tool %q takes no input", t.Name)
	}
	args[prop] = input
	return args, nil
}

// firstProperty returns the first required property, or the only property, of the tool's input schema.
func firstProperty(t *mcp.Tool) string {
	schema, ok := t.InputSchema.(map[string]any)
	if !ok {
		return ""
	}
	if req, ok := schema["required"].([]any); ok && len(req) > 0 {
		if name, ok := req[0].(string); ok {
			return name
		}
	}
	if props, ok := schema["properties"].(map[string]any); ok && len(props) == 1 {
		for name := range props {
			return name
		}
	}
	return ""
}
