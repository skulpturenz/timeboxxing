package appmetadata

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

// desktopEntry holds the [Desktop Entry] fields we consume from a freedesktop
// .desktop file. Kept OS-agnostic (parsing is the same everywhere) so it is
// unit-testable off Linux.
type desktopEntry struct {
	Name           string
	Comment        string
	Categories     string
	Icon           string
	Exec           string
	StartupWMClass string
}

// parseDesktopFile reads the [Desktop Entry] group of a .desktop file. Localized
// keys (Name[de]) are ignored in favor of the unlocalized value.
func parseDesktopFile(path string) (desktopEntry, error) {
	f, err := os.Open(path)
	if err != nil {
		return desktopEntry{}, err
	}
	defer f.Close()

	var entry desktopEntry
	inGroup := false
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			inGroup = line == "[Desktop Entry]"
			continue
		}
		if !inGroup {
			continue
		}
		key, value, found := strings.Cut(line, "=")
		if !found {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		switch key {
		case "Name":
			entry.Name = value
		case "Comment":
			entry.Comment = value
		case "Categories":
			entry.Categories = value
		case "Icon":
			entry.Icon = value
		case "Exec":
			entry.Exec = value
		case "StartupWMClass":
			entry.StartupWMClass = value
		}
	}
	if err := scanner.Err(); err != nil {
		return desktopEntry{}, err
	}
	return entry, nil
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

// lastDotSegment returns the final dotted segment ("code" from "com.foo.code").
func lastDotSegment(s string) string {
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
