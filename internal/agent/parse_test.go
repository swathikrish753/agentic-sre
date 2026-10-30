package agent

import (
	"errors"
	"testing"
)

func TestParseReply(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		want    Step
		wantErr error
	}{
		{
			name: "action",
			in:   "Thought: list pods first\nAction: get_pods\nAction Input: default",
			want: Step{Thought: "list pods first", Action: "get_pods", ActionInput: "default"},
		},
		{
			name: "final answer",
			in:   "Thought: I know\nFinal Answer: cart cannot reach redis",
			want: Step{Thought: "I know", FinalAnswer: "cart cannot reach redis"},
		},
		{
			name: "multi-line final answer",
			in:   "Final Answer: root cause: redis down\nfix: restart redis",
			want: Step{FinalAnswer: "root cause: redis down\nfix: restart redis"},
		},
		{
			name: "invented observation is cut",
			in:   "Action: get_pods\nAction Input: \nObservation: everything is fine\nFinal Answer: all good",
			want: Step{Action: "get_pods"},
		},
		{
			name: "action wins over final answer",
			in:   "Action: get_logs\nAction Input: cart\nFinal Answer: guessing",
			want: Step{Action: "get_logs", ActionInput: "cart"},
		},
		{
			name:    "no format",
			in:      "I think the cart pod is broken.",
			wantErr: ErrBadFormat,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseReply(tt.in)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("err = %v, want %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Fatalf("got %+v, want %+v", got, tt.want)
			}
		})
	}
}
