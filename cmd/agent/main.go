package main

import (
	"fmt"
	"os"

	"github.com/swathikrish753/agentic-sre/pkg/config"
	"github.com/swathikrish753/agentic-sre/pkg/logger"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintln(os.Stderr, "config error:", err)
		os.Exit(1)
	}

	log := logger.New(cfg.Log.Level, cfg.Log.Format)
	log = logger.WithService(log, "agent")

	log.Info("agentic-sre agent bootstrapping (Day 1)")
	log.WithFields(map[string]any{
		"provider": cfg.LLM.Provider,
		"model":    cfg.LLM.Model,
		"base_url": cfg.LLM.BaseURL,
		"timeout":  cfg.LLM.Timeout.String(),
	}).Info("llm configured")
	log.Debug("this line only shows when LOG_LEVEL=debug")
}
