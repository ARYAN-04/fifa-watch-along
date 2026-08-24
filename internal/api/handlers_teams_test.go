package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/fifa-watch-along/fifa-hub/internal/store"
	"github.com/fifa-watch-along/fifa-hub/internal/store/db"
)

func seedCompareStore(t *testing.T) (*store.Store, map[string]db.Team) {
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
	teams := make(map[string]db.Team)
	for _, name := range []string{"Arsenal FC", "Chelsea FC", "Liverpool FC", "Everton FC"} {
		team, err := st.GetOrCreateTeam(ctx, db.GetOrCreateTeamParams{Name: name})
		if err != nil {
			t.Fatalf("seed team %s: %v", name, err)
		}
		teams[name] = team
	}

	matches := []struct {
		ext        string
		kickoff    string
		home, away db.Team
		hg, ag     int64
	}{
		{"900001", "2026-08-01T14:00:00Z", teams["Arsenal FC"], teams["Chelsea FC"], 2, 1},
		{"900002", "2026-08-05T14:00:00Z", teams["Chelsea FC"], teams["Arsenal FC"], 0, 0},
		{"900003", "2026-08-10T14:00:00Z", teams["Arsenal FC"], teams["Chelsea FC"], 3, 2},
		{"900004", "2026-08-12T14:00:00Z", teams["Arsenal FC"], teams["Liverpool FC"], 1, 4},
		{"900005", "2026-08-13T14:00:00Z", teams["Chelsea FC"], teams["Everton FC"], 2, 2},
	}
	for _, m := range matches {
		if _, err := st.UpsertMatch(ctx, db.UpsertMatchParams{
			CompetitionID: 1,
			ExternalID:    sqlNullString(m.ext),
			Season:        "2026-27",
			UtcKickoff:    m.kickoff,
			Status:        "FINISHED",
			HomeTeamID:    m.home.ID,
			AwayTeamID:    m.away.ID,
			HomeGoals:     sqlInt(m.hg),
			AwayGoals:     sqlInt(m.ag),
			Minute:        sqlInt(90),
		}); err != nil {
			t.Fatalf("seed match %s: %v", m.ext, err)
		}
	}
	if err := st.UpsertEloRating(ctx, db.UpsertEloRatingParams{TeamID: teams["Arsenal FC"].ID, Rating: 1750.5, UpdatedAt: "2026-08-24T00:00:00Z"}); err != nil {
		t.Fatalf("seed elo: %v", err)
	}
	return st, teams
}

func getCompare(t *testing.T, mux *http.ServeMux, query string) (int, compareResponse) {
	t.Helper()
	srv := httptest.NewServer(mux)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/teams/compare" + query)
	if err != nil {
		t.Fatalf("GET /api/teams/compare: %v", err)
	}
	defer resp.Body.Close()
	var out compareResponse
	if resp.StatusCode == http.StatusOK {
		if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
			t.Fatalf("decode response: %v", err)
		}
	}
	return resp.StatusCode, out
}

