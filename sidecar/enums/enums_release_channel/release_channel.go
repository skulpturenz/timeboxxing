package enumsreleasechannel

import (
	"fmt"
	"strings"
)

type ReleaseChannel int

const (
	Unknown ReleaseChannel = iota
	Stable
	Beta
	Alpha
)

func (channel ReleaseChannel) String() string {
	return []string{"unknown", "stable", "beta", "alpha"}[channel]
}

func Parse(s string) (ReleaseChannel, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "stable":
		return Stable, nil
	case "beta":
		return Beta, nil
	case "alpha":
		return Alpha, nil
	}

	return Unknown, fmt.Errorf("unrecognized release channel: %s", s)
}
