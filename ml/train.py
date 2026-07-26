import json
from pathlib import Path

import joblib
import numpy as np
from sklearn.linear_model import LogisticRegression
from sklearn.ensemble import RandomForestClassifier, VotingClassifier
from sklearn.calibration import CalibratedClassifierCV
from sklearn.preprocessing import StandardScaler
from sklearn.pipeline import make_pipeline
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
        # Original perspective
        X.append(build_game_state_features(
            r['score_diff'], r['minute'], r['xg_diff'],
            r['pre_match_elo_diff'], r['red_card_diff'],
        ))
        y.append(r['label'])

        # Symmetric inverse perspective for neutral-ground balance
        X.append(build_game_state_features(
            -r['score_diff'], r['minute'], -r['xg_diff'],
            -r['pre_match_elo_diff'], -r['red_card_diff'],
        ))
        y.append(-r['label'])

    return np.array(X), np.array(y)


def train():
    X, y = load_data()
    X_train, X_test, y_train, y_test = train_test_split(
        X, y, test_size=0.2, random_state=42, stratify=y
    )

    lr_pipeline = make_pipeline(
        StandardScaler(),
        LogisticRegression(C=0.5, max_iter=1000, random_state=42)
    )

    rf = RandomForestClassifier(
        n_estimators=100, max_depth=4, min_samples_leaf=10, random_state=42
    )
    calibrated_rf = CalibratedClassifierCV(rf, method='sigmoid', cv=5)

    model = VotingClassifier(
        estimators=[('lr', lr_pipeline), ('rf', calibrated_rf)],
        voting='soft'
    )
    model.fit(X_train, y_train)

    probs = model.predict_proba(X_test)
    print(f"Log loss on held-out set: {log_loss(y_test, probs):.4f}")

    out_path = Path(__file__).parent / 'win_prob_model.pkl'
    joblib.dump(model, out_path)
    print(f"Saved tuned model to {out_path}")

    sanity = [
        (0,   5,  0.0,   0, 0, "0-0 at 5 min, equal teams"),
        (1,   5,  0.2,   0, 0, "1-0 at 5 min (early goal)"),
        (1,  20,  0.4,   0, 0, "1-0 at 20 min"),
        (1,  45,  0.5,   0, 0, "1-0 at HT (45 min)"),
        (1,  75,  0.8,   0, 0, "1-0 at 75 min"),
        (1,  88,  1.0,   0, 0, "1-0 at 88 min (late goal)"),
        (2,  85,  1.5,  50, 0, "2-0 at 85 min, stronger team"),
        (0,  85, -0.8, 100, 0, "0-0 at 85 min, losing xG battle"),
        (-1, 60,  0.3,   0, 0, "1-0 down at 60 min, winning xG"),
    ]
    print("\nSanity checks (home_win / draw / away_win):")
    for sd, mn, xg, elo, rc, desc in sanity:
        f = build_game_state_features(sd, mn, xg, elo, rc)
        p = model.predict_proba([f])[0]
        pm = dict(zip(model.classes_.tolist(), p.tolist()))
        print(f"  {desc:38s}")
        print(f"    Win {pm.get(1, 0):.1%} | Draw {pm.get(0, 0):.1%}"
              f" | Loss {pm.get(-1, 0):.1%}")


if __name__ == '__main__':
    train()
