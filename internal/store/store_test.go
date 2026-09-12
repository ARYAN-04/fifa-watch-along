package store

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/fifa-watch-along/fifa-hub/internal/store/db"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	st, err := Open(filepath.Join(t.TempDir(), "football.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() {
		if err := st.Close(); err != nil {
			t.Fatalf("close store: %v", err)
		}
	})
	return st
}

func ns(s string) sql.NullString { return sql.NullString{String: s, Valid: true} }
func ni(i int64) sql.NullInt64   { return sql.NullInt64{Int64: i, Valid: true} }
func mustTeam(t *testing.T, st *Store, name, short, country string) db.Team {
	t.Helper()
	team, err := st.GetOrCreateTeam(context.Background(), db.GetOrCreateTeamParams{
		Name:      name,
		ShortName: ns(short),
		Country:   ns(country),
	})
	if err != nil {
		t.Fatalf("get or create team %s: %v", name, err)
	}
	return team
}

func TestOpenMigratesAndReopens(t *testing.T) {
	path := filepath.Join(t.TempDir(), "football.db")
	for i := 0; i < 2; i++ {
		st, err := Open(path)
		if err != nil {
			t.Fatalf("open iteration %d: %v", i, err)
		}
		if err := st.Close(); err != nil {
			t.Fatalf("close iteration %d: %v", i, err)
		}
	}
}

func TestCompetitionUpsertAndGet(t *testing.T) {
	ctx := context.Background()
	st := newTestStore(t)

	comp, err := st.UpsertCompetition(ctx, db.UpsertCompetitionParams{
		Code: "PL", Name: "Premier League", Country: ns("England"), Enabled: 1,
	})
	if err != nil {
		t.Fatalf("upsert competition: %v", err)
	}
	got, err := st.GetCompetitionByCode(ctx, "PL")
	if err != nil {
		t.Fatalf("get competition: %v", err)
	}
	if got.ID != comp.ID || got.Name != "Premier League" || got.Enabled != 1 {
		t.Fatalf("unexpected competition: %+v", got)
	}

	enabled, err := st.GetEnabledCompetitions(ctx)
	if err != nil {
		t.Fatalf("get enabled competitions: %v", err)
	}
	if len(enabled) != 1 {
		t.Fatalf("expected 1 enabled competition, got %d", len(enabled))
	}
}

func TestTeamCrosswalkElo(t *testing.T) {
	ctx := context.Background()
	st := newTestStore(t)

	team := mustTeam(t, st, "Arsenal", "ARS", "England")
	again := mustTeam(t, st, "Arsenal", "ARS", "England")
	if again.ID != team.ID {
		t.Fatalf("expected same team id %d, got %d", team.ID, again.ID)
	}
	fetched, err := st.GetTeamByID(ctx, team.ID)
	if err != nil {
		t.Fatalf("get team by id: %v", err)
	}
	if fetched.Name != "Arsenal" {
		t.Fatalf("unexpected team: %+v", fetched)
	}

	cw, err := st.UpsertCrosswalk(ctx, db.UpsertCrosswalkParams{
		TeamID: ni(team.ID), Provider: "football_data", ProviderID: "42",
	})
	if err != nil {
		t.Fatalf("upsert crosswalk: %v", err)
	}
	if !cw.TeamID.Valid || cw.TeamID.Int64 != team.ID {
		t.Fatalf("unexpected crosswalk: %+v", cw)
	}
	rows, err := st.GetTeamCrosswalk(ctx, ni(team.ID))
	if err != nil {
		t.Fatalf("get crosswalk: %v", err)
	}
	if len(rows) != 1 || rows[0].Provider != "football_data" || rows[0].ProviderID != "42" {
		t.Fatalf("unexpected crosswalk rows: %+v", rows)
	}

	if _, err := st.UpsertCrosswalk(ctx, db.UpsertCrosswalkParams{
		CompCode: ns("PL"), Provider: "reep_id", ProviderID: "comp-eng.1",
	}); err != nil {
		t.Fatalf("upsert competition crosswalk row: %v", err)
	}
	teamRows, err := st.GetTeamCrosswalk(ctx, ni(team.ID))
	if err != nil {
		t.Fatalf("get crosswalk after comp row: %v", err)
	}
	if len(teamRows) != 1 {
		t.Fatalf("competition crosswalk leaked into team query: %+v", teamRows)
	}

	if err := st.UpsertEloRating(ctx, db.UpsertEloRatingParams{
		TeamID: team.ID, Rating: 1789.5, UpdatedAt: "2026-08-24T00:00:00Z",
	}); err != nil {
		t.Fatalf("upsert elo: %v", err)
	}
	rating, err := st.GetEloRating(ctx, team.ID)
	if err != nil {
		t.Fatalf("get elo: %v", err)
	}
	if rating.Rating != 1789.5 {
		t.Fatalf("expected elo 1789.5, got %f", rating.Rating)
	}
}

