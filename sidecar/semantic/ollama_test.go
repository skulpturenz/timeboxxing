package semantic

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestOllamaGeneratorToolCall(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req map[string]any
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if _, ok := req["tools"]; !ok {
			t.Fatalf("expected tools in request: %+v", req)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"message":{"content":"","tool_calls":[{"function":{"name":"get_app_usage_totals","arguments":{"started_at":"2026-06-13T00:00:00Z"}}}]}}`))
	}))
	defer server.Close()

	generator, err := NewOllamaGenerator(OllamaConfig{
		BaseURL: server.URL,
		Model:   "gemma-model",
		Client:  server.Client(),
	})
	if err != nil {
		t.Fatalf("create generator: %v", err)
	}

	response, err := generator.GenerateWithTools(context.Background(), []ChatMessage{{Role: ChatRoleUser, Content: "question"}}, []ChatTool{{
		Name:        "get_app_usage_totals",
		Description: "usage",
		Parameters:  map[string]any{"type": "object"},
	}})
	if err != nil {
		t.Fatalf("generate with tools: %v", err)
	}
	if len(response.ToolCalls) != 1 {
		t.Fatalf("expected one tool call, got %#v", response.ToolCalls)
	}
	if response.ToolCalls[0].Name != "get_app_usage_totals" {
		t.Fatalf("unexpected tool call %#v", response.ToolCalls[0])
	}
	if !strings.Contains(response.ToolCalls[0].Arguments, "started_at") {
		t.Fatalf("unexpected tool arguments %q", response.ToolCalls[0].Arguments)
	}
}
