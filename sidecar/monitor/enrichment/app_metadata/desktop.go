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
		// final dotted segment ("code" from "com.foo.code"), e.g. WM_CLASS "code"
		// vs org.something.code.desktop.
		baseTail := base
		if idx := strings.LastIndex(base, "."); idx >= 0 {
			baseTail = base[idx+1:]
		}
		if strings.EqualFold(entry.StartupWMClass, idLower) ||
			strings.EqualFold(base, idLower) ||
			strings.EqualFold(baseTail, idLower) {
			return true
		}
	}
	if exeBase != "" && strings.EqualFold(execBase(entry.Exec), exeBase) {
		return true
	}
	return false
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
