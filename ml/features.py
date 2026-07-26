def build_game_state_features(
    score_diff: int,
    minute: int,
    xg_diff: float,
    pre_match_elo_diff: float,
    red_card_diff: int = 0,
) -> list:
    minute_norm = min(minute, 95) / 95.0
    time_remaining = 1.0 - minute_norm

    effective_lead = score_diff * (minute_norm ** 0.5)
    lead_in_remaining_time = score_diff * (1.0 - (time_remaining ** 0.5))
    lead_per_time = score_diff / (time_remaining + 0.1)

    return [
        score_diff,
        minute_norm,
        time_remaining,
        xg_diff,
        pre_match_elo_diff / 400.0,
        red_card_diff,
        score_diff * time_remaining,
        effective_lead,
        lead_in_remaining_time,
        lead_per_time,
    ]
