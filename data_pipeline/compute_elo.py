"""Offline club-Elo computation over FINISHED matches in football.db.

Usage: uv run python data_pipeline/compute_elo.py [--db football.db] [--k 20] [--base 1500]

Standard Elo with expected = 1/(1+10^((opp-team)/400)) and a World-Football-Elo-style
goal-difference multiplier G: |diff|<=1 -> 1, ==2 -> 1.5, else (11+|diff|)/8.
Recomputed from scratch each run; final ratings upserted into elo_ratings.
"""

import argparse
import sqlite3
from datetime import datetime, timezone


def parse_args():
    parser = argparse.ArgumentParser(description="Compute offline club Elo ratings.")
    parser.add_argument("--db", default="football.db")
    parser.add_argument("--k", type=float, default=20.0)
    parser.add_argument("--base", type=float, default=1500.0)
    return parser.parse_args()


def goal_diff_multiplier(diff: int) -> float:
    if diff <= 1:
        return 1.0
    if diff == 2:
        return 1.5
    return (11.0 + diff) / 8.0


def compute_ratings(conn: sqlite3.Connection, k: float, base: float) -> dict[int, float]:
    rows = conn.execute(
        """
        SELECT utc_kickoff, home_team_id, away_team_id, home_goals, away_goals
        FROM matches
        WHERE status = 'FINISHED'
          AND home_goals IS NOT NULL AND away_goals IS NOT NULL
        ORDER BY utc_kickoff ASC
        """
    ).fetchall()

    ratings: dict[int, float] = {}
    for _kickoff, home_id, away_id, hg, ag in rows:
        rh = ratings.get(home_id, base)
        ra = ratings.get(away_id, base)
        expected_home = 1.0 / (1.0 + 10.0 ** ((ra - rh) / 400.0))
        actual_home = 1.0 if hg > ag else 0.5 if hg == ag else 0.0
        delta = k * goal_diff_multiplier(abs(hg - ag)) * (actual_home - expected_home)
        ratings[home_id] = rh + delta
        ratings[away_id] = ra - delta
    return ratings


def persist(conn: sqlite3.Connection, ratings: dict[int, float]) -> None:
    now = datetime.now(timezone.utc).isoformat(timespec="seconds")
    conn.executemany(
        """
        INSERT INTO elo_ratings (team_id, rating, updated_at)
        VALUES (?, ?, ?)
        ON CONFLICT(team_id) DO UPDATE SET
            rating = excluded.rating,
            updated_at = excluded.updated_at
        """,
        [(team_id, rating, now) for team_id, rating in sorted(ratings.items())],
    )
    conn.commit()


def print_top10(conn: sqlite3.Connection, ratings: dict[int, float], base: float) -> None:
    names = dict(conn.execute("SELECT id, name FROM teams").fetchall())
    ranked = sorted(ratings.items(), key=lambda kv: kv[1], reverse=True)[:10]
    print(f"{'Rank':<6}{'Team':<28}{'Rating':>10}")
    print("-" * 44)
    for i, (team_id, rating) in enumerate(ranked, start=1):
        print(f"{i:<6}{names.get(team_id, f'team {team_id}'):<28}{rating:>10.1f}")
    print(f"\nComputed {len(ratings)} team ratings (base {base:g}).")


def main() -> int:
    args = parse_args()
    conn = sqlite3.connect(args.db)
    try:
        ratings = compute_ratings(conn, args.k, args.base)
        persist(conn, ratings)
        print_top10(conn, ratings, args.base)
    finally:
        conn.close()
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
