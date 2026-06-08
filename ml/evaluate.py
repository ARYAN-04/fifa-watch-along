"""
Calibration evaluation: run this after train.py has produced win_prob_model.pkl.
Plots a calibration curve and prints brier scores per class.
"""
import json
from pathlib import Path

import numpy as np
from sklearn.metrics import brier_score_loss
from sklearn.model_selection import train_test_split

import joblib
from features import build_game_state_features


def evaluate():
    data_path = Path(__file__).resolve().parent.parent / 'data_pipeline' / 'data' / 'wc2022_game_states.json'
    model_path = Path(__file__).parent / 'win_prob_model.pkl'

    with open(data_path) as f:
        rows = json.load(f)
    X, y = [], []
    for r in rows:
        X.append(build_game_state_features(
            r['score_diff'], r['minute'], r['xg_diff'],
            r['pre_match_elo_diff'], r['red_card_diff'],
        ))
        y.append(r['label'])
    X, y = np.array(X), np.array(y)

    _, X_test, _, y_test = train_test_split(
        X, y, test_size=0.2, random_state=42, stratify=y
    )

    model = joblib.load(model_path)
    probs = model.predict_proba(X_test)
    classes = model.classes_.tolist()

    print("Brier scores (lower is better, 0 = perfect):")
    for i, cls in enumerate(classes):
        y_binary = (y_test == cls).astype(int)
        score = brier_score_loss(y_binary, probs[:, i])
        label = {1: 'Home Win', 0: 'Draw', -1: 'Away Win'}[cls]
        print(f"  {label}: {score:.4f}")

    print(f"\nMean predicted probability vs actual frequency:")
    for cls in classes:
        label = {1: 'Home Win', 0: 'Draw', -1: 'Away Win'}[cls]
        idx = classes.index(cls)
        mask = (y_test == cls)
        mean_pred = probs[mask, idx].mean()
        actual_freq = mask.mean()
        print(f"  {label}: mean_pred={mean_pred:.3f}, actual_freq={actual_freq:.3f}")


if __name__ == '__main__':
    evaluate()
