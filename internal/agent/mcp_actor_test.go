package agent

import (
	"context"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/swathikrish753/agentic-sre/internal/tools"
)

// newTestMCPActor connects a real MCP client to the real tools server through in-memory pipes.
func newTestMCPActor(t *testing.T) *MCPActor {
	t.Helper()
	ctx := context.Background()

	clientT, serverT := mcp.NewInMemoryTransports()
	if _, err := tools.NewServer().Connect(ctx, serverT, nil); err != nil {
		t.Fatal(err)
	}
	session, err := mcp.NewClient(&mcp.Implementation{Name: "copilot-test", Version: "v0"}, nil).Connect(ctx, clientT, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { session.Close() })

	a, err := NewMCPActor(ctx, session)
	if err != nil {
		t.Fatal(err)
	}
	return a
}

func TestMCPActorAct(t *testing.T) {
	a := newTestMCPActor(t)
	ctx := context.Background()

	if got := strings.Join(a.ToolNames(), ","); got != "get_logs,get_pods" {
		t.Fatalf("ToolNames = %s", got)
	}

	tests := []struct {
		name, tool, input string
		want, wantErr     string
	}{
		{name: "plain input", tool: "get_logs", input: "cart-7d9f8b6c5-x2k4p", want: "connection refused"},
		{name: "json input", tool: "get_logs", input: `{"pod":"cart-7d9f8b6c5-x2k4p"}`, want: "redis:6379"},
		{name: "optional input left empty", tool: "get_pods", input: "", want: "CrashLoopBackOff"},
		{name: "tool error reaches the model", tool: "get_logs", input: "nope", wantErr: `pod "nope" not found`},
		{name: "missing required arg", tool: "get_logs", input: "", wantErr: "pod"},
		{name: "unknown tool", tool: "delete_cluster", input: "now", wantErr: `unknown tool "delete_cluster"`},
		{name: "bad json", tool: "get_logs", input: "{pod:", wantErr: "not valid JSON"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := a.Act(ctx, tt.tool, tt.input)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("err = %v, want it to contain %q", err, tt.wantErr)
				}
				return
			}
			if err != nil || !strings.Contains(got, tt.want) {
				t.Fatalf("got %q, %v; want %q", got, err, tt.want)
			}
		})
	}
}

// The same Day 2 loop, unchanged, now driving real MCP tools.
func TestRunWithMCPActor(t *testing.T) {
	brain := &scriptedLLM{replies: []string{
		"Thought: check pods\nAction: get_pods\nAction Input: default",
		"Thought: cart is crashing\nAction: get_logs\nAction Input: cart-7d9f8b6c5-x2k4p",
		"Final Answer: cart cannot reach redis:6379",
	}}
	a := New(brain, newTestMCPActor(t), "test", 6, quietLogger())

	if _, err := a.Run(context.Background(), "Why is cart down?"); err != nil {
		t.Fatal(err)
	}
	lastObs := brain.last[len(brain.last)-1].Content
	if !strings.Contains(lastObs, "Observation: panic: failed to connect to redis") {
		t.Fatalf("last observation = %q", lastObs)
	}
}
