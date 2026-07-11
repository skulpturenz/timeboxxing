package enumsjournalmode

import (
	"fmt"
	"strings"
)

type JournalMode int

const (
	Delete JournalMode = iota
	Truncate
	Persist
	Memory
	WAL
	Off
)

func (env JournalMode) String() string {
	return []string{"DELETE", "TRUNCATE", "PERSIST", "MEMORY", "WAL", "OFF"}[env]
}

func Parse(s string) (JournalMode, error) {
	switch strings.ToLower(s) {
	case "delete":
		return Delete, nil
	case "truncate":
		return Truncate, nil
	case "persist":
		return Persist, nil
	case "memory":
		return Memory, nil
	case "wal":
		return WAL, nil
	}

	return WAL, fmt.Errorf("unrecognized journal mode: %s", s)
}
