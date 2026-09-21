# ADR 0002: Local model (Ollama) behind a swappable LLM interface

- Status: Accepted
- Date: 2026-09-21

## Context
The project must run with no paid accounts. Local models are free and private but
can be less reliable at tool-calling than frontier cloud models.

## Decision
Default to **Ollama** locally, but program against an internal `LLM` interface so
the provider is swappable. Model choice is env-driven (`LLM_MODEL`).

## Consequences
- (+) Zero-cost, self-hosted default; nothing to sign up for.
- (+) If local tool-calling proves shaky, swap in a cloud provider behind the same
  interface with no call-site changes.
- (−) Must defensively handle malformed tool calls from smaller models (planned Day 10).
