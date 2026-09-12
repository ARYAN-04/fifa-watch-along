package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fifa-watch-along/fifa-hub/internal/store/db"
)

func newMockServer(t *testing.T) string {
	t.Helper()
	mux := http.NewServeMux()
	Register(mux, Deps{Mocks: true})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv.URL
}

func getJSON(t *testing.T, url string, dst any) int {
	t.Helper()
	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("GET %s: %v", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
		if err := json.NewDecoder(resp.Body).Decode(dst); err != nil {
			t.Fatalf("decode %s: %v", url, err)
		}
	}
	return resp.StatusCode
}

func TestMockStandingsAndFixtures(t *testing.T) {
	base := newMockServer(t)

	var st standingsResponse
	if code := getJSON(t, base+"/api/leagues/pl/standings", &st); code != http.StatusOK {
		t.Fatalf("standings status = %d, want 200", code)
	}
	if len(st.Standings) != 4 {
		t.Fatalf("standings rows = %d, want 4", len(st.Standings))
	}
	if st.Standings[0].Team.ID <= 0 {
		t.Errorf("standings rows need team ids for the compare page")
	}

	var fx fixturesResponse
	if code := getJSON(t, base+"/api/leagues/pl/fixtures", &fx); code != http.StatusOK {
		t.Fatalf("fixtures status = %d, want 200", code)
	}
	if fx.Season != "2025-26" || len(fx.Fixtures) != 3 {
		t.Fatalf("season/fixtures = %s/%d, want 2025-26/3", fx.Season, len(fx.Fixtures))
	}
}

