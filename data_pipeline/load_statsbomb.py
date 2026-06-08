import json
from pathlib import Path
from statsbombpy import sb


def extract_game_states(match_id, home_team, away_team,
                        home_score, away_score):
    events = sb.events(match_id=match_id)
    shots = events[events['type'] == 'Shot'].copy()
    goals = shots[shots['shot_outcome'] == 'Goal'].copy()

    if home_score > away_score:
        label = 1
    elif home_score < away_score:
        label = -1
    else:
        label = 0

    rows = []
    for minute in range(1, 96):
        h_goals = len(goals[
            (goals['team'] == home_team) & (goals['minute'] <= minute)
        ])
        a_goals = len(goals[
            (goals['team'] == away_team) & (goals['minute'] <= minute)
        ])
        h_xg = float(shots[
            (shots['team'] == home_team) & (shots['minute'] <= minute)
        ]['shot_statsbomb_xg'].sum())
        a_xg = float(shots[
            (shots['team'] == away_team) & (shots['minute'] <= minute)
        ]['shot_statsbomb_xg'].sum())

        rows.append({
            'match_id': match_id,
            'minute': minute,
            'score_diff': h_goals - a_goals,
            'xg_diff': round(h_xg - a_xg, 4),
            'pre_match_elo_diff': 0,
            'red_card_diff': 0,
            'label': label,
        })
    return rows


def build_dataset():
    matches = sb.matches(competition_id=43, season_id=106)
    all_rows = []

    for _, m in matches.iterrows():
        rows = extract_game_states(
            match_id=m['match_id'],
            home_team=m['home_team'],
            away_team=m['away_team'],
            home_score=m['home_score'],
            away_score=m['away_score'],
        )
        all_rows.extend(rows)
        print(f"  {m['home_team']} {m['home_score']}-"
              f"{m['away_score']} {m['away_team']} — "
              f"{len(rows)} rows")

    out_path = Path(__file__).parent / 'data' / 'wc2022_game_states.json'
    with open(out_path, 'w') as f:
        json.dump(all_rows, f)

    print(f"\nTotal rows: {len(all_rows)} from {len(matches)} matches")
    print(f"Saved to {out_path}")


if __name__ == '__main__':
    build_dataset()
