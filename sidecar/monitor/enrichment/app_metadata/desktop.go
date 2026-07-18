package appmetadata

import (
	"path/filepath"
	"strings"
)

// desktopEntry holds the [Desktop Entry] fields we consume from a freedesktop
// .desktop file. Kept OS-agnostic (parsing is the same everywhere) so it is
// unit-testable off Linux.
type desktopEntry struct {
	Name           string `ini:"Name"`
	Comment        string `ini:"Comment"`
	Categories     string `ini:"Categories"`
	Icon           string `ini:"Icon"`
	Exec           string `ini:"Exec"`
	StartupWMClass string `ini:"StartupWMClass"`
}

// desktopMatches reports whether a .desktop entry (with file basename `base`)
// belongs to the app identified by idLower (lowercased WM_CLASS / app_id) or the
// executable basename exeBase.
func desktopMatches(entry desktopEntry, base string, idLower string, exeBase string) bool {
	if idLower != "" {
		if strings.EqualFold(entry.StartupWMClass, idLower) ||
			strings.EqualFold(base, idLower) ||
			// e.g. WM_CLASS "code" vs org.something.code.desktop
			strings.EqualFold(lastDotSegment(base), idLower) {
			return true
		}
	}
	if exeBase != "" && strings.EqualFold(execBase(entry.Exec), exeBase) {
		return true
	}
	return false
}

func lastDotSegment(s string) string {
	// final dotted segment ("code" from "com.foo.code").
	if idx := strings.LastIndex(s, "."); idx >= 0 {
		return s[idx+1:]
	}
	return s
}

// execBase extracts the executable basename from an Exec= line, dropping field
// codes (%U, %f) and any arguments. The first token may be double-quoted and
// contain spaces (freedesktop Exec quoting), e.g. `"/opt/My App/bin" %u`.
func execBase(exec string) string {
	exec = strings.TrimSpace(exec)
	if exec == "" {
		return ""
	}
	var first string
	if exec[0] == '"' {
		if end := strings.IndexByte(exec[1:], '"'); end >= 0 {
			first = exec[1 : 1+end]
		} else {
			first = exec[1:]
		}
	} else {
		first, _, _ = strings.Cut(exec, " ")
	}
	return filepath.Base(first)
}
