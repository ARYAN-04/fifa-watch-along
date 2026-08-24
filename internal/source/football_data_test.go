package source

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

const liveFixture = `{
  "matches": [
    {
      "id": 541001,
      "utcDate": "2026-08-24T14:00:00Z",
      "status": "IN_PLAY",
      "minute": 63,
      "homeTeam": {"name": "Arsenal"},
      "awayTeam": {"name": "Chelsea"},
      "score": {"fullTime": {"home": 2, "away": 1}}
    },
    {
      "id": 541002,
      "utcDate": "2026-08-24T16:30:00Z",
      "status": "PAUSED",
      "minute": null,
      "homeTeam": {"name": "Liverpool"},
      "awayTeam": {"name": "Everton"},
      "score": {"fullTime": {"home": null, "away": null}}
    }
  ]
}`

func newTestClient(srv *httptest.Server, backoff time.Duration) *FootballData {
	return &FootballData{
		APIKey:     "test-key",
		BaseURL:    srv.URL,
		HTTPClient: srv.Client(),
		Backoff:    backoff,
	}
}

func TestLiveMatchesParsesAndNormalizes(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("X-Auth-Token")
		if gotAuth != "test-key" {
			t.Errorf("X-Auth-Token = %q, want %q", gotAuth, "test-key")
		}
		if r.URL.Query().Get("status") != "LIVE" || r.URL.Query().Get("competitions") != "PL" {
			t.Errorf("query = %v", r.URL.RawQuery)
		}
		w.Write([]byte(liveFixture))
	}))
	defer srv.Close()

	matches, err := newTestClient(srv, time.Millisecond).LiveMatches(context.Background(), "PL")
	if err != nil {
		t.Fatalf("LiveMatches: %v", err)
	}
	if len(matches) != 2 {
		t.Fatalf("len(matches) = %d, want 2", len(matches))
	}
	m := matches[0]
	if m.ExternalID != "541001" || m.HomeTeam != "Arsenal" || m.AwayTeam != "Chelsea" {
		t.Errorf("match identity wrong: %+v", m)
	}
	if m.Status != "LIVE" {
		t.Errorf("status = %q, want LIVE (normalized)", m.Status)
	}
	if m.HomeGoals == nil || *m.HomeGoals != 2 || m.AwayGoals == nil || *m.AwayGoals != 1 {
		t.Errorf("goals wrong: %+v", m)
	}
	if m.Minute == nil || *m.Minute != 63 {
		t.Errorf("minute = %v, want 63", m.Minute)
	}
	m2 := matches[1]
	if m2.Status != "LIVE" || m2.Minute != nil || m2.HomeGoals != nil || m2.AwayGoals != nil {
		t.Errorf("second match wrong: %+v", m2)
	}
}

func TestRetryOnServerError(t *testing.T) {
	hits := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		if hits < 3 {
			http.Error(w, "boom", http.StatusInternalServerError)
			return
		}
		w.Write([]byte(liveFixture))
	}))
	defer srv.Close()

	start := time.Now()
	matches, err := newTestClient(srv, 10*time.Millisecond).LiveMatches(context.Background(), "PL")
	if err != nil {
		t.Fatalf("LiveMatches: %v", err)
	}
	if len(matches) != 2 {
		t.Fatalf("len(matches) = %d, want 2", len(matches))
	}
	if hits != 3 {
		t.Errorf("hits = %d, want 3", hits)
	}
	if elapsed := time.Since(start); elapsed < 25*time.Millisecond {
		t.Errorf("elapsed %v suggests no exponential backoff between retries", elapsed)
	}
}

func TestRateLimit429HonorsRetryAfter(t *testing.T) {
	hits := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		if hits == 1 {
			w.Header().Set("Retry-After", "1")
			http.Error(w, "rate limited", http.StatusTooManyRequests)
			return
		}
		w.Write([]byte(liveFixture))
	}))
	defer srv.Close()

	start := time.Now()
	if _, err := newTestClient(srv, time.Millisecond).LiveMatches(context.Background(), "PL"); err != nil {
		t.Fatalf("LiveMatches: %v", err)
	}
	if hits != 2 {
		t.Errorf("hits = %d, want 2", hits)
	}
	if elapsed := time.Since(start); elapsed < time.Second {
		t.Errorf("elapsed %v, want >= 1s (Retry-After honored)", elapsed)
	}
}

func TestRetriesExhausted(t *testing.T) {
	hits := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		http.Error(w, "down", http.StatusInternalServerError)
	}))
	defer srv.Close()

	_, err := newTestClient(srv, time.Millisecond).LiveMatches(context.Background(), "PL")
	if err == nil {
		t.Fatal("expected error after exhausting retries")
	}
	if hits != maxAttempts {
		t.Errorf("hits = %d, want %d", hits, maxAttempts)
	}
}

func TestFatalStatusNotRetried(t *testing.T) {
	hits := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		http.Error(w, "forbidden", http.StatusForbidden)
	}))
	defer srv.Close()

	_, err := newTestClient(srv, time.Millisecond).LiveMatches(context.Background(), "PL")
	if err == nil {
		t.Fatal("expected error on 403")
	}
	if hits != 1 {
		t.Errorf("hits = %d, want 1 (non-retryable status)", hits)
	}
}

func TestNormalizeStatus(t *testing.T) {
	cases := map[string]string{
		"SCHEDULED": "SCHEDULED",
		"TIMED":     "SCHEDULED",
		"IN_PLAY":   "LIVE",
		"PAUSED":    "LIVE",
		"FINISHED":  "FINISHED",
		"AET":       "FINISHED",
		"PEN":       "FINISHED",
		"POSTPONED": "SCHEDULED",
		"UNKNOWN":   "SCHEDULED",
	}
	for in, want := range cases {
		if got := NormalizeStatus(in); got != want {
			t.Errorf("NormalizeStatus(%q) = %q, want %q", in, got, want)
		}
	}
}
