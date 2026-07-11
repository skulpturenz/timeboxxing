package enumsoperatingsystem

import (
	"fmt"
	"strings"
)

type OperatingSystem int

const (
	Unknown OperatingSystem = iota
	MacOS
	Windows
)

func Parse(s string) (OperatingSystem, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "macos":
		return MacOS, nil
	case "windows":
		return Windows, nil
	}

	return Unknown, fmt.Errorf("unrecognized operating system: %s", s)
}

func (os OperatingSystem) String() string {
	switch os {
	case MacOS:
		return "macos"
	case Windows:
		return "windows"
	default:
		return ""
	}
}
