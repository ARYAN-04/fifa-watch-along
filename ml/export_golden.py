"""Generate the golden parity fixture for the Go inference engine.

Samples 500 game states deterministically from wc2022_game_states.json,
computes pickle predict_proba, writes internal/inference/testdata/golden.json.
"""

import json
import random
from pathlib import Path

import joblib
import numpy as np

from features import build_game_state_features

ROOT = Path(__file__).resolve().parent.parent
STATES_PATH = ROOT / "data_pipeline" / "data" / "wc2022_game_states.json"
MODEL_PATH = Path(__file__).resolve().parent / "win_prob_model.pkl"
OUT_PATH = ROOT / "internal" / "inference" / "testdata" / "golden.json"

N_SAMPLES = 500
SEED = 42


def main() -> None:
    with open(STATES_PATH) as f:
        states = json.load(f)

    rng = random.Random(SEED)
    sampled = rng.sample(states, N_SAMPLES)

    X = np.array([
        build_game_state_features(
            s["score_diff"], s["minute"], s["xg_diff"],
            s["pre_match_elo_diff"], s["red_card_diff"],
        )
        for s in sampled
    ])

    model = joblib.load(MODEL_PATH)
    probs = model.predict_proba(X)
    class_order = model.classes_.tolist()

    fixture = {
        "classes": class_order,
        "cases": [
            {"input": list(x), "expected": list(p)}
            for x, p in zip(X.tolist(), probs.tolist())
        ],
    }

    OUT_PATH.parent.mkdir(parents=True, exist_ok=True)
    with open(OUT_PATH, "w") as f:
        json.dump(fixture, f)

    print(f"Wrote {len(fixture['cases'])} golden cases to {OUT_PATH} (class order {class_order})")


if __name__ == "__main__":
    main()
