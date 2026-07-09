package secrets

import (
	"bufio"
	"io"
	"os"
	"strings"
)

// Environment variable names used as keys in the stdin handoff payload. They match the names the
// parent process would otherwise have exported, so a dev running the sidecar standalone can still
// fall back to real environment variables (see envs.ResolvedOpenRouterAPIKey).
const (
	openRouterAPIKeyName = "SIDECAR_OPENROUTER_API_KEY"
	ollamaAPIKeyName     = "SIDECAR_OLLAMA_API_KEY"
	databaseKeyName      = "SIDECAR_DATABASE_KEY"
)

// Startup holds secrets handed to the sidecar by its parent process over stdin. Passing them this
// way keeps them out of the environment block, so they can't leak via /proc/<pid>/environ, be
// inherited by grandchild processes, or be scraped into crash/telemetry reports.
type Startup struct {
	OpenRouterAPIKey string
	OllamaAPIKey     string
	DatabaseKey      string
}

// LoadFromStdin reads the handoff payload when stdin is piped by the parent process. It is a no-op
// (returning zero secrets) when stdin is a terminal, so a manual `./timeboxxing-sidecar` run in a
// shell never blocks waiting for input.
func LoadFromStdin() Startup {
	if !isPipe(os.Stdin) {
		return Startup{}
	}
	return Parse(os.Stdin)
}

// Parse reads newline-delimited KEY=VALUE lines until EOF and extracts the known secrets. Unknown
// keys and malformed lines are ignored.
func Parse(r io.Reader) Startup {
	var startup Startup
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		key, value, ok := strings.Cut(scanner.Text(), "=")
		if !ok {
			continue
		}
		switch key {
		case openRouterAPIKeyName:
			startup.OpenRouterAPIKey = value
		case ollamaAPIKeyName:
			startup.OllamaAPIKey = value
		case databaseKeyName:
			startup.DatabaseKey = value
		}
	}
	return startup
}

func isPipe(file *os.File) bool {
	info, err := file.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice == 0
}
