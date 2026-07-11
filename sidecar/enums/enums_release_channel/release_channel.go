package enumsreleasechannel

import (
	"fmt"
	"strings"
)

// ReleaseChannel values match the release_channels seed ids (db/seeds/release_channels).
type ReleaseChannel int

const (
	Unknown ReleaseChannel = iota
	Stable
	Beta
	Alpha
)

func (channel ReleaseChannel) String() string {
	switch channel {
	case Stable:
		return "Stable"
	case Beta:
		return "Beta"
	case Alpha:
		return "Alpha"
	default:
		return ""
	}
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
