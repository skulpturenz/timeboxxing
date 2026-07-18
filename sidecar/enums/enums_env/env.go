package enumsenv

import (
	"fmt"
	"strings"
)

type Environment int

const (
	Production Environment = iota
	Development
	Test
	Local
)

func (env Environment) String() string {
	return []string{"production", "development", "test", "local"}[env]
}

func Parse(s string) (Environment, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "production":
		return Production, nil
	case "development":
		return Development, nil
	case "test":
		return Test, nil
	case "local":
		return Local, nil
	}

	return Development, fmt.Errorf("unrecognized env: %s", s)
}
