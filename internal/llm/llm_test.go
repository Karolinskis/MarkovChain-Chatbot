package llm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestChat(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/chat" {
			t.Errorf("path = %q, want /api/chat", r.URL.Path)
		}
		var req chatRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatal(err)
		}
		if req.Model != "test-model" || req.Stream || req.Think {
			t.Errorf("request = %+v, want model test-model with stream and think off", req)
		}
		if len(req.Messages) != 1 || req.Messages[0].Content != "hi" {
			t.Errorf("messages = %+v", req.Messages)
		}
		json.NewEncoder(w).Encode(chatResponse{Message: Message{Role: "assistant", Content: "hello"}})
	}))
	defer srv.Close()

	got, err := New(srv.URL, "test-model").Chat(context.Background(), []Message{{Role: "user", Content: "hi"}})
	if err != nil {
		t.Fatalf("Chat() = %v", err)
	}
	if got != "hello" {
		t.Errorf("Chat() = %q, want %q", got, "hello")
	}
}

func TestChatError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(chatResponse{Error: `model "missing" not found`})
	}))
	defer srv.Close()

	if _, err := New(srv.URL, "missing").Chat(context.Background(), nil); err == nil {
		t.Fatal("Chat() on error status should return an error")
	}
}
