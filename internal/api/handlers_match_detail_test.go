package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fifa-watch-along/fifa-hub/internal/store/db"
)

func serveMatchRoute(t *testing.T, mux *http.ServeMux, path string) (*http.Response, func()) {
	t.Helper()
	srv := httptest.NewServer(mux)
	resp, err := http.Get(srv.URL + path)
	if err != nil {
		srv.Close()
		t.Fatalf("GET %s: %v", path, err)
	}
	return resp, srv.Close
}

func TestMatchDetailHandlerFound(t *testing.T) {
	st := seedLiveMatchStore(t)
	mux := http.NewServeMux()
	Register(mux, Deps{Store: st})

	resp, closeSrv := serveMatchRoute(t, mux, "/api/matches/1")
	defer func() { resp.Body.Close(); closeSrv() }()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var got matchDetail
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.ID != 1 || got.Status != "LIVE" || got.Season != "2026-27" {
		t.Errorf("detail = %+v", got)
	}
	if got.Home.Name != "Arsenal" || got.Away.Name != "Chelsea" {
		t.Errorf("teams = %s vs %s", got.Home.Name, got.Away.Name)
	}
	if got.HomeGoals == nil || *got.HomeGoals != 2 || got.Minute == nil || *got.Minute != 63 {
		t.Errorf("score/minute = %+v", got)
	}
}

func TestMatchDetailHandlerNotFound(t *testing.T) {
	st := seedLiveMatchStore(t)
	mux := http.NewServeMux()
	Register(mux, Deps{Store: st})

	resp, closeSrv := serveMatchRoute(t, mux, "/api/matches/999")
	defer func() { resp.Body.Close(); closeSrv() }()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", resp.StatusCode)
	}
}

func TestMatchEventsOrderedChronologically(t *testing.T) {
	st := seedLiveMatchStore(t)
	ctx := t.Context()
	for _, ev := range []db.InsertMatchEventParams{
		{MatchID: 1, Minute: 63, Type: "GOAL", Side: "AWAY"},
		{MatchID: 1, Minute: 20, Type: "GOAL", Side: "HOME"},
		{MatchID: 1, Minute: 55, Type: "GOAL", Side: "HOME"},
	} {
		ev.Player = sqlNullString("")
		ev.Detail = sqlNullString("")
		if _, err := st.InsertMatchEvent(ctx, ev); err != nil {
			t.Fatalf("seed event: %v", err)
		}
	}
	mux := http.NewServeMux()
	Register(mux, Deps{Store: st})

	resp, closeSrv := serveMatchRoute(t, mux, "/api/matches/1/events")
	defer func() { resp.Body.Close(); closeSrv() }()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var got struct {
		Events []matchEvent `json:"events"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(got.Events) != 3 {
		t.Fatalf("events = %d, want 3", len(got.Events))
	}
	wantMinutes := []int64{20, 55, 63}
	for i, e := range got.Events {
		if e.Minute != wantMinutes[i] {
			t.Errorf("events[%d].minute = %d, want %d", i, e.Minute, wantMinutes[i])
		}
	}
}

func TestWinProbabilityShape(t *testing.T) {
	st := seedLiveMatchStore(t)
	ctx := t.Context()
	snapshots := []db.InsertWinProbSnapshotParams{
		{MatchID: 1, Minute: 1, Home: 0.4, Draw: 0.3, Away: 0.3},
		{MatchID: 1, Minute: 45, Home: 0.6, Draw: 0.25, Away: 0.15},
	}
	for _, s := range snapshots {
		if err := st.InsertWinProbSnapshot(ctx, s); err != nil {
			t.Fatalf("seed snapshot: %v", err)
		}
	}
	mux := http.NewServeMux()
	Register(mux, Deps{Store: st})

	resp, closeSrv := serveMatchRoute(t, mux, "/api/matches/1/win-probability")
	defer func() { resp.Body.Close(); closeSrv() }()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var got winProbabilityResponse
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.PreMatch == nil || got.PreMatch.Home != 0.4 || got.PreMatch.Draw != 0.3 || got.PreMatch.Away != 0.3 {
		t.Errorf("preMatch = %+v, want earliest snapshot (minute 1)", got.PreMatch)
	}
	if len(got.Snapshots) != 2 {
		t.Fatalf("snapshots = %d, want 2", len(got.Snapshots))
	}
	if got.Snapshots[1].Minute != 45 || got.Snapshots[1].Home != 0.6 {
		t.Errorf("snapshots[1] = %+v", got.Snapshots[1])
	}
}

func TestWinProbabilityFinishedWithoutSnapshotsPreMatchNull(t *testing.T) {
	st := seedLiveMatchStore(t)
	mux := http.NewServeMux()
	Register(mux, Deps{Store: st})

	resp, closeSrv := serveMatchRoute(t, mux, "/api/matches/2/win-probability")
	defer func() { resp.Body.Close(); closeSrv() }()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	val, ok := body["preMatch"]
	if !ok || val != nil {
		t.Errorf("preMatch = %v (%T), want explicit null", val, val)
	}
	snaps, ok := body["snapshots"].([]any)
	if !ok || len(snaps) != 0 {
		t.Errorf("snapshots = %v, want empty array", body["snapshots"])
	}
}

func TestWinProbabilityNotFound(t *testing.T) {
	st := seedLiveMatchStore(t)
	mux := http.NewServeMux()
	Register(mux, Deps{Store: st})

	resp, closeSrv := serveMatchRoute(t, mux, "/api/matches/999/win-probability")
	defer func() { resp.Body.Close(); closeSrv() }()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", resp.StatusCode)
	}
}
