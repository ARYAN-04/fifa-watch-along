package inference

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"testing"
)

type goldenCase struct {
	Input    []float64 `json:"input"`
	Expected []float64 `json:"expected"`
}

type goldenFixture struct {
	Cases []goldenCase `json:"cases"`
}

func loadGolden(t *testing.T) []goldenCase {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("testdata", "golden.json"))
	if err != nil {
		t.Fatalf("read golden fixture: %v", err)
	}
	var f goldenFixture
	if err := json.Unmarshal(raw, &f); err != nil {
		t.Fatalf("parse golden fixture: %v", err)
	}
	return f.Cases
}

func TestPredictGoldenParity(t *testing.T) {
	const tol = 1e-6

	eng, err := New(filepath.Join("..", "..", "ml", "export", "model.json"))
	if err != nil {
		t.Fatalf("load model: %v", err)
	}

	cases := loadGolden(t)
	maxAbs := 0.0
	for idx, c := range cases {
		var in [10]float64
		copy(in[:], c.Input)
		got, err := eng.Predict(in)
		if err != nil {
			t.Fatalf("case %d: %v", idx, err)
		}
		for k := range got {
			d := math.Abs(got[k] - c.Expected[k])
			if d > maxAbs {
				maxAbs = d
			}
			if d > tol {
				t.Fatalf("case %d class %d: got %.15f want %.15f (diff %g exceeds %g)",
					idx, k, got[k], c.Expected[k], d, tol)
			}
		}
	}
	fmt.Printf("PASS: %d golden cases, max abs error %.3e (tolerance %g)\n", len(cases), maxAbs, tol)
}
