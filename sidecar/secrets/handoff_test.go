package secrets

import (
	"strings"
	"testing"
)

func TestParseReadsKnownSecrets(t *testing.T) {
	payload := "SIDECAR_OPENROUTER_API_KEY=sk-or-v1-abc\nSIDECAR_OLLAMA_API_KEY=ollama-token\n"

	got := Parse(strings.NewReader(payload))

	if got.OpenRouterAPIKey != "sk-or-v1-abc" {
		t.Errorf("OpenRouterAPIKey = %q, want %q", got.OpenRouterAPIKey, "sk-or-v1-abc")
	}
	if got.OllamaAPIKey != "ollama-token" {
		t.Errorf("OllamaAPIKey = %q, want %q", got.OllamaAPIKey, "ollama-token")
	}
}

func TestParseIgnoresUnknownAndMalformedLines(t *testing.T) {
	payload := "UNKNOWN=whatever\nno-equals-sign\nSIDECAR_OPENROUTER_API_KEY=only-this\n"

	got := Parse(strings.NewReader(payload))

	if got.OpenRouterAPIKey != "only-this" {
		t.Errorf("OpenRouterAPIKey = %q, want %q", got.OpenRouterAPIKey, "only-this")
	}
	if got.OllamaAPIKey != "" {
		t.Errorf("OllamaAPIKey = %q, want empty", got.OllamaAPIKey)
	}
}

func TestParseHandlesEmptyInput(t *testing.T) {
	got := Parse(strings.NewReader(""))

	if got != (Startup{}) {
		t.Errorf("Parse(empty) = %+v, want zero value", got)
	}
}

func TestParsePreservesValuesContainingEquals(t *testing.T) {
	// strings.Cut splits on the first '=', so a value that itself contains '=' is preserved.
	got := Parse(strings.NewReader("SIDECAR_OPENROUTER_API_KEY=a=b=c\n"))

	if got.OpenRouterAPIKey != "a=b=c" {
		t.Errorf("OpenRouterAPIKey = %q, want %q", got.OpenRouterAPIKey, "a=b=c")
	}
}
