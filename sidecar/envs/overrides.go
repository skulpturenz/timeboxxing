package envs

import "strings"

// Secret overrides received out-of-band (over stdin) from the parent process. They take precedence
// over the environment-variable configuration but are never written back into the environment, so
// the secrets stay out of the process environment block.
var (
	openRouterAPIKeyOverride string
	ollamaAPIKeyOverride     string
	databaseKeyOverride      string
)

// SetSecretOverrides installs the stdin-provided secrets. Blank values leave the environment-based
// configuration in effect.
func SetSecretOverrides(openRouterAPIKey, ollamaAPIKey, databaseKey string) {
	openRouterAPIKeyOverride = strings.TrimSpace(openRouterAPIKey)
	ollamaAPIKeyOverride = strings.TrimSpace(ollamaAPIKey)
	databaseKeyOverride = strings.TrimSpace(databaseKey)
}

// ResolvedDatabaseKey returns the stdin override when present, otherwise the environment value. The
// bool reports whether a non-empty key is configured.
func ResolvedDatabaseKey() (string, bool) {
	if databaseKeyOverride != "" {
		return databaseKeyOverride, true
	}
	value, _ := DatabaseKey.Value()
	value = strings.TrimSpace(value)
	return value, value != ""
}

// ResolvedOpenRouterAPIKey returns the stdin override when present, otherwise the environment
// value. The bool reports whether a non-empty key is configured.
func ResolvedOpenRouterAPIKey() (string, bool) {
	if openRouterAPIKeyOverride != "" {
		return openRouterAPIKeyOverride, true
	}
	value, _ := OpenRouterAPIKey.Value()
	value = strings.TrimSpace(value)
	return value, value != ""
}

// ResolvedOllamaAPIKey returns the stdin override when present, otherwise the environment value.
func ResolvedOllamaAPIKey() (string, bool) {
	if ollamaAPIKeyOverride != "" {
		return ollamaAPIKeyOverride, true
	}
	value, _ := OllamaAPIKey.Value()
	value = strings.TrimSpace(value)
	return value, value != ""
}
