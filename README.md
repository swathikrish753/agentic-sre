# agentic-sre

An autonomous SRE agent written in Go. It watches a Kubernetes cluster, diagnoses
failures (CrashLoopBackOff, ImagePullBackOff, OOMKilled, Pending), and proposes fixes
that a human approves before they run. Every run is traced, and an eval harness
measures how well it performs.

> Status: in progress. Days 1-2 of 22 are done (LLM client + ReAct loop).

## How it works

```
cmd/copilot ──▶ Agent.Run (ReAct loop) ──▶ llm.Client ──▶ Ollama (local) / cloud model
                     │
                     ├──▶ Actor ──▶ FakeActor today → MCP tools → Kubernetes
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
- **`Actor`**: canned fake tools today, real MCP tools next.

### Safety built into the loop

- Four exits: final answer, max steps, context timeout, LLM error.
- Model output is not trusted: invented `Observation:` lines are cut, a reply in the
  wrong format gets a reminder instead of a crash, and tool errors become observations
  so the model can recover.

## Layout

```
cmd/copilot/        entrypoint: wires brain + hands + agent, runs one task
internal/llm/       Client interface + OllamaClient (net/http, streaming via NDJSON)
internal/agent/     ReAct loop, prompt, reply parser, Actor interface
pkg/config/         env-driven config with defaults
pkg/logger/         logrus setup (text or JSON)
docs/adr/           architecture decisions
```

## Run it

Requires Go 1.25+ and [Ollama](https://ollama.com).

```sh
ollama pull llama3.2:3b
go run ./cmd/copilot "The cart service is down. Find out why."
```

Config comes from env vars: `LLM_BASE_URL` (default `http://localhost:11434`),
`LLM_MODEL` (default `llama3.2:3b`), `LOG_LEVEL`, `LOG_FORMAT`.

## Test

```sh
make test   # go test -race ./...
```

Tests need no Ollama: the LLM is replaced by a scripted fake, and the Ollama client is
tested against an `httptest` server.

## Roadmap

| Phase | Days | What |
|---|---|---|
| 0. Foundations | 1-4 | ✅ LLM client · ✅ ReAct loop · MCP tools · tracing & token accounting |
| 1. Environment | 5-6 | minikube + slim ecommerce app · read-only cluster tools |
| 2. Diagnosis | 7-12 | failure classes · structured report · eval harness (20 injected failures) |
| 3. Remediation | 13-18 | write tools · guardrails (dry-run, approval, allowlist) · remediation evals |
| 4. Polish | 19-22 | memory · planning · local vs cloud model head-to-head · demo |
