// mcp-k8s is a standalone MCP server exposing cluster tools over stdio.
// Any MCP client can launch it: our copilot, Claude Desktop, an IDE.
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/swathikrish753/agentic-sre/internal/tools"
)

func main() {
	// stdout carries the MCP protocol, so errors must go to stderr.
	if err := tools.NewServer().Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		fmt.Fprintln(os.Stderr, "mcp-k8s:", err)
		os.Exit(1)
	}
}
