package postgres

import (
	"embed"
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// Migrate applies all pending schema migrations against the given database
// URI. It is safe to call repeatedly: already-applied migrations are skipped.
// The URI must use the pgx5 scheme (e.g. "pgx5://user:pass@host/db") or be
// rewritten by the caller.
func Migrate(databaseURI string) error {
	src, err := iofs.New(migrationsFS, "migrations")
	if err != nil {
		return fmt.Errorf("open migrations: %w", err)
	}

	uri := databaseURI
	if len(uri) > 11 && uri[:11] == "postgres://" {
		uri = "pgx5://" + uri[11:]
	} else if len(uri) > 13 && uri[:13] == "postgresql://" {
		uri = "pgx5://" + uri[13:]
	}

	m, err := migrate.NewWithSourceInstance("iofs", src, uri)
	if err != nil {
		return fmt.Errorf("migrate init: %w", err)
	}
	defer func() {
		_, _ = m.Close()
	}()

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("migrate up: %w", err)
	}
	return nil
}