func seedMatch(t *testing.T, ctx context.Context, st *Store, compID int64, ext string, status string, home, away db.Team, hg, ag, minute int64) db.Match {
	t.Helper()
	m, err := st.UpsertMatch(ctx, db.UpsertMatchParams{
		CompetitionID: compID,
		ExternalID:    ns(ext),
		Season:        "2026",
		UtcKickoff:    "2026-08-24T15:00:00Z",
		Status:        status,
		HomeTeamID:    home.ID,
		AwayTeamID:    away.ID,
		HomeGoals:     ni(hg),
		AwayGoals:     ni(ag),
		Minute:        ni(minute),
	})
	if err != nil {
		t.Fatalf("upsert match %s: %v", ext, err)
	}
	return m
}

func TestMatchLifecycle(t *testing.T) {
	ctx := context.Background()
	st := newTestStore(t)

	comp, err := st.UpsertCompetition(ctx, db.UpsertCompetitionParams{
		Code: "PL", Name: "Premier League", Country: ns("England"), Enabled: 1,
	})
	if err != nil {
		t.Fatalf("upsert competition: %v", err)
	}
	home := mustTeam(t, st, "Arsenal", "ARS", "England")
	away := mustTeam(t, st, "Chelsea", "CHE", "England")

	m := seedMatch(t, ctx, st, comp.ID, "fd:1001", "LIVE", home, away, 1, 0, 67)
	live, err := st.GetLiveMatches(ctx)
	if err != nil {
		t.Fatalf("get live matches: %v", err)
	}
	if len(live) != 1 || live[0].ID != m.ID || live[0].Status != "LIVE" {
		t.Fatalf("unexpected live matches: %+v", live)
	}

	if _, err := st.UpsertMatch(ctx, db.UpsertMatchParams{
		CompetitionID: comp.ID,
		ExternalID:    ns("fd:1001"),
		Season:        "2026",
		UtcKickoff:    "2026-08-24T15:00:00Z",
		Status:        "FINISHED",
		HomeTeamID:    home.ID,
		AwayTeamID:    away.ID,
		HomeGoals:     ni(2),
		AwayGoals:     ni(1),
		Minute:        ni(90),
	}); err != nil {
		t.Fatalf("upsert match update: %v", err)
	}
	live, err = st.GetLiveMatches(ctx)
	if err != nil {
		t.Fatalf("get live matches after finish: %v", err)
	}
	if len(live) != 0 {
		t.Fatalf("expected no live matches after finish, got %d", len(live))
	}
	updated, err := st.GetMatchByID(ctx, m.ID)
	if err != nil {
		t.Fatalf("get match by id: %v", err)
	}
	if updated.Status != "FINISHED" || updated.HomeGoals.Int64 != 2 || updated.AwayGoals.Int64 != 1 {
		t.Fatalf("unexpected updated match: %+v", updated)
	}

	if err := st.UpdateMatchScore(ctx, db.UpdateMatchScoreParams{
		HomeGoals: ni(3), AwayGoals: ni(1), Minute: ni(90), ID: m.ID,
	}); err != nil {
		t.Fatalf("update match score: %v", err)
	}
	scored, err := st.GetMatchByID(ctx, m.ID)
	if err != nil {
		t.Fatalf("get match by id after score update: %v", err)
	}
	if scored.HomeGoals.Int64 != 3 {
		t.Fatalf("expected home goals 3, got %d", scored.HomeGoals.Int64)
	}

	events := []db.MatchEvent{
		{Minute: 23, Type: "GOAL", Side: "away"},
		{Minute: 12, Type: "GOAL", Side: "home"},
	}
	for _, e := range events {
		id, err := st.InsertMatchEvent(ctx, db.InsertMatchEventParams{
			MatchID: m.ID, Minute: e.Minute, Type: e.Type, Side: e.Side,
			Player: ns("Bukayo Saka"), Detail: ns("left foot"),
		})
		if err != nil {
			t.Fatalf("insert event: %v", err)
		}
		if id == 0 {
			t.Fatal("expected nonzero event id")
		}
	}
	gotEvents, err := st.GetMatchEvents(ctx, m.ID)
	if err != nil {
		t.Fatalf("get events: %v", err)
	}
	if len(gotEvents) != 2 || gotEvents[0].Minute != 12 || gotEvents[1].Minute != 23 {
		t.Fatalf("events not ordered by minute: %+v", gotEvents)
	}
}

