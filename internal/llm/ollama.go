package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// OllamaClient calls a local Ollama server over plain net/http.
type OllamaClient struct {
	baseURL string
	http    *http.Client
}

// Compile-time check that OllamaClient satisfies Client.
var _ Client = (*OllamaClient)(nil)

// NewOllamaClient returns a client for an Ollama server such as http://localhost:11434.
func NewOllamaClient(baseURL string, httpClient *http.Client) *OllamaClient {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &OllamaClient{baseURL: strings.TrimRight(baseURL, "/"), http: httpClient}
}

// Ollama's wire format for POST /api/chat.
type ollamaMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ollamaRequest struct {
	Model    string          `json:"model"`
	Messages []ollamaMessage `json:"messages"`
	Stream   bool            `json:"stream"`
	Options  map[string]any  `json:"options,omitempty"`
}

type ollamaResponse struct {
	Message ollamaMessage `json:"message"`
	Done    bool          `json:"done"`
	Error   string        `json:"error,omitempty"`
}

// toOllamaMessages is the only place our Message type meets Ollama's JSON.
func toOllamaMessages(msgs []Message) []ollamaMessage {
	out := make([]ollamaMessage, len(msgs))
	for i, m := range msgs {
		out[i] = ollamaMessage{Role: string(m.Role), Content: m.Content}
	}
	return out
}

// Complete sends stream:false and reads back one JSON object.
func (c *OllamaClient) Complete(ctx context.Context, req CompletionRequest) (string, error) {
	resp, err := c.post(ctx, req, false)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var out ollamaResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", fmt.Errorf("ollama: decode response: %w", err)
	}
	if out.Error != "" {
		return "", fmt.Errorf("ollama: %s", out.Error)
	}
	return out.Message.Content, nil
}

// Stream sends stream:true and reads newline-delimited JSON, one chunk per line.
func (c *OllamaClient) Stream(ctx context.Context, req CompletionRequest, onToken func(string)) (string, error) {
	resp, err := c.post(ctx, req, true)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var full strings.Builder
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}
		var chunk ollamaResponse
		if err := json.Unmarshal(line, &chunk); err != nil {
			return full.String(), fmt.Errorf("ollama: decode chunk: %w", err)
		}
		if chunk.Error != "" {
			return full.String(), fmt.Errorf("ollama: %s", chunk.Error)
		}
		if tok := chunk.Message.Content; tok != "" {
			full.WriteString(tok)
			if onToken != nil {
				onToken(tok)
			}
		}
		if chunk.Done {
			break
		}
	}
	if err := scanner.Err(); err != nil {
		return full.String(), fmt.Errorf("ollama: read stream: %w", err)
	}
	return full.String(), nil
}

// post builds and sends the HTTP request. A non-200 status is an error.
func (c *OllamaClient) post(ctx context.Context, req CompletionRequest, stream bool) (*http.Response, error) {
	body := ollamaRequest{
		Model:    req.Model,
		Messages: toOllamaMessages(req.Messages),
		Stream:   stream,
		Options:  map[string]any{"temperature": req.Temperature},
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("ollama: encode request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/chat", bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("ollama: build request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("ollama: call /api/chat: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		msg, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("ollama: status %d: %s", resp.StatusCode, strings.TrimSpace(string(msg)))
	}
	return resp, nil
}
