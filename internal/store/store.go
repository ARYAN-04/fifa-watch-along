package store

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"sort"

	"github.com/fifa-watch-along/fifa-hub/internal/store/db"

	_ "modernc.org/sqlite"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

type Store struct {
	*db.Queries
	conn *sql.DB
}

func Open(path string) (*Store, error) {
	conn, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite %q: %w", path, err)
	}
	if err := conn.Ping(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("ping sqlite %q: %w", path, err)
	}
	if err := migrate(context.Background(), conn); err != nil {
		conn.Close()
		return nil, fmt.Errorf("migrate sqlite %q: %w", path, err)
	}
	return &Store{Queries: db.New(conn), conn: conn}, nil
}

func migrate(ctx context.Context, conn *sql.DB) error {
	entries, err := fs.ReadDir(migrationsFS, "migrations")
	if err != nil {
		return fmt.Errorf("read migrations dir: %w", err)
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			names = append(names, entry.Name())
		}
	}
	sort.Strings(names)
	for _, name := range names {
		content, err := migrationsFS.ReadFile("migrations/" + name)
		if err != nil {
			return fmt.Errorf("read migration %s: %w", name, err)
		}
		if _, err := conn.ExecContext(ctx, string(content)); err != nil {
			return fmt.Errorf("apply migration %s: %w", name, err)
		}
	}
	return nil
}

func (s *Store) Close() error {
	if s == nil || s.conn == nil {
		return nil
	}
	if err := s.conn.Close(); err != nil {
		return fmt.Errorf("close sqlite: %w", err)
	}
	return nil
}
