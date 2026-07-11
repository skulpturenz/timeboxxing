package db

import (
	"database/sql"
	"fmt"

	"github.com/mattn/go-sqlite3"
)

type ExtensionLoader = func() (path *string, entrypoint string, error error)

func registerExtensions(driverName string, loaders ...ExtensionLoader) {
	sql.Register(driverName, &sqlite3.SQLiteDriver{
		ConnectHook: func(conn *sqlite3.SQLiteConn) error {
			for _, fn := range loaders {
				path, entrypoint, err := fn()
				if err != nil {
					return fmt.Errorf("failed to load extension with entrypoint %v: %w", entrypoint, err)
				}

				if err := conn.LoadExtension(*path, entrypoint); err != nil {
					return fmt.Errorf("failed to load extension with entrypoint %v: %w", entrypoint, err)
				}
			}

			return nil
		},
	})
}
