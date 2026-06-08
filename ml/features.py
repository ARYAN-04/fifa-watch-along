def build_game_state_features(
    score_diff: int,
    minute: int,
    xg_diff: float,
    pre_match_elo_diff: float,
    red_card_diff: int = 0,
) -> list:
    minute_norm = min(minute, 95) / 95.0
    time_remaining = 1.0 - minute_norm

    return [
        score_diff,
        minute_norm,
        time_remaining,
        xg_diff,
        pre_match_elo_diff / 400.0,
        red_card_diff,
        score_diff * time_remaining,
    ]
