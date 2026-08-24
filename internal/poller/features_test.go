package poller

import (
	"context"
	"math"
	"testing"
	"time"

	"github.com/fifa-watch-along/fifa-hub/internal/inference"
	"github.com/fifa-watch-along/fifa-hub/internal/source"
)

func fixedPredict([10]float64) ([3]float64, error) {
	return [3]float64{0.5, 0.3, 0.2}, nil
}

func TestPollerInsertsSnapshotOncePerMinute(t *testing.T) {
	ctx := context.Background()
	st := openTestStore(t)
	src := &fakeSource{
		snapshots: [][]source.Match{
			{testMatch("541003", "Arsenal", "Chelsea", intPtr(0), intPtr(0), intPtr(10))},
			{testMatch("541003", "Arsenal", "Chelsea", intPtr(0), intPtr(0), intPtr(10))},
			{testMatch("541003", "Arsenal", "Chelsea", intPtr(1), intPtr(0), intPtr(30))},
			{testMatch("541003", "Arsenal", "Chelsea", intPtr(1), intPtr(0), intPtr(30))},
		},
	}
	d := Deps{Store: st, Source: src, Predict: fixedPredict, Every: time.Second}
	seen := map[string]matchState{}

	for range 4 {
		poll(ctx, d, seen)
	}

	snapshots, err := st.ListWinProbSnapshotsByMatch(ctx, 1)
	if err != nil {
		t.Fatalf("ListWinProbSnapshotsByMatch: %v", err)
	}
	if len(snapshots) != 2 {
		t.Fatalf("snapshots = %d, want 2 (one per distinct minute)", len(snapshots))
	}
	if snapshots[0].Minute != 10 || snapshots[1].Minute != 30 {
		t.Errorf("minutes = %d,%d want 10,30", snapshots[0].Minute, snapshots[1].Minute)
	}
	if snapshots[1].Home != 0.5 || snapshots[1].Draw != 0.3 || snapshots[1].Away != 0.2 {
		t.Errorf("probs = %+v, want stub values", snapshots[1])
	}
}

func TestPollerNilPredictSkipsSnapshots(t *testing.T) {
	ctx := context.Background()
	st := openTestStore(t)
	src := &fakeSource{
		snapshots: [][]source.Match{
			{testMatch("541004", "Liverpool", "Everton", intPtr(1), intPtr(0), intPtr(15))},
		},
	}
	d := Deps{Store: st, Source: src, Every: time.Second}

	poll(ctx, d, map[string]matchState{})

	snapshots, err := st.ListWinProbSnapshotsByMatch(ctx, 1)
	if err != nil {
		t.Fatalf("ListWinProbSnapshotsByMatch: %v", err)
	}
	if len(snapshots) != 0 {
		t.Errorf("snapshots with nil Predict = %d, want 0", len(snapshots))
	}
}

func TestBuildFeaturesMatchesPythonSemantics(t *testing.T) {
	got := inference.BuildFeatures(2, 1, 63, 1800, 1600)
	minuteNorm := 63.0 / 95.0
	timeRemaining := 1.0 - minuteNorm
	scoreDiff := 1.0
	want := [10]float64{
		scoreDiff,
		minuteNorm,
		timeRemaining,
		0,
		200.0 / 400.0,
		0,
		scoreDiff * timeRemaining,
		scoreDiff * math.Sqrt(minuteNorm),
		scoreDiff * (1.0 - math.Sqrt(timeRemaining)),
		scoreDiff / (timeRemaining + 0.1),
	}
	for i := range want {
		if diff := got[i] - want[i]; diff < -1e-12 || diff > 1e-12 {
			t.Errorf("feature[%d] = %.15f, want %.15f", i, got[i], want[i])
		}
	}
}

func TestBuildFeaturesZeroEloDefaultsTo1500(t *testing.T) {
	got := inference.BuildFeatures(0, 0, 1, 0, 0)
	if got[4] != 0 {
		t.Errorf("elo_diff/400 = %f, want 0 for missing elos", got[4])
	}
}
