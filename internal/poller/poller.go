package poller

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/fifa-watch-along/fifa-hub/internal/inference"
	"github.com/fifa-watch-along/fifa-hub/internal/source"
	"github.com/fifa-watch-along/fifa-hub/internal/store"
	"github.com/fifa-watch-along/fifa-hub/internal/store/db"
)

type Deps struct {
	Store   *store.Store
	Source  source.DataSource
	Predict func([10]float64) ([3]float64, error)
	Every   time.Duration
}

type matchState struct {
	home int
	away int
}

func Run(ctx context.Context, d Deps) {
	if d.Store == nil {
		log.Println("poller: nil store, stopping")
		return
	}
	if d.Source == nil {
		log.Println("poller: no data source configured, idling")
		<-ctx.Done()
		return
	}
	if d.Every <= 0 {
		d.Every = 15 * time.Second
	}
	seen := map[string]matchState{}
	poll(ctx, d, seen)
	ticker := time.NewTicker(d.Every)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			poll(ctx, d, seen)
		}
	}
}

func poll(ctx context.Context, d Deps, seen map[string]matchState) {
	comps, err := d.Store.GetEnabledCompetitions(ctx)
	if err != nil {
		log.Printf("poller: list competitions: %v", err)
		return
	}
	for _, comp := range comps {
		code, err := providerCode(ctx, d.Store.Queries, comp.Code)
		if err != nil {
			log.Printf("poller: provider code %s: %v", comp.Code, err)
			continue
		}
		matches, err := d.Source.LiveMatches(ctx, code)
		if err != nil {
			log.Printf("poller: live matches %s (%s): %v", comp.Code, code, err)
			continue
		}
		for _, m := range matches {
			if err := applyMatch(ctx, d, seen, comp.ID, m); err != nil {
				log.Printf("poller: apply match %s: %v", m.ExternalID, err)
			}
		}
	}
}

func providerCode(ctx context.Context, q *db.Queries, compCode string) (string, error) {
	row, err := q.GetCompProviderCode(ctx, db.GetCompProviderCodeParams{
		CompCode: sql.NullString{String: compCode, Valid: true},
		Provider: "football_data",
	})
	if errors.Is(err, sql.ErrNoRows) {
		return compCode, nil
	}
	if err != nil {
		return "", err
	}
	return row, nil
}

func applyMatch(ctx context.Context, d Deps, seen map[string]matchState, competitionID int64, m source.Match) error {
	q := d.Store.Queries
	homeID, err := getOrCreateTeamID(ctx, q, m.HomeTeam)
	if err != nil {
		return fmt.Errorf("home team %q: %w", m.HomeTeam, err)
	}
	awayID, err := getOrCreateTeamID(ctx, q, m.AwayTeam)
	if err != nil {
		return fmt.Errorf("away team %q: %w", m.AwayTeam, err)
	}
	row, err := q.UpsertMatch(ctx, db.UpsertMatchParams{
		CompetitionID: competitionID,
		ExternalID:    nullString(m.ExternalID),
		Season:        seasonOf(m.UTCDate),
		UtcKickoff:    m.UTCDate,
		Status:        m.Status,
		HomeTeamID:    homeID,
		AwayTeamID:    awayID,
		HomeGoals:     nullInt64(m.HomeGoals),
		AwayGoals:     nullInt64(m.AwayGoals),
		Minute:        nullInt64(m.Minute),
	})
	if err != nil {
		return fmt.Errorf("upsert: %w", err)
	}
	cur := matchState{}
	if m.HomeGoals != nil {
		cur.home = *m.HomeGoals
	}
	if m.AwayGoals != nil {
		cur.away = *m.AwayGoals
	}
	prev, tracked := seen[m.ExternalID]
	if tracked {
		if err := recordGoalEvents(ctx, q, row.ID, m.Minute, prev, cur); err != nil {
			return fmt.Errorf("goal events: %w", err)
		}
	}
	seen[m.ExternalID] = cur
	if err := recordSnapshot(ctx, d.Predict, q, row); err != nil {
		return fmt.Errorf("win prob snapshot: %w", err)
	}
	return nil
}

func recordSnapshot(ctx context.Context, predict func([10]float64) ([3]float64, error), q *db.Queries, row db.Match) error {
	if predict == nil || row.Status != "LIVE" || !row.Minute.Valid {
		return nil
	}
	existing, err := q.ListWinProbSnapshotsByMatch(ctx, row.ID)
	if err != nil {
		return fmt.Errorf("list snapshots: %w", err)
	}
	for _, s := range existing {
		if s.Minute == row.Minute.Int64 {
			return nil
		}
	}
	feats := inference.BuildFeatures(
		float64(nullInt64Val(row.HomeGoals)),
		float64(nullInt64Val(row.AwayGoals)),
		float64(row.Minute.Int64),
		eloOf(ctx, q, row.HomeTeamID),
		eloOf(ctx, q, row.AwayTeamID),
	)
	probs, err := predict(feats)
	if err != nil {
		return fmt.Errorf("predict: %w", err)
	}
	return q.InsertWinProbSnapshot(ctx, db.InsertWinProbSnapshotParams{
		MatchID: row.ID,
		Minute:  row.Minute.Int64,
		Home:    probs[0],
		Draw:    probs[1],
		Away:    probs[2],
	})
}

func eloOf(ctx context.Context, q *db.Queries, teamID int64) float64 {
	rating, err := q.GetEloRating(ctx, teamID)
	if err != nil {
		return 0
	}
	return rating.Rating
}

func recordGoalEvents(ctx context.Context, q *db.Queries, matchID int64, minute *int, prev, cur matchState) error {
	minuteVal := int64(0)
	if minute != nil {
		minuteVal = int64(*minute)
	}
	sides := []struct {
		name string
		from int
		to   int
	}{
		{"HOME", prev.home, cur.home},
		{"AWAY", prev.away, cur.away},
	}
	for _, s := range sides {
		for i := s.from; i < s.to; i++ {
			if _, err := q.InsertMatchEvent(ctx, db.InsertMatchEventParams{
				MatchID: matchID,
				Minute:  minuteVal,
				Type:    "GOAL",
				Side:    s.name,
			}); err != nil {
				return err
			}
		}
	}
	return nil
}

func getOrCreateTeamID(ctx context.Context, q *db.Queries, name string) (int64, error) {
	team, err := q.GetOrCreateTeam(ctx, db.GetOrCreateTeamParams{Name: name})
	if err != nil {
		return 0, err
	}
	return team.ID, nil
}

func seasonOf(utcDate string) string {
	var y, mo int
	if _, err := fmt.Sscanf(utcDate, "%d-%d", &y, &mo); err != nil || y == 0 {
		return "unknown"
	}
	if mo >= 7 {
		return fmt.Sprintf("%d-%02d", y, (y+1)%100)
	}
	return fmt.Sprintf("%d-%02d", y-1, y%100)
}

func nullString(s string) sql.NullString {
	return sql.NullString{String: s, Valid: true}
}

func nullInt64(p *int) sql.NullInt64 {
	if p == nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: int64(*p), Valid: true}
}

func nullInt64Val(n sql.NullInt64) int64 {
	if !n.Valid {
		return 0
	}
	return n.Int64
}