func TestMockMatchDetailRoundTrip(t *testing.T) {
	base := newMockServer(t)

	var detail matchDetail
	if code := getJSON(t, base+"/api/matches/900001", &detail); code != http.StatusOK {
		t.Fatalf("match status = %d, want 200", code)
	}
	if detail.Home.Name != "Arsenal" || detail.Away.Name != "Chelsea" {
		t.Errorf("match teams = %s vs %s, want Arsenal vs Chelsea", detail.Home.Name, detail.Away.Name)
	}

	var wc2026 matchDetail
	if code := getJSON(t, base+"/api/matches/1903", &wc2026); code != http.StatusOK {
		t.Fatalf("wc2026 match status = %d, want 200", code)
	}
	if wc2026.Home.Name != "France" || wc2026.Away.Name != "England" {
		t.Errorf("match teams = %s vs %s, want France vs England", wc2026.Home.Name, wc2026.Away.Name)
	}
	if wc2026.HomeGoals == nil || *wc2026.HomeGoals != 4 || wc2026.AwayGoals == nil || *wc2026.AwayGoals != 6 {
		t.Errorf("score = %v-%v, want 4-6", wc2026.HomeGoals, wc2026.AwayGoals)
	}

	var events struct {
		Events []matchEvent `json:"events"`
	}
	if code := getJSON(t, base+"/api/matches/900001/events", &events); code != http.StatusOK {
		t.Fatalf("events status = %d, want 200", code)
	}
	if len(events.Events) != 3 {
		t.Errorf("events = %d, want 3", len(events.Events))
	}

	var probs winProbabilityResponse
	if code := getJSON(t, base+"/api/matches/900001/win-probability", &probs); code != http.StatusOK {
		t.Fatalf("win-probability status = %d, want 200", code)
	}
	if probs.PreMatch == nil || len(probs.Snapshots) != 9 {
		t.Errorf("want preMatch + 9 snapshots, got %+v", probs)
	}

	for _, path := range []string{"/api/matches/999/events", "/api/matches/999/win-probability", "/api/matches/999"} {
		resp, err := http.Get(base + path)
		if err != nil {
			t.Fatalf("GET %s: %v", path, err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("GET %s status = %d, want 404", path, resp.StatusCode)
		}
	}
	resp, err := http.Get(base + "/api/matches/abc")
	if err != nil {
		t.Fatalf("GET invalid id: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("invalid id status = %d, want 400", resp.StatusCode)
	}
}

func TestMockCompare(t *testing.T) {
	base := newMockServer(t)

	var out compareResponse
	if code := getJSON(t, base+"/api/teams/compare?home=1&away=2", &out); code != http.StatusOK {
		t.Fatalf("compare status = %d, want 200", code)
	}
	if out.Home.ID != 1 || out.Away.ID != 2 {
		t.Errorf("compare echoes ids: home=%d away=%d, want 1/2", out.Home.ID, out.Away.ID)
	}
	if out.H2H.Played == 0 || len(out.H2H.Matches) == 0 {
		t.Errorf("compare needs h2h data, got %+v", out.H2H)
	}

	resp, err := http.Get(base + "/api/teams/compare?home=x&away=2")
	if err != nil {
		t.Fatalf("GET invalid compare: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("invalid compare status = %d, want 400", resp.StatusCode)
	}
}

func TestLeagueLookupCaseInsensitive(t *testing.T) {
	st := seedLeagueStore(t)
	srv := httptest.NewServer(newLeagueMux(st))
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/leagues/pl/standings")
	if err != nil {
		t.Fatalf("GET lowercase standings: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("lowercase league status = %d, want 200 (frontend sends 'pl', DB stores 'PL')", resp.StatusCode)
	}
}

func TestMockLeagueScoping(t *testing.T) {
	base := newMockServer(t)

	var pl standingsResponse
	if code := getJSON(t, base+"/api/leagues/pl/standings", &pl); code != http.StatusOK {
		t.Fatalf("pl standings status = %d, want 200", code)
	}
	if pl.League != "PL" || len(pl.Standings) != 4 || pl.Standings[0].Team.Name != "Arsenal" {
		t.Errorf("pl standings wrong: %+v", pl)
	}

	var wc standingsResponse
	if code := getJSON(t, base+"/api/leagues/wc2026/standings", &wc); code != http.StatusOK {
		t.Fatalf("wc standings status = %d, want 200", code)
	}
	if wc.League != "WC2026" || len(wc.Standings) != 4 || wc.Standings[0].Team.Name != "Spain" {
		t.Errorf("wc standings wrong: %+v", wc)
	}
	if wc.Standings[0].Points != 6 {
		t.Errorf("spain points = %d, want 6", wc.Standings[0].Points)
	}

	var ucl standingsResponse
	if code := getJSON(t, base+"/api/leagues/ucl/standings", &ucl); code != http.StatusOK {
		t.Fatalf("ucl standings status = %d, want 200", code)
	}
	if len(ucl.Standings) != 0 {
		t.Errorf("ucl standings = %d rows, want empty (no data)", len(ucl.Standings))
	}

	var wcfx fixturesResponse
	if code := getJSON(t, base+"/api/leagues/wc2026/fixtures", &wcfx); code != http.StatusOK {
		t.Fatalf("wc fixtures status = %d, want 200", code)
	}
	if wcfx.Season != "2026" || len(wcfx.Fixtures) != 4 {
		t.Fatalf("wc fixtures season/count = %s/%d, want 2026/4", wcfx.Season, len(wcfx.Fixtures))
	}
	if wcfx.Fixtures[2].ID != 1903 {
		t.Errorf("wc fixture ids wrong: %+v", wcfx.Fixtures)
	}

	var uclfx fixturesResponse
	if code := getJSON(t, base+"/api/leagues/ucl/fixtures", &uclfx); code != http.StatusOK {
		t.Fatalf("ucl fixtures status = %d, want 200", code)
	}
	if len(uclfx.Fixtures) != 0 {
		t.Errorf("ucl fixtures = %d rows, want empty (no data)", len(uclfx.Fixtures))
	}
}

func TestMockReplayMatches(t *testing.T) {
	base := newMockServer(t)

	var out replayMatchesResponse
	if code := getJSON(t, base+"/api/replay/matches", &out); code != http.StatusOK {
		t.Fatalf("replay status = %d, want 200", code)
	}
	if len(out.Matches) != 5 {
		t.Fatalf("replay matches = %d, want 5 (900001 + 4 WC2026)", len(out.Matches))
	}
	if out.Matches[0].Snapshots <= 0 || out.Matches[0].Home.ID <= 0 {
		t.Errorf("replay rows need snapshot counts and team ids: %+v", out.Matches[0])
	}
}

func TestReplayMatchesListsSnapshotMatches(t *testing.T) {
	st := seedLeagueStore(t)
	ctx := t.Context()
	arsenal, err := st.GetOrCreateTeam(ctx, db.GetOrCreateTeamParams{Name: "Arsenal"})
	if err != nil {
		t.Fatalf("get team: %v", err)
	}
	chelsea, err := st.GetOrCreateTeam(ctx, db.GetOrCreateTeamParams{Name: "Chelsea"})
	if err != nil {
		t.Fatalf("get team: %v", err)
	}
	match, err := st.UpsertMatch(ctx, db.UpsertMatchParams{
		CompetitionID: 1,
		Season:        "2025-26",
		UtcKickoff:    "2026-01-10T15:00:00Z",
		Status:        "FINISHED",
		HomeTeamID:    arsenal.ID,
		AwayTeamID:    chelsea.ID,
		HomeGoals:     sqlInt(2),
		AwayGoals:     sqlInt(1),
	})
	if err != nil {
		t.Fatalf("seed match: %v", err)
	}
	for _, minute := range []int64{10, 20} {
		if err := st.InsertWinProbSnapshot(ctx, db.InsertWinProbSnapshotParams{
			MatchID: match.ID, Minute: minute, Home: 0.6, Draw: 0.25, Away: 0.15,
		}); err != nil {
			t.Fatalf("seed snapshot: %v", err)
		}
	}
	srv := httptest.NewServer(newLeagueMux(st))
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/replay/matches")
	if err != nil {
		t.Fatalf("GET replay: %v", err)
	}
	defer resp.Body.Close()
	var out replayMatchesResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode replay: %v", err)
	}
	if len(out.Matches) != 1 {
		t.Fatalf("replay matches = %d, want 1 (only the snapshotted match)", len(out.Matches))
	}
	got := out.Matches[0]
	if got.ID != match.ID || got.Snapshots != 2 || got.League != "PL" {
		t.Errorf("replay row = %+v, want id %d with 2 snapshots in PL", got, match.ID)
	}
	if got.Home.Name != "Arsenal" || got.Away.Name != "Chelsea" {
		t.Errorf("replay teams = %s vs %s, want Arsenal vs Chelsea", got.Home.Name, got.Away.Name)
	}
}
