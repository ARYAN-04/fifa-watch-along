package source

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

const fixtureOpenfootballJSON = `{
  "name": "Premier League 2025-26",
  "matches": [
    {
      "round": "Matchday 1",
      "date": "2025-08-15",
      "time": "20:00",
      "team1": "Liverpool FC",
      "team2": "AFC Bournemouth",
      "score": { "ft": [4, 2], "ht": [1, 0] }
    },
    {
      "round": "Matchday 1",
      "date": "2025-08-16",
      "time": "12:30",
      "team1": "Aston Villa FC",
      "team2": "Newcastle United FC",
      "score": [0, 0]
    },
    {
      "round": "Matchday 2",
      "date": "2025-08-23",
      "time": "15:00",
      "team1": "Chelsea FC",
      "team2": "Arsenal FC"
    }
  ]
}`

func TestGitHubStaticParsesBothScoreShapes(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/2025-26/en.1.json" {
			t.Errorf("unexpected path %q", r.URL.Path)
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if _, err := w.Write([]byte(fixtureOpenfootballJSON)); err != nil {
			t.Errorf("write fixture: %v", err)
		}
	}))
	defer srv.Close()

	gs := NewGitHubStatic()
	gs.BaseURL = srv.URL

	matches, err := gs.LiveMatches(t.Context(), "PL")
	if err != nil {
		t.Fatalf("LiveMatches: %v", err)
	}
	if len(matches) != 3 {
		t.Fatalf("matches = %d, want 3", len(matches))
	}

	first := matches[0]
	if first.HomeTeam != "Liverpool FC" || first.AwayTeam != "AFC Bournemouth" {
		t.Errorf("first teams = %s vs %s", first.HomeTeam, first.AwayTeam)
	}
	if first.HomeGoals == nil || *first.HomeGoals != 4 || first.AwayGoals == nil || *first.AwayGoals != 2 {
		t.Errorf("first score (object shape) = %v:%v, want 4:2", first.HomeGoals, first.AwayGoals)
	}
	if first.Status != "FINISHED" {
		t.Errorf("first status = %q, want FINISHED", first.Status)
	}
	if first.UTCDate != "2025-08-15T20:00:00Z" {
		t.Errorf("first utcDate = %q, want 2025-08-15T20:00:00Z", first.UTCDate)
	}
	wantExt := "2025-26/2025-08-15/Liverpool FC/AFC Bournemouth"
	if first.ExternalID != wantExt {
		t.Errorf("first externalId = %q, want %q", first.ExternalID, wantExt)
	}

	second := matches[1]
	if second.HomeGoals == nil || *second.HomeGoals != 0 || second.AwayGoals == nil || *second.AwayGoals != 0 {
		t.Errorf("second score (bare array shape) = %v:%v, want 0:0", second.HomeGoals, second.AwayGoals)
	}
	if second.Status != "FINISHED" {
		t.Errorf("second status = %q, want FINISHED", second.Status)
	}

	future := matches[2]
	if future.HomeGoals != nil || future.AwayGoals != nil {
		t.Errorf("future match should have no score: %+v", future)
	}
	if future.Status != "SCHEDULED" {
		t.Errorf("future status = %q, want SCHEDULED", future.Status)
	}
	if future.UTCDate != "2025-08-23T15:00:00Z" {
		t.Errorf("future utcDate = %q, want 2025-08-23T15:00:00Z", future.UTCDate)
	}
}

func TestGitHubStaticUnsupportedLeague(t *testing.T) {
	gs := NewGitHubStatic()
	if _, err := gs.LiveMatches(context.Background(), "UCL"); err == nil {
		t.Fatal("expected error for unsupported league, got nil")
	}
}

func TestGitHubStaticNonOKStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	defer srv.Close()

	gs := NewGitHubStatic()
	gs.BaseURL = srv.URL
	if _, err := gs.LiveMatches(t.Context(), "PL"); err == nil {
		t.Fatal("expected error for non-200 response, got nil")
	}
}