func TestTeamCompareHandler(t *testing.T) {
	st, teams := seedCompareStore(t)
	mux := http.NewServeMux()
	Register(mux, Deps{Store: st})

	status, out := getCompare(t, mux,
		"?home="+itoa(teams["Arsenal FC"].ID)+"&away="+itoa(teams["Chelsea FC"].ID))
	if status != http.StatusOK {
		t.Fatalf("status = %d, want 200", status)
	}

	if out.Home.Name != "Arsenal FC" || out.Home.Elo == nil || *out.Home.Elo != 1750.5 {
		t.Errorf("home = %+v elo %v, want Arsenal FC with elo 1750.5", out.Home, out.Home.Elo)
	}
	if out.Away.Name != "Chelsea FC" || out.Away.Elo != nil {
		t.Errorf("away = %+v elo %v, want Chelsea FC with null elo", out.Away, out.Away.Elo)
	}

	wantForm := []formEntry{
		{"L", "Liverpool FC", "2026-08-12T14:00:00Z", "1-4"},
		{"W", "Chelsea FC", "2026-08-10T14:00:00Z", "3-2"},
		{"D", "Chelsea FC", "2026-08-05T14:00:00Z", "0-0"},
		{"W", "Chelsea FC", "2026-08-01T14:00:00Z", "2-1"},
	}
	if len(out.Home.Form) != len(wantForm) {
		t.Fatalf("home form entries = %d, want %d", len(out.Home.Form), len(wantForm))
	}
	for i, want := range wantForm {
		if out.Home.Form[i] != want {
			t.Errorf("home form[%d] = %+v, want %+v", i, out.Home.Form[i], want)
		}
	}

	chelseaWant := []string{"D", "L", "D", "L"}
	for i, want := range chelseaWant {
		if out.Away.Form[i].Result != want {
			t.Errorf("away form[%d].result = %q, want %q", i, out.Away.Form[i].Result, want)
		}
	}

	if out.H2H.Played != 3 || out.H2H.HomeWins != 2 || out.H2H.Draws != 1 || out.H2H.AwayWins != 0 {
		t.Errorf("h2h record = %+v, want played 3 homeWins 2 draws 1 awayWins 0", out.H2H)
	}
	if out.H2H.AvgGoals != 2.7 {
		t.Errorf("avgGoals = %v, want 2.7", out.H2H.AvgGoals)
	}
	if len(out.H2H.Matches) != 3 {
		t.Fatalf("h2h matches = %d, want 3", len(out.H2H.Matches))
	}
	newest := out.H2H.Matches[0]
	if newest.Date != "2026-08-10T14:00:00Z" || newest.HomeGoals == nil || *newest.HomeGoals != 3 || newest.AwayGoals == nil || *newest.AwayGoals != 2 {
		t.Errorf("newest h2h match = %+v, want Aug 10 3-2", newest)
	}
}

func TestTeamCompareTruncatesToLast10(t *testing.T) {
	st, teams := seedCompareStore(t)
	ctx := t.Context()
	for i := 0; i < 12; i++ {
		ext := "9100" + string(rune('a'+i))
		kickoff := "2027-01-" + itoa(int64(10+i%18)) + "T12:00:00Z"
		var hg, ag int64 = int64(i), int64(i % 2)
		if _, err := st.UpsertMatch(ctx, db.UpsertMatchParams{
			CompetitionID: 1,
			ExternalID:    sqlNullString(ext),
			Season:        "2026-27",
			UtcKickoff:    kickoff,
			Status:        "FINISHED",
			HomeTeamID:    teams["Everton FC"].ID,
			AwayTeamID:    teams["Liverpool FC"].ID,
			HomeGoals:     sqlInt(hg),
			AwayGoals:     sqlInt(ag),
		}); err != nil {
			t.Fatalf("seed extra match %d: %v", i, err)
		}
	}
	mux := http.NewServeMux()
	Register(mux, Deps{Store: st})

	status, out := getCompare(t, mux,
		"?home="+itoa(teams["Everton FC"].ID)+"&away="+itoa(teams["Liverpool FC"].ID))
	if status != http.StatusOK {
		t.Fatalf("status = %d, want 200", status)
	}
	if out.H2H.Played != 12 {
		t.Errorf("played = %d, want 12", out.H2H.Played)
	}
	if len(out.H2H.Matches) != 10 {
		t.Errorf("returned matches = %d, want 10", len(out.H2H.Matches))
	}
}

func TestTeamCompareUnknownTeam404(t *testing.T) {
	st, teams := seedCompareStore(t)
	mux := http.NewServeMux()
	Register(mux, Deps{Store: st})

	for _, query := range []string{
		"?home=999&away=" + itoa(teams["Chelsea FC"].ID),
		"?home=" + itoa(teams["Arsenal FC"].ID) + "&away=999",
	} {
		status, _ := getCompare(t, mux, query)
		if status != http.StatusNotFound {
			t.Errorf("status for %s = %d, want 404", query, status)
		}
	}
}

func TestTeamCompareInvalidIDs400(t *testing.T) {
	st, _ := seedCompareStore(t)
	mux := http.NewServeMux()
	Register(mux, Deps{Store: st})

	status, _ := getCompare(t, mux, "?home=abc&away=1")
	if status != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", status)
	}
}

func itoa(v int64) string { return strconv.FormatInt(v, 10) }
