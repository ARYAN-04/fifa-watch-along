package api

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/fifa-watch-along/fifa-hub/internal/store"
	"github.com/fifa-watch-along/fifa-hub/internal/store/db"
)

func seedLiveMatchStore(t *testing.T) *store.Store {
	t.Helper()
	st, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { st.Close() })

	ctx := t.Context()
	if _, err := st.UpsertCompetition(ctx, db.UpsertCompetitionParams{Code: "PL", Name: "Premier League", Enabled: 1}); err != nil {
		t.Fatalf("seed competition: %v", err)
	}
	home, err := st.GetOrCreateTeam(ctx, db.GetOrCreateTeamParams{Name: "Arsenal"})
	if err != nil {
		t.Fatalf("seed home team: %v", err)
	}
	away, err := st.GetOrCreateTeam(ctx, db.GetOrCreateTeamParams{Name: "Chelsea"})
	if err != nil {
		t.Fatalf("seed away team: %v", err)
	}
	live := db.UpsertMatchParams{
		CompetitionID: 1,
		ExternalID:    sqlNullString("541001"),
		Season:        "2026-27",
		UtcKickoff:    "2026-08-24T14:00:00Z",
		Status:        "LIVE",
		HomeTeamID:    home.ID,
		AwayTeamID:    away.ID,
		HomeGoals:     sqlInt(2),
		AwayGoals:     sqlInt(1),
		Minute:        sqlInt(63),
	}
	if _, err := st.UpsertMatch(ctx, live); err != nil {
		t.Fatalf("seed live match: %v", err)
	}

	liv2, err := st.GetOrCreateTeam(ctx, db.GetOrCreateTeamParams{Name: "Liverpool"})
	if err != nil {
		t.Fatalf("seed team 3: %v", err)
	}
	everton, err := st.GetOrCreateTeam(ctx, db.GetOrCreateTeamParams{Name: "Everton"})
	if err != nil {
		t.Fatalf("seed team 4: %v", err)
	}
	done := db.UpsertMatchParams{
		CompetitionID: 1,
		ExternalID:    sqlNullString("541002"),
		Season:        "2026-27",
		UtcKickoff:    "2026-08-23T14:00:00Z",
		Status:        "FINISHED",
		HomeTeamID:    liv2.ID,
		AwayTeamID:    everton.ID,
	}
	if _, err := st.UpsertMatch(ctx, done); err != nil {
		t.Fatalf("seed finished match: %v", err)
	}
	return st
}

func sqlNullString(s string) sql.NullString { return sql.NullString{String: s, Valid: true} }
func sqlInt(v int64) sql.NullInt64          { return sql.NullInt64{Int64: v, Valid: true} }

func getLiveScores(t *testing.T, mux *http.ServeMux) liveScoresResponse {
	t.Helper()
	srv := httptest.NewServer(mux)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/scores/live")
	if err != nil {
		t.Fatalf("GET /api/scores/live: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var out liveScoresResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return out
}

func TestLiveScoresHandler(t *testing.T) {
	st := seedLiveMatchStore(t)
	mux := http.NewServeMux()
	Register(mux, Deps{Store: st})

	out := getLiveScores(t, mux)
	if len(out.Matches) != 1 {
		t.Fatalf("matches = %d, want 1 (finished match excluded)", len(out.Matches))
	}
	m := out.Matches[0]
	if m.External == nil || *m.External != "541001" {
		t.Errorf("externalId = %v, want 541001", m.External)
	}
	if m.Home.Name != "Arsenal" || m.Away.Name != "Chelsea" {
		t.Errorf("teams = %s vs %s, want Arsenal vs Chelsea", m.Home.Name, m.Away.Name)
	}
	if m.HomeGoals == nil || *m.HomeGoals != 2 {
		t.Errorf("homeGoals = %v, want 2", m.HomeGoals)
	}
	if m.AwayGoals == nil || *m.AwayGoals != 1 {
		t.Errorf("awayGoals = %v, want 1", m.AwayGoals)
	}
	if m.Minute == nil || *m.Minute != 63 {
		t.Errorf("minute = %v, want 63", m.Minute)
	}
	if m.Status != "LIVE" {
		t.Errorf("status = %q, want LIVE", m.Status)
	}
	if m.ID <= 0 {
		t.Errorf("id = %d, want > 0", m.ID)
	}
}

func TestMocksModeServesCannedWithoutDB(t *testing.T) {
	mux := http.NewServeMux()
	Register(mux, Deps{Mocks: true})

	out := getLiveScores(t, mux)
	if len(out.Matches) != 2 {
		t.Fatalf("matches = %d, want 2 canned matches", len(out.Matches))
	}
	inPlay := out.Matches[0]
	if inPlay.Status != "LIVE" || inPlay.Minute == nil || inPlay.HomeGoals == nil {
		t.Errorf("first canned match should be in-play with score and minute: %+v", inPlay)
	}
	kickoff := out.Matches[1]
	if kickoff.Status != "LIVE" || kickoff.HomeGoals != nil || kickoff.AwayGoals != nil {
		t.Errorf("second canned match should be goalless at kickoff: %+v", kickoff)
	}
	for i, m := range out.Matches {
		if m.External == nil || m.Home.Name == "" || m.Away.Name == "" {
			t.Errorf("canned match %d missing required fields: %+v", i, m)
		}
	}
}