func TestWinProbSnapshots(t *testing.T) {
	ctx := context.Background()
	st := newTestStore(t)

	comp, err := st.UpsertCompetition(ctx, db.UpsertCompetitionParams{
		Code: "PL", Name: "Premier League", Country: ns("England"), Enabled: 1,
	})
	if err != nil {
		t.Fatalf("upsert competition: %v", err)
	}
	m := seedMatch(t, ctx, st, comp.ID, "fd:2002", "LIVE",
		mustTeam(t, st, "Liverpool", "LIV", "England"),
		mustTeam(t, st, "Everton", "EVE", "England"), 1, 0, 30)

	snapshots := []struct {
		minute           int64
		home, draw, away float64
	}{
		{10, 0.45, 0.17, 0.38},
		{20, 0.52, 0.20, 0.28},
		{20, 0.55, 0.21, 0.24},
	}
	for _, s := range snapshots {
		if err := st.InsertWinProbSnapshot(ctx, db.InsertWinProbSnapshotParams{
			MatchID: m.ID, Minute: s.minute, Home: s.home, Draw: s.draw, Away: s.away,
		}); err != nil {
			t.Fatalf("insert snapshot at %d: %v", s.minute, err)
		}
	}
	got, err := st.ListWinProbSnapshotsByMatch(ctx, m.ID)
	if err != nil {
		t.Fatalf("list snapshots: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 snapshots after minute-20 upsert, got %d", len(got))
	}
	last := got[len(got)-1]
	if last.Minute != 20 || last.Home != 0.55 || last.Draw != 0.21 || last.Away != 0.24 {
		t.Fatalf("unexpected last snapshot: %+v", last)
	}
}

