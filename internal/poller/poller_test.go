package poller

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/fifa-watch-along/fifa-hub/internal/source"
	"github.com/fifa-watch-along/fifa-hub/internal/store"
	"github.com/fifa-watch-along/fifa-hub/internal/store/db"
)

type fakeSource struct {
	snapshots [][]source.Match
	calls     int
	err       error
}

func (f *fakeSource) LiveMatches(ctx context.Context, leagueCode string) ([]source.Match, error) {
	f.calls++
	if f.err != nil {
		return nil, f.err
	}
	idx := f.calls - 1
	if idx >= len(f.snapshots) {
		idx = len(f.snapshots) - 1
	}
	return f.snapshots[idx], nil
}

func testMatch(extID, home, away string, homeGoals, awayGoals, minute *int) source.Match {
	return source.Match{
		ExternalID: extID,
		HomeTeam:   home,
		AwayTeam:   away,
		HomeGoals:  homeGoals,
		AwayGoals:  awayGoals,
		Minute:     minute,
		Status:     "LIVE",
		UTCDate:    "2026-08-24T14:00:00Z",
	}
}

func intPtr(v int) *int { return &v }

func openTestStore(t *testing.T) *store.Store {
	t.Helper()
	st, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { st.Close() })

	_, err = st.UpsertCompetition(context.Background(), db.UpsertCompetitionParams{
		Code: "PL", Name: "Premier League", Enabled: 1,
	})
	if err != nil {
		t.Fatalf("seed competition: %v", err)
	}
	return st
}

func TestPollerUpsertsAndDiffsGoalEvents(t *testing.T) {
	ctx := context.Background()
	st := openTestStore(t)
	src := &fakeSource{
		snapshots: [][]source.Match{
			{testMatch("541001", "Arsenal", "Chelsea", intPtr(0), intPtr(0), intPtr(1))},
			{testMatch("541001", "Arsenal", "Chelsea", intPtr(1), intPtr(0), intPtr(20))},
			{testMatch("541001", "Arsenal", "Chelsea", intPtr(1), intPtr(2), intPtr(75))},
		},
	}
	d := Deps{Store: st, Source: src, Every: time.Second}
	seen := map[string]matchState{}

	poll(ctx, d, seen)

	live, err := st.GetLiveMatches(ctx)
	if err != nil {
		t.Fatalf("GetLiveMatches: %v", err)
	}
	if len(live) != 1 {
		t.Fatalf("live matches = %d, want 1", len(live))
	}
	matchID := live[0].ID

	events, err := st.GetMatchEvents(ctx, matchID)
	if err != nil {
		t.Fatalf("GetMatchEvents: %v", err)
	}
	if len(events) != 0 {
		t.Fatalf("events after first snapshot = %d, want 0", len(events))
	}

	poll(ctx, d, seen)

	live, err = st.GetLiveMatches(ctx)
	if err != nil {
		t.Fatalf("GetLiveMatches: %v", err)
	}
	if !live[0].HomeGoals.Valid || live[0].HomeGoals.Int64 != 1 {
		t.Errorf("home goals = %+v, want 1", live[0].HomeGoals)
	}
	if !live[0].Minute.Valid || live[0].Minute.Int64 != 20 {
		t.Errorf("minute = %+v, want 20", live[0].Minute)
	}
	events, err = st.GetMatchEvents(ctx, matchID)
	if err != nil {
		t.Fatalf("GetMatchEvents: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("goal events = %d, want 1", len(events))
	}
	ev := events[0]
	if ev.Type != "GOAL" || ev.Side != "HOME" || ev.Minute != 20 {
		t.Errorf("event = type %q side %q minute %d, want GOAL/HOME/20", ev.Type, ev.Side, ev.Minute)
	}

	poll(ctx, d, seen)

	events, err = st.GetMatchEvents(ctx, matchID)
	if err != nil {
		t.Fatalf("GetMatchEvents: %v", err)
	}
	if len(events) != 3 {
		t.Fatalf("total events = %d, want 3 (1 HOME + 2 AWAY)", len(events))
	}
	var awayGoals int
	for _, e := range events {
		if e.Side == "AWAY" && e.Minute == 75 {
			awayGoals++
		}
	}
	if awayGoals != 2 {
		t.Errorf("AWAY goal events at minute 75 = %d, want 2", awayGoals)
	}
}

func TestPollerNewMatchMidLoopGetsNoSpuriousEvents(t *testing.T) {
	ctx := context.Background()
	st := openTestStore(t)
	src := &fakeSource{
		snapshots: [][]source.Match{
			nil,
			{testMatch("541002", "Liverpool", "Everton", intPtr(1), intPtr(0), intPtr(5))},
		},
	}
	d := Deps{Store: st, Source: src, Every: time.Second}
	seen := map[string]matchState{}

	poll(ctx, d, seen)
	poll(ctx, d, seen)

	live, err := st.GetLiveMatches(ctx)
	if err != nil {
		t.Fatalf("GetLiveMatches: %v", err)
	}
	if len(live) != 1 {
		t.Fatalf("live matches = %d, want 1", len(live))
	}
	events, err := st.GetMatchEvents(ctx, live[0].ID)
	if err != nil {
		t.Fatalf("GetMatchEvents: %v", err)
	}
	if len(events) != 0 {
		t.Errorf("events for newly tracked match = %d, want 0", len(events))
	}
}

func TestPollerSourceErrorDoesNotCrashLoop(t *testing.T) {
	ctx := context.Background()
	st := openTestStore(t)
	src := &fakeSource{err: errors.New("api down")}
	d := Deps{Store: st, Source: src, Every: time.Second}

	poll(ctx, d, map[string]matchState{})

	live, err := st.GetLiveMatches(ctx)
	if err != nil {
		t.Fatalf("GetLiveMatches: %v", err)
	}
	if len(live) != 0 {
		t.Errorf("live matches = %d, want 0", len(live))
	}
}

func TestSeasonOf(t *testing.T) {
	cases := []struct{ date, want string }{
		{"2026-08-24T14:00:00Z", "2026-27"},
		{"2025-08-16T12:00:00Z", "2025-26"},
		{"2026-05-15T12:00:00Z", "2025-26"},
		{"garbage", "unknown"},
	}
	for _, c := range cases {
		if got := seasonOf(c.date); got != c.want {
			t.Errorf("seasonOf(%q) = %q, want %q", c.date, got, c.want)
		}
	}
}

func TestRunWithNilGuardsReturnsOnCancelledCtx(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	done := make(chan struct{})
	go func() { Run(ctx, Deps{}); close(done) }()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Run with nil store did not return")
	}
	done = make(chan struct{})
	go func() { Run(ctx, Deps{Source: &fakeSource{}}); close(done) }()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Run with nil store but non-nil source did not return")
	}
}
