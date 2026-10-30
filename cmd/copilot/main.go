package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/swathikrish753/agentic-sre/internal/agent"
	"github.com/swathikrish753/agentic-sre/internal/llm"
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

	brain := llm.NewOllamaClient(cfg.LLM.BaseURL, nil)
	hands := agent.FakeActor{}
	a := agent.New(brain, hands, cfg.LLM.Model, maxSteps, log)

	task := "The cart service is down. Find out why."
	if len(os.Args) > 1 {
		task = os.Args[1]
	}

	answer, err := a.Run(ctx, task)
	if err != nil {
		log.WithError(err).Error("agent run failed")
		os.Exit(1)
	}
	fmt.Println(answer)
}
