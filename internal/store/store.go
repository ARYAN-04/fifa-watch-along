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

// WinProbSnapshotExists reports whether a snapshot already exists for the
// match at the given minute, so the poller can skip redundant writes.
func (s *Store) WinProbSnapshotExists(ctx context.Context, matchID, minute int64) (bool, error) {
	var exists bool
	err := s.conn.QueryRowContext(ctx,
		`SELECT EXISTS(SELECT 1 FROM win_prob_snapshots WHERE match_id = ? AND minute = ?)`,
		matchID, minute).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("snapshot exists check: %w", err)
	}
	return exists, nil
}

// ReplayableMatch is a match with a stored probability timeline.
// Replay is historical, so this ignores the competitions.enabled flag.
type ReplayableMatch struct {
	ID        int64
	League    string
	Season    string
	Kickoff   string
	Status    string
	HomeID    int64
	HomeName  string
	AwayID    int64
	AwayName  string
	HomeGoals sql.NullInt64
	AwayGoals sql.NullInt64
	Snapshots int64
}

// ListReplayableMatches returns matches that have win-probability snapshots,
// newest kickoff first.
func (s *Store) ListReplayableMatches(ctx context.Context) ([]ReplayableMatch, error) {
	rows, err := s.conn.QueryContext(ctx, `
		SELECT m.id, comp.code, m.season, m.utc_kickoff, m.status,
			m.home_team_id, ht.name, m.away_team_id, at.name,
			m.home_goals, m.away_goals, COUNT(s.id)
		FROM matches m
		JOIN competitions comp ON comp.id = m.competition_id
		JOIN teams ht ON ht.id = m.home_team_id
		JOIN teams at ON at.id = m.away_team_id
		JOIN win_prob_snapshots s ON s.match_id = m.id
		GROUP BY m.id
		ORDER BY m.utc_kickoff DESC`)
	if err != nil {
		return nil, fmt.Errorf("list replayable matches: %w", err)
	}
	defer rows.Close()
	var out []ReplayableMatch
	for rows.Next() {
		var m ReplayableMatch
		if err := rows.Scan(&m.ID, &m.League, &m.Season, &m.Kickoff, &m.Status,
			&m.HomeID, &m.HomeName, &m.AwayID, &m.AwayName,
			&m.HomeGoals, &m.AwayGoals, &m.Snapshots); err != nil {
			return nil, fmt.Errorf("scan replayable match: %w", err)
		}
		out = append(out, m)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}
