import os
import joblib
from pathlib import Path
from ml.features import build_game_state_features

# Load ML model
MODEL_PATH = Path(__file__).resolve().parent.parent.parent / 'ml' / 'win_prob_model.pkl'

_model = None

def get_model():
    global _model
    if _model is None:
        if not MODEL_PATH.exists():
            raise FileNotFoundError(f"Model file not found at {MODEL_PATH}")
        _model = joblib.load(MODEL_PATH)
    return _model

def predict_win_probability(
    score_diff: int,
    minute: int,
    xg_diff: float,
    pre_match_elo_diff: float,
    red_card_diff: int = 0
) -> dict:
    """
    Computes ML win probabilities (home_win, draw, away_win) for a given match state.
    """
    model = get_model()
    features = build_game_state_features(
        score_diff=score_diff,
        minute=minute,
        xg_diff=xg_diff,
        pre_match_elo_diff=pre_match_elo_diff,
        red_card_diff=red_card_diff
    )
    
    probs = model.predict_proba([features])[0]
    classes = model.classes_
    
    # Map probabilities to classes
    class_probs = dict(zip(classes.tolist(), probs.tolist()))
    
    # Label mapping: 1 is Home Win, 0 is Draw, -1 is Away Win (Loss)
    return {
        "home_win": class_probs.get(1, 0.0),
        "draw": class_probs.get(0, 0.0),
        "away_win": class_probs.get(-1, 0.0)
    }
