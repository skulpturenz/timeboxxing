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
	Linux
)

func (os OperatingSystem) String() string {
	return []string{"unknown", "macos", "windows", "linux"}[os]
}

func Parse(s string) (OperatingSystem, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "darwin":
		return MacOS, nil
	case "windows":
		return Windows, nil
	case "linux":
		return Linux, nil
	}

	return Unknown, fmt.Errorf("unrecognized operating system: %s", s)
}
