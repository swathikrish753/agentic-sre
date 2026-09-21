# ADR 0001: Hand-rolled agent loop, MCP for tools

- Status: Accepted
- Date: 2026-09-21

## Context
We need an agent that plans, calls tools, and reflects. Options ranged from
adopting a full framework (Google ADK for Go, Microsoft Agent Framework for Go)
to building the loop ourselves.

## Decision
Build the plan/act/observe/reflect loop by hand in `internal/agent`, and expose
tools over the **official MCP Go SDK** rather than framework-native tool APIs.

## Consequences
- (+) Full understanding and control of the reasoning loop — the part most worth
  being able to explain.
- (+) MCP is the 2026 industry standard; our tool servers are reusable by any
  MCP client, not locked to one framework.
- (−) More code to write and maintain than gluing a framework together.
- Revisit if loop complexity (multi-agent orchestration) outgrows a hand-rolled core.
