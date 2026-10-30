// Package tools is the MCP server that exposes cluster tools to any MCP client.
// Today the data is canned; Day 6 swaps it for real Kubernetes calls.
package tools

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// GetPodsInput is the JSON input for get_pods. The struct tags become the tool's JSON Schema.
type GetPodsInput struct {
	Namespace string `json:"namespace,omitempty" jsonschema:"Kubernetes namespace, defaults to default"`
}

// GetLogsInput is the JSON input for get_logs.
type GetLogsInput struct {
	Pod string `json:"pod" jsonschema:"exact pod name, as shown by get_pods"`
}

// NewServer builds an MCP server with every cluster tool registered.
func NewServer() *mcp.Server {
	s := mcp.NewServer(&mcp.Implementation{Name: "mcp-k8s", Version: "v0.1.0"}, nil)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "get_pods",
		Description: "List pods in a namespace with their status and restart count.",
	}, getPods)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "get_logs",
		Description: "Return the recent logs of one pod.",
	}, getLogs)

	return s
}

func getPods(_ context.Context, _ *mcp.CallToolRequest, in GetPodsInput) (*mcp.CallToolResult, any, error) {
	ns := in.Namespace
	if ns == "" {
		ns = "default"
	}
	pods, ok := fakePods[ns]
	if !ok {
		return nil, nil, fmt.Errorf("namespace %q not found", ns)
	}
	return text(pods), nil, nil
}

func getLogs(_ context.Context, _ *mcp.CallToolRequest, in GetLogsInput) (*mcp.CallToolResult, any, error) {
	logs, ok := fakeLogs[in.Pod]
	if !ok {
		return nil, nil, fmt.Errorf("pod %q not found", in.Pod)
	}
	return text(logs), nil, nil
}

// text wraps a string as a tool result the model can read.
func text(s string) *mcp.CallToolResult {
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: s}}}
}

// Canned cluster state until the real cluster arrives (Day 5-6).
var fakePods = map[string]string{
	"default": "NAME                       READY   STATUS             RESTARTS\n" +
		"cart-7d9f8b6c5-x2k4p       0/1     CrashLoopBackOff   6\n" +
		"frontend-5c8d7f9b4-q9w8e   1/1     Running            0",
}

var fakeLogs = map[string]string{
	"cart-7d9f8b6c5-x2k4p":     "panic: failed to connect to redis at redis:6379: connection refused",
	"frontend-5c8d7f9b4-q9w8e": "GET /healthz 200",
}
