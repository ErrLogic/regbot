package db

import (
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

// RunMigrations applies all pending SQL migrations from the given directory.
func RunMigrations(dsn, migrationsDir string) error {
	m, err := migrate.New("file://"+migrationsDir, dsn)
	if err != nil {
		return fmt.Errorf("db: migrate init: %w", err)
	}
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("db: migrate up: %w", err)
	}
	return nil
}
