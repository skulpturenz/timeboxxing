package db

import (
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"path"
	"sort"

	"github.com/golang-migrate/migrate/v4"
	migratesqlite "github.com/golang-migrate/migrate/v4/database/sqlite3"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

//go:embed schema/*.sql seeds/*/*.sql
var migrationFiles embed.FS

func runSchemaMigrations(conn *sql.DB) error {
	if err := runMigrationsFromDir(conn, "schema", migratesqlite.DefaultMigrationsTable); err != nil {
		return fmt.Errorf("run schema migrations: %w", err)
	}

	return nil
}

func runSeedMigrations(conn *sql.DB) error {
	getSeedDirs := func() ([]string, error) {
		dir := "seeds"
		entries, err := migrationFiles.ReadDir(dir)
		if err != nil {
			return nil, err
		}

		dirs := make([]string, 0, len(entries))
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}

			dirs = append(dirs, path.Join(dir, entry.Name()))
		}

		return dirs, nil
	}

	seedDirs, err := getSeedDirs()
	if err != nil {
		return fmt.Errorf("read %s seed migration directories: %w", seedDirs, err)
	}

	// most of the seeder scripts are for reference data, there are no dependencies between
	// we add a record to `application_settings` for default settings which depends on reference data
	// run it last
	sort.Slice(seedDirs, func(i int, j int) bool {
		if path.Base(seedDirs[i]) != "application_settings" && path.Base(seedDirs[j]) == "application_settings" {
			return true
		}

		return seedDirs[i] < seedDirs[j]
	})

	for _, seedDir := range seedDirs {
		migrationTable := fmt.Sprintf("seed_migrations_%s", path.Base(seedDir))

		if err := runMigrationsFromDir(conn, seedDir, migrationTable); err != nil {
			return fmt.Errorf("run seed migrations %s: %w", seedDir, err)
		}
	}

	return nil
}

func runMigrationsFromDir(conn *sql.DB, dir string, migrationsTable string) error {
	sourceDriver, err := iofs.New(migrationFiles, dir)
	if err != nil {
		return fmt.Errorf("create migration source: %w", err)
	}

	databaseDriver, err := migratesqlite.WithInstance(conn, &migratesqlite.Config{
		MigrationsTable: migrationsTable,
		DatabaseName:    "",
		NoTxWrap:        false,
	})
	if err != nil {
		return fmt.Errorf("create migration database driver: %w", err)
	}

	migrator, err := migrate.NewWithInstance(dir, sourceDriver, "sqlite3", databaseDriver)
	if err != nil {
		return fmt.Errorf("create migrator: %w", err)
	}

	if err := migrator.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return err
	}

	return nil
}
