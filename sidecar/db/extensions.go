package db

import (
	"database/sql"
	"fmt"
	"slices"
	"sync"

	"github.com/mattn/go-sqlite3"
)

type ExtensionLoader = func() (path *string, entrypoint string, error error)

var (
	mu                sync.Mutex
	registeredDrivers []string
)

func registerExtensions(driverName string, loaders ...ExtensionLoader) {
	mu.Lock()
	defer mu.Unlock()

	if slices.Contains(registeredDrivers, driverName) {
		return
	}
	registeredDrivers = append(registeredDrivers, driverName)

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