func TestFixturesQueries(t *testing.T) {
	ctx := context.Background()
	st := newTestStore(t)

	comp, err := st.UpsertCompetition(ctx, db.UpsertCompetitionParams{
		Code: "PL", Name: "Premier League", Country: ns("England"), Enabled: 1,
	})
	if err != nil {
		t.Fatalf("upsert competition: %v", err)
	}
	other, err := st.UpsertCompetition(ctx, db.UpsertCompetitionParams{
		Code: "UCL", Name: "Champions League", Country: ns("Europe"), Enabled: 0,
	})
	if err != nil {
		t.Fatalf("upsert other competition: %v", err)
	}
	a := mustTeam(t, st, "Arsenal", "ARS", "England")
	b := mustTeam(t, st, "Chelsea", "CHE", "England")

	seedMatch(t, ctx, st, comp.ID, "fd:3001", "FINISHED", a, b, 2, 0, 90)
	seedMatch(t, ctx, st, comp.ID, "fd:3002", "SCHEDULED", b, a, 0, 0, 0)
	seedMatch(t, ctx, st, other.ID, "fd:3003", "FINISHED", a, b, 1, 1, 120)

	fixtures, err := st.ListFixturesByLeagueAndSeason(ctx, db.ListFixturesByLeagueAndSeasonParams{
		CompetitionID: comp.ID, Season: "2026",
	})
	if err != nil {
		t.Fatalf("list fixtures: %v", err)
	}
	if len(fixtures) != 2 {
		t.Fatalf("expected 2 fixtures for PL/2026, got %d", len(fixtures))
	}
	if fixtures[0].HomeTeamName != "Arsenal" || fixtures[1].HomeTeamName != "Chelsea" {
		t.Fatalf("fixtures not ordered or joined names wrong: %+v", fixtures)
	}

	finished, err := st.ListFinishedByLeagueAndSeason(ctx, db.ListFinishedByLeagueAndSeasonParams{
		CompetitionID: comp.ID, Season: "2026",
	})
	if err != nil {
		t.Fatalf("list finished: %v", err)
	}
	if len(finished) != 1 || finished[0].ExternalID.String != "fd:3001" {
		t.Fatalf("unexpected finished list: %+v", finished)
	}
}

func TestStandings(t *testing.T) {
	ctx := context.Background()
	st := newTestStore(t)

	comp, err := st.UpsertCompetition(ctx, db.UpsertCompetitionParams{
		Code: "PL", Name: "Premier League", Country: ns("England"), Enabled: 1,
	})
	if err != nil {
		t.Fatalf("upsert competition: %v", err)
	}
	other, err := st.UpsertCompetition(ctx, db.UpsertCompetitionParams{
		Code: "UCL", Name: "Champions League", Country: ns("Europe"), Enabled: 0,
	})
	if err != nil {
		t.Fatalf("upsert other competition: %v", err)
	}
	a := mustTeam(t, st, "Arsenal", "ARS", "England")
	b := mustTeam(t, st, "Chelsea", "CHE", "England")
	c := mustTeam(t, st, "Liverpool", "LIV", "England")

	seedMatch(t, ctx, st, comp.ID, "fd:4001", "FINISHED", a, b, 2, 0, 90)
	seedMatch(t, ctx, st, comp.ID, "fd:4002", "FINISHED", a, c, 1, 1, 90)
	seedMatch(t, ctx, st, comp.ID, "fd:4003", "FINISHED", b, c, 3, 1, 90)
	seedMatch(t, ctx, st, other.ID, "fd:4004", "FINISHED", a, b, 5, 5, 120)

	table, err := st.GetStandingsByCompetition(ctx, comp.ID)
	if err != nil {
		t.Fatalf("get standings: %v", err)
	}
	expected := []db.GetStandingsByCompetitionRow{
		{TeamName: "Arsenal", Played: 2, Won: 1, Drawn: 1, Lost: 0, GoalsFor: 3, GoalsAgainst: 1, GoalDiff: 2, Points: 4},
		{TeamName: "Chelsea", Played: 2, Won: 1, Drawn: 0, Lost: 1, GoalsFor: 3, GoalsAgainst: 3, GoalDiff: 0, Points: 3},
		{TeamName: "Liverpool", Played: 2, Won: 0, Drawn: 1, Lost: 1, GoalsFor: 2, GoalsAgainst: 4, GoalDiff: -2, Points: 1},
	}
	if len(table) != len(expected) {
		t.Fatalf("expected %d standings rows, got %d: %+v", len(expected), len(table), table)
	}
	for i, want := range expected {
		got := table[i]
		if got.TeamName != want.TeamName || got.Played != want.Played || got.Won != want.Won ||
			got.Drawn != want.Drawn || got.Lost != want.Lost || got.GoalsFor != want.GoalsFor ||
			got.GoalsAgainst != want.GoalsAgainst || got.GoalDiff != want.GoalDiff || got.Points != want.Points {
			t.Fatalf("row %d: expected %+v, got %+v", i, want, got)
		}
	}
}
