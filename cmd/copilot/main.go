package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/swathikrish753/agentic-sre/internal/agent"
	"github.com/swathikrish753/agentic-sre/internal/llm"
	"github.com/swathikrish753/agentic-sre/internal/trace"
	"github.com/swathikrish753/agentic-sre/pkg/config"
	"github.com/swathikrish753/agentic-sre/pkg/logger"
)

const maxSteps = 6

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintln(os.Stderr, "config error:", err)
		os.Exit(1)
	}
	log := logger.WithService(logger.New(cfg.Log.Level, cfg.Log.Format), "copilot")

	// The whole run gets 2 minutes, then ctx cancels every call in flight.
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// Start the tool server as a child process and speak MCP over its stdin/stdout.
	client := mcp.NewClient(&mcp.Implementation{Name: "copilot", Version: "v0.1.0"}, nil)
	session, err := client.Connect(ctx, &mcp.CommandTransport{Command: exec.Command(cfg.MCP.K8sServer)}, nil)
	if err != nil {
		log.WithError(err).WithField("server", cfg.MCP.K8sServer).Error("start MCP server (run `make build` first)")
		os.Exit(1)
	}
	defer session.Close()

	hands, err := agent.NewMCPActor(ctx, session)
	if err != nil {
		log.WithError(err).Error("list MCP tools")
		os.Exit(1)
	}
	log.WithField("tools", hands.ToolNames()).Info("connected to mcp-k8s")

	// One trace file per run: traces/<run-id>.jsonl.
	runID := trace.NewRunID(time.Now())
	tracer, tracePath, err := trace.NewFile(cfg.Trace.Dir, runID)
	if err != nil {
		log.WithError(err).Error("open trace file")
		os.Exit(1)
	}
	defer tracer.Close()
	log = log.WithField("run_id", runID)

	brain := llm.NewOllamaClient(cfg.LLM.BaseURL, nil)
	a := agent.New(brain, hands, cfg.LLM.Model, maxSteps, log).WithTracer(tracer, runID)

	task := "The cart service is down. Find out why."
	if len(os.Args) > 1 {
		task = os.Args[1]
	}

	answer, err := a.Run(ctx, task)
	log.WithField("trace", tracePath).Info("trace written")
	if err != nil {
		log.WithError(err).Error("agent run failed")
		tracer.Close() // os.Exit skips defers
		os.Exit(1)
	}
	fmt.Println(answer)
}
