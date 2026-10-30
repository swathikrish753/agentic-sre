package llm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOllamaComplete(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req ollamaRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if req.Stream || req.Model != "llama3.2:3b" || len(req.Messages) != 1 {
			t.Fatalf("unexpected request: %+v", req)
		}
		w.Write([]byte(`{"message":{"role":"assistant","content":"pong"},"done":true}`))
	}))
	defer srv.Close()

	c := NewOllamaClient(srv.URL, nil)
	got, err := c.Complete(context.Background(), CompletionRequest{
		Model:    "llama3.2:3b",
		Messages: []Message{{Role: RoleUser, Content: "ping"}},
	})
	if err != nil || got != "pong" {
		t.Fatalf("got %q, %v", got, err)
	}
}

func TestOllamaStream(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"message":{"content":"Hel"},"done":false}` + "\n" +
			`{"message":{"content":"lo"},"done":false}` + "\n" +
			`{"message":{"content":""},"done":true}` + "\n"))
	}))
	defer srv.Close()

	var tokens []string
	got, err := NewOllamaClient(srv.URL, nil).Stream(context.Background(), CompletionRequest{},
		func(tok string) { tokens = append(tokens, tok) })
	if err != nil || got != "Hello" || len(tokens) != 2 {
		t.Fatalf("got %q, tokens %v, err %v", got, tokens, err)
	}
}

func TestOllamaNon200(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "model not found", http.StatusNotFound)
	}))
	defer srv.Close()

	if _, err := NewOllamaClient(srv.URL, nil).Complete(context.Background(), CompletionRequest{}); err == nil {
		t.Fatal("expected an error for status 404")
	}
}
