import json
from pathlib import Path

import joblib
import numpy as np
from sklearn.calibration import CalibratedClassifierCV
from sklearn.ensemble import HistGradientBoostingClassifier
from sklearn.metrics import log_loss
from sklearn.model_selection import train_test_split

from features import build_game_state_features


def load_data(path=None):
    if path is None:
        path = Path(__file__).resolve().parent.parent / 'data_pipeline' / 'data' / 'wc2022_game_states.json'
    with open(path) as f:
        rows = json.load(f)
    X, y = [], []
    for r in rows:
        X.append(build_game_state_features(
            r['score_diff'], r['minute'], r['xg_diff'],
            r['pre_match_elo_diff'], r['red_card_diff'],
        ))
        y.append(r['label'])
    return np.array(X), np.array(y)


def train():
    X, y = load_data()
    X_train, X_test, y_train, y_test = train_test_split(
        X, y, test_size=0.2, random_state=42, stratify=y
    )

    base = HistGradientBoostingClassifier(
        max_iter=200, learning_rate=0.05, max_depth=4, random_state=42
    )
    model = CalibratedClassifierCV(base, method='isotonic', cv=5)
    model.fit(X_train, y_train)

    probs = model.predict_proba(X_test)
    print(f"Log loss on held-out set: {log_loss(y_test, probs):.4f}")

    out_path = Path(__file__).parent / 'win_prob_model.pkl'
    joblib.dump(model, out_path)
    print(f"Saved to {out_path}")

    sanity = [
        (0,  1,  0.0,   0, 0, "Kickoff, equal teams"),
        (1, 45,  0.5,   0, 0, "1-0 up at HT, slight xG edge"),
        (2, 85,  1.2,  50, 0, "2-0 up at 85min, stronger team"),
        (0, 80, -0.8, 100, 0, "0-0 at 80min, losing xG battle"),
        (-1, 60, 0.3,   0, 0, "1-0 down at 60, winning xG"),
    ]
    print("\nSanity checks (home_win / draw / away_win):")
    for sd, mn, xg, elo, rc, desc in sanity:
        f = build_game_state_features(sd, mn, xg, elo, rc)
        p = model.predict_proba([f])[0]
        pm = dict(zip(model.classes_.tolist(), p.tolist()))
        print(f"  {desc}")
        print(f"    Win {pm.get(1, 0):.1%} | Draw {pm.get(0, 0):.1%}"
              f" | Loss {pm.get(-1, 0):.1%}")


if __name__ == '__main__':
    train()
