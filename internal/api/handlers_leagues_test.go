package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/fifa-watch-along/fifa-hub/internal/store"
	"github.com/fifa-watch-along/fifa-hub/internal/store/db"
)

func seedLeagueStore(t *testing.T) *store.Store {
	t.Helper()
	st, err := store.Open(filepath.Join(t.TempDir(), "leagues-test.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { st.Close() })

	ctx := t.Context()
	if _, err := st.UpsertCompetition(ctx, db.UpsertCompetitionParams{Code: "PL", Name: "Premier League", Enabled: 1}); err != nil {
		t.Fatalf("seed competition: %v", err)
	}
	teams := map[string]db.Team{}
	for _, name := range []string{"Arsenal", "Chelsea", "Liverpool", "Everton"} {
		team, err := st.GetOrCreateTeam(ctx, db.GetOrCreateTeamParams{Name: name})
		if err != nil {
			t.Fatalf("seed team %s: %v", name, err)
		}
		teams[name] = team
	}
	matches := []db.UpsertMatchParams{
		{
			CompetitionID: 1,
			Season:        "2025-26",
			UtcKickoff:    "2026-01-10T15:00:00Z",
			Status:        "FINISHED",
			HomeTeamID:    teams["Arsenal"].ID,
			AwayTeamID:    teams["Chelsea"].ID,
			HomeGoals:     sqlInt(2),
			AwayGoals:     sqlInt(1),
		},
		{
			CompetitionID: 1,
			ExternalID:    sqlNullString("fd-900"),
			Season:        "2025-26",
			UtcKickoff:    "2026-01-09T20:00:00Z",
			Status:        "FINISHED",
			HomeTeamID:    teams["Liverpool"].ID,
			AwayTeamID:    teams["Everton"].ID,
			HomeGoals:     sqlInt(3),
			AwayGoals:     sqlInt(3),
		},
		{
			CompetitionID: 1,
			Season:        "2024-25",
			UtcKickoff:    "2025-02-01T15:00:00Z",
			Status:        "FINISHED",
			HomeTeamID:    teams["Chelsea"].ID,
			AwayTeamID:    teams["Liverpool"].ID,
			HomeGoals:     sqlInt(0),
			AwayGoals:     sqlInt(4),
		},
		{
			CompetitionID: 1,
			Season:        "2025-26",
			UtcKickoff:    "2026-08-30T15:00:00Z",
			Status:        "SCHEDULED",
			HomeTeamID:    teams["Everton"].ID,
			AwayTeamID:    teams["Arsenal"].ID,
		},
	}
	for i, m := range matches {
		if _, err := st.UpsertMatch(ctx, m); err != nil {
			t.Fatalf("seed match %d: %v", i, err)
		}
	}
	return st
}

func TestLeagueStandings(t *testing.T) {
	st := seedLeagueStore(t)
	srv := httptest.NewServer(newLeagueMux(st))
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/leagues/PL/standings")
	if err != nil {
		t.Fatalf("GET standings: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var out standingsResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if out.League != "PL" {
		t.Errorf("league = %q, want PL", out.League)
	}
	if out.Season != "2025-26" {
		t.Errorf("season = %q, want 2025-26 (latest with matches)", out.Season)
	}
	wantOrder := []struct {
		name                      string
		pos, played, won, gd, pts int64
	}{
		{"Liverpool", 1, 2, 1, 4, 4},
		{"Arsenal", 2, 1, 1, 1, 3},
		{"Everton", 3, 1, 0, 0, 1},
		{"Chelsea", 4, 2, 0, -5, 0},
	}
	if len(out.Standings) != len(wantOrder) {
		t.Fatalf("standings rows = %d, want %d: %+v", len(out.Standings), len(wantOrder), out.Standings)
	}
	for i, want := range wantOrder {
		got := out.Standings[i]
		if got.Team.Name != want.name {
			t.Errorf("row %d team = %q, want %q", i, got.Team.Name, want.name)
		}
		if got.Position != want.pos || got.Played != want.played || got.Won != want.won || got.GD != want.gd || got.Points != want.pts {
			t.Errorf("row %d (%s) pos/played/won/gd/pts = %d/%d/%d/%d/%d, want %d/%d/%d/%d/%d",
				i, got.Team.Name, got.Position, got.Played, got.Won, got.GD, got.Points,
				want.pos, want.played, want.won, want.gd, want.pts)
		}
		if got.Team.ID <= 0 {
			t.Errorf("row %d missing team id", i)
		}
	}
	chelsea := out.Standings[3]
	if chelsea.GF != 1 || chelsea.GA != 6 || chelsea.Drawn != 0 || chelsea.Lost != 2 {
		t.Errorf("chelsea gf/ga/drawn/lost = %d/%d/%d/%d, want 1/6/0/2", chelsea.GF, chelsea.GA, chelsea.Drawn, chelsea.Lost)
	}
}

func TestLeagueFixturesDefaultLatestSeason(t *testing.T) {
	st := seedLeagueStore(t)
	srv := httptest.NewServer(newLeagueMux(st))
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/leagues/PL/fixtures")
	if err != nil {
		t.Fatalf("GET fixtures: %v", err)
	}
	defer resp.Body.Close()
	var out fixturesResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if out.League != "PL" || out.Season != "2025-26" {
		t.Fatalf("league/season = %s/%s, want PL/2025-26", out.League, out.Season)
	}
	if len(out.Fixtures) != 3 {
		t.Fatalf("fixtures = %d, want 3 for latest season", len(out.Fixtures))
	}
	if !(out.Fixtures[0].Kickoff < out.Fixtures[1].Kickoff && out.Fixtures[1].Kickoff < out.Fixtures[2].Kickoff) {
		t.Errorf("fixtures not ordered by kickoff asc: %+v", out.Fixtures)
	}
	first := out.Fixtures[0]
	if first.Home.Name != "Liverpool" || first.Away.Name != "Everton" {
		t.Errorf("first fixture teams = %s vs %s, want Liverpool vs Everton", first.Home.Name, first.Away.Name)
	}
	if first.External == nil || *first.External != "fd-900" {
		t.Errorf("first fixture externalId = %v, want fd-900", first.External)
	}
	if first.HomeGoals == nil || *first.HomeGoals != 3 || first.AwayGoals == nil || *first.AwayGoals != 3 {
		t.Errorf("first fixture score = %v:%v, want 3:3", first.HomeGoals, first.AwayGoals)
	}
	if first.ID <= 0 || first.Status != "FINISHED" {
		t.Errorf("first fixture id/status = %d/%q, want >0/FINISHED", first.ID, first.Status)
	}
	future := out.Fixtures[2]
	if future.Status != "SCHEDULED" || future.HomeGoals != nil || future.AwayGoals != nil {
		t.Errorf("last fixture should be unscored SCHEDULED: %+v", future)
	}
}

func TestLeagueFixturesExplicitSeason(t *testing.T) {
	st := seedLeagueStore(t)
	srv := httptest.NewServer(newLeagueMux(st))
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/leagues/PL/fixtures?season=2024-25")
	if err != nil {
		t.Fatalf("GET fixtures: %v", err)
	}
	defer resp.Body.Close()
	var out fixturesResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if out.Season != "2024-25" || len(out.Fixtures) != 1 {
		t.Fatalf("season/fixtures = %s/%d, want 2024-25/1", out.Season, len(out.Fixtures))
	}
	if out.Fixtures[0].Home.Name != "Chelsea" || out.Fixtures[0].Away.Name != "Liverpool" {
		t.Errorf("fixture teams = %s vs %s, want Chelsea vs Liverpool", out.Fixtures[0].Home.Name, out.Fixtures[0].Away.Name)
	}
}

func TestLeagueUnknownReturns404(t *testing.T) {
	st := seedLeagueStore(t)
	srv := httptest.NewServer(newLeagueMux(st))
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/leagues/UCL/standings")
	if err != nil {
		t.Fatalf("GET standings: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", resp.StatusCode)
	}
}

func newLeagueMux(st *store.Store) *http.ServeMux {
	mux := http.NewServeMux()
	Register(mux, Deps{Store: st})
	return mux
}
