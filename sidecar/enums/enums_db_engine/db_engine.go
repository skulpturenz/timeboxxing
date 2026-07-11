package enumsdbengine

import "fmt"

type DbEngine int

const (
	Sqlite DbEngine = iota
)

func (env DbEngine) String() string {
	return []string{"sqlite"}[env]
}

func Parse(s string) (DbEngine, error) {
	switch s {
	case "sqlite":
		return Sqlite, nil
	}

	return Sqlite, fmt.Errorf("unrecognized db type: %s", s)
}
