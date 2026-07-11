package db

import "errors"

var (
	ErrDatabaseKeyMismatch = errors.New("database is encrypted but the provided key does not match")
)
