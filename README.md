# agentic-sre

An autonomous SRE agent written in Go. It watches a Kubernetes cluster, diagnoses
failures (CrashLoopBackOff, ImagePullBackOff, OOMKilled, Pending), and proposes fixes
that a human approves before they run. Every run is traced, and an eval harness
measures how well it performs.

> Status: in progress. Days 1-3 of 22 are done (LLM client, ReAct loop, MCP tools).

## How it works

```
cmd/copilot ──▶ Agent.Run (ReAct loop) ──▶ llm.Client ──▶ Ollama (local) / cloud model
                     │
                     ├──▶ MCPActor ──stdio──▶ mcp-k8s (MCP server) ──▶ Kubernetes (Day 6)
                     └──▶ history []Message (the agent's working memory)
```

Each step the model replies in a fixed format:

```
Thought: the cart pod is crashing, check its logs
Action: get_logs
Action Input: cart-7d9f8b6c5-x2k4p
```

The loop runs the tool, sends the result back as `Observation: ...`, and repeats until
the model replies with `Final Answer: ...`.

Two interfaces carry the design, so the loop never changes when either side is swapped:

- **`llm.Client`**: Ollama, a cloud model, or a scripted fake in tests.
- **`Actor`**: real MCP tools via `MCPActor`, or canned `FakeActor` in loop tests.

### Safety built into the loop

- Four exits: final answer, max steps, context timeout, LLM error.
- Model output is not trusted: invented `Observation:` lines are cut, a reply in the
  wrong format gets a reminder instead of a crash, and tool errors become observations
  so the model can recover.

## Layout

```
cmd/copilot/        entrypoint: starts mcp-k8s, wires brain + hands + agent, runs one task
cmd/mcp-k8s/        standalone MCP server over stdio (works with any MCP client)
internal/tools/     MCP tool definitions: get_pods, get_logs (canned data until Day 6)
internal/llm/       Client interface + OllamaClient (net/http, streaming via NDJSON)
internal/agent/     ReAct loop, prompt, reply parser, Actor interface, MCPActor
pkg/config/         env-driven config with defaults
pkg/logger/         logrus setup (text or JSON)
docs/adr/           architecture decisions
```

## Run it

Requires Go 1.25+ and [Ollama](https://ollama.com).

```sh
ollama pull llama3.2:3b
make build
./bin/copilot "The cart service is down. Find out why."
```

The copilot starts `bin/mcp-k8s` as a child process and talks MCP to it over
stdin/stdout. `mcp-k8s` also works on its own with any MCP client.

Config comes from env vars: `LLM_BASE_URL` (default `http://localhost:11434`),
`LLM_MODEL` (default `llama3.2:3b`), `MCP_K8S_SERVER` (default `bin/mcp-k8s`),
`LOG_LEVEL`, `LOG_FORMAT`.

## Test

```sh
make test   # go test -race ./...
```

Tests need no Ollama: the LLM is replaced by a scripted fake, the Ollama client is
tested against an `httptest` server, and the MCP server is tested both in memory and
as a real child process over stdio.

## Roadmap

| Phase | Days | What |
|---|---|---|
| 0. Foundations | 1-4 | ✅ LLM client · ✅ ReAct loop · ✅ MCP tools · tracing & token accounting |
| 1. Environment | 5-6 | minikube + slim ecommerce app · read-only cluster tools |
| 2. Diagnosis | 7-12 | failure classes · structured report · eval harness (20 injected failures) |
| 3. Remediation | 13-18 | write tools · guardrails (dry-run, approval, allowlist) · remediation evals |
| 4. Polish | 19-22 | memory · planning · local vs cloud model head-to-head · demo |
