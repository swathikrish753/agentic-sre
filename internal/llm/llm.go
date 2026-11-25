// Package llm defines how the agent talks to a language model. Nothing outside
// this package knows which provider (Ollama, a cloud model, a test fake) is used.
package llm

import (
	"context"
	"time"
)

// Role says who wrote a message in the conversation.
type Role string

const (
	RoleSystem    Role = "system"
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
)

// Message is one turn of the conversation.
type Message struct {
	Role    Role
	Content string
}

// CompletionRequest is everything needed for one model call.
type CompletionRequest struct {
	Model       string
	Messages    []Message
	Temperature float64
}

// Usage is what one model call cost: tokens in, tokens out, and time spent.
type Usage struct {
	PromptTokens     int           // tokens the model read (the whole history)
	CompletionTokens int           // tokens the model wrote
	TotalDuration    time.Duration // server-side time for the whole call
	LoadDuration     time.Duration // time spent loading the model into memory
}

// Response is a model's answer plus its usage.
type Response struct {
	Text  string
	Usage Usage
}

// Client is the agent's phone line to a model.
type Client interface {
	// Complete blocks until the whole answer is ready.
	Complete(ctx context.Context, req CompletionRequest) (Response, error)
	// Stream pushes the answer piece by piece to onToken and returns the full response.
	Stream(ctx context.Context, req CompletionRequest, onToken func(string)) (Response, error)
}
