package agent

import (
	"context"
	"fmt"
	"sort"
)

// Actor is the agent's hands: it runs a named tool with an input and returns what it saw.
type Actor interface {
	ToolNames() []string
	Act(ctx context.Context, tool, input string) (string, error)
}

// FakeActor returns canned cluster data so the loop can run without Kubernetes.
type FakeActor struct{}

var _ Actor = FakeActor{}

var fakeTools = map[string]func(input string) (string, error){
	"get_pods": func(string) (string, error) {
		return "NAME                       READY   STATUS             RESTARTS\n" +
			"cart-7d9f8b6c5-x2k4p       0/1     CrashLoopBackOff   6\n" +
			"frontend-5c8d7f9b4-q9w8e   1/1     Running            0", nil
	},
	"get_logs": func(pod string) (string, error) {
		if pod == "cart-7d9f8b6c5-x2k4p" {
			return "panic: failed to connect to redis at redis:6379: connection refused", nil
		}
		return "", fmt.Errorf("pod %q not found", pod)
	},
}

func (FakeActor) ToolNames() []string {
	names := make([]string, 0, len(fakeTools))
	for name := range fakeTools {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func (FakeActor) Act(_ context.Context, tool, input string) (string, error) {
	fn, ok := fakeTools[tool]
	if !ok {
		return "", fmt.Errorf("unknown tool %q", tool)
	}
	return fn(input)
}
