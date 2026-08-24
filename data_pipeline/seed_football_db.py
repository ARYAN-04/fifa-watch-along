"""Seed football.db from openfootball PL season JSONs, the frozen Reep register,
and the legacy wc2026.db (events + win probability snapshots).

Usage: uv run data_pipeline/seed_football_db.py
Idempotent: safe to run repeatedly against an existing football.db.
"""

import csv
import json
import re
import sqlite3
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
SCHEMA_PATH = ROOT / "internal/store/migrations/001_schema.sql"
OPENFOOTBALL_DIR = ROOT / "data_pipeline/data/openfootball"
REEP_TEAMS_CSV = ROOT / "data_pipeline/data/reep/teams.csv"
LEGACY_DB_PATH = ROOT / "wc2026.db"
TARGET_DB_PATH = ROOT / "football.db"

SEASONS = ["2021-22", "2022-23", "2023-24", "2024-25", "2025-26"]
COMPETITIONS = [
    ("PL", "Premier League", "England", 1),
    ("UCL", "UEFA Champions League", "Europe", 0),
    ("La Liga", "La Liga", "Spain", 0),
    ("Serie A", "Serie A", "Italy", 0),
    ("Bundesliga", "Bundesliga", "Germany", 0),
    ("WC2026", "FIFA World Cup 2026", "International", 0),
]
FOOTBALL_DATA_CODES = {
    "PL": "PL",
    "UCL": "CL",
    "La Liga": "PD",
    "Serie A": "SA",
    "Bundesliga": "BL1",
    "WC2026": "WC",
}
EVENT_TYPE_MAP = {"GOAL": "GOAL", "YELLOW_CARD": "CARD", "RED_CARD": "CARD", "SUBSTITUTION": "SUB"}
STOP_TOKENS = {"fc", "afc", "cf", "sc", "club", "the"}
REEP_SKIP_SUBSTRINGS = (
    "reserves", "academy", "historia", "history of", "ownership",
    "w.f.c.", "women", "ii", "development squad",
)


def normalize_name(raw):
    text = raw.lower().replace("&", " and ")
    text = re.sub(r"[^a-z0-9 ]+", " ", text)
    tokens = [t for t in text.split() if t not in STOP_TOKENS and len(t) > 1]
    return " ".join(tokens)


def display_name(raw):
    text = re.sub(r"\s*(FC|AFC|CF|SC)$", "", raw.strip())
    return text


def slugify(raw):
    return re.sub(r"[^a-z0-9]+", "-", raw.lower()).strip("-")


def load_reep_index():
    index = {}
    with REEP_TEAMS_CSV.open(newline="", encoding="utf-8") as handle:
        for row in csv.DictReader(handle):
            name = (row.get("name") or "").strip()
            if not name:
                continue
            lowered = name.lower()
            if any(marker in lowered for marker in REEP_SKIP_SUBSTRINGS):
                continue
            key = normalize_name(name)
            entry = {
                "reep_id": row.get("reep_id") or "",
                "clubeleo": row.get("key_clubelo") or "",
            }
            existing = index.get(key)
            if existing is None or (not existing["clubeleo"] and entry["clubeleo"]):
                index[key] = entry
    return index


def upsert_competition(conn, code, name, country, enabled):
    conn.execute(
        """INSERT INTO competitions (code, name, country, enabled)
           VALUES (?, ?, ?, ?)
           ON CONFLICT(code) DO UPDATE SET
               name = excluded.name, country = excluded.country, enabled = excluded.enabled""",
        (code, name, country, enabled),
    )
    row = conn.execute("SELECT id FROM competitions WHERE code = ?", (code,)).fetchone()
    return row[0]


def upsert_team(conn, name, country=None):
    conn.execute(
        """INSERT INTO teams (name, short_name, country)
           VALUES (?, ?, ?)
           ON CONFLICT(name) DO UPDATE SET country = COALESCE(excluded.country, teams.country)""",
        (name, display_name(name), country),
    )
    row = conn.execute("SELECT id FROM teams WHERE name = ?", (name,)).fetchone()
    return row[0]


def add_crosswalk(conn, team_id, provider, provider_id):
    conn.execute(
        """INSERT INTO id_crosswalk (team_id, comp_code, provider, provider_id)
           VALUES (?, NULL, ?, ?)
           ON CONFLICT(team_id, provider) DO UPDATE SET provider_id = excluded.provider_id""",
        (team_id, provider, provider_id),
    )


def add_comp_crosswalk(conn, comp_code, provider, provider_id):
    cur = conn.execute(
        "UPDATE id_crosswalk SET provider_id = ? WHERE comp_code = ? AND provider = ?",
        (provider_id, comp_code, provider),
    )
    if cur.rowcount == 0:
        conn.execute(
            """INSERT INTO id_crosswalk (team_id, comp_code, provider, provider_id)
               VALUES (NULL, ?, ?, ?)""",
            (comp_code, provider, provider_id),
        )


def upsert_match(conn, competition_id, external_id, season, utc_kickoff,
                 status, home_team_id, away_team_id, home_goals, away_goals, minute):
    conn.execute(
        """INSERT INTO matches (competition_id, external_id, season, utc_kickoff, status,
                                home_team_id, away_team_id, home_goals, away_goals, minute)
           VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
           ON CONFLICT(external_id) DO UPDATE SET
               status = excluded.status,
               utc_kickoff = excluded.utc_kickoff,
               home_goals = excluded.home_goals,
               away_goals = excluded.away_goals,
               minute = excluded.minute""",
        (competition_id, external_id, season, utc_kickoff, status,
         home_team_id, away_team_id, home_goals, away_goals, minute),
    )
    row = conn.execute(
        "SELECT id FROM matches WHERE external_id = ?", (external_id,)
    ).fetchone()
    return row[0]


def seed_openfootball(conn, competition_id, reep_index):
    team_ids = {}
    match_count = 0
    finished_count = 0
    for season in SEASONS:
        path = OPENFOOTBALL_DIR / f"{season}.en.1.json"
        data = json.loads(path.read_text(encoding="utf-8"))
        for match in data["matches"]:
            team1, team2 = match["team1"], match["team2"]
            for team_name in (team1, team2):
                if team_name not in team_ids:
                    team_ids[team_name] = upsert_team(conn, team_name, country="England")
                    add_crosswalk(conn, team_ids[team_name], "openfootball_name", team_name)
                    reep_entry = reep_index.get(normalize_name(team_name))
                    if reep_entry:
                        if reep_entry["reep_id"]:
                            add_crosswalk(conn, team_ids[team_name], "reep_id", reep_entry["reep_id"])
                        if reep_entry["clubeleo"]:
                            add_crosswalk(conn, team_ids[team_name], "clubeleo", reep_entry["clubeleo"])
            raw_score = match.get("score")
            if isinstance(raw_score, dict):
                raw_score = raw_score.get("ft")
            if isinstance(raw_score, list) and len(raw_score) == 2:
                status = "FINISHED"
                home_goals, away_goals = int(raw_score[0]), int(raw_score[1])
                finished_count += 1
            else:
                status, home_goals, away_goals = "SCHEDULED", None, None
            kickoff_time = match.get("time") or "00:00"
            external_id = f"of:{season}:{match['date']}:{slugify(team1)}-{slugify(team2)}"
            upsert_match(conn, competition_id, external_id, season,
                         f"{match['date']}T{kickoff_time}:00Z", status,
                         team_ids[team1], team_ids[team2], home_goals, away_goals, None)
            match_count += 1
    print(f"openfootball: {match_count} matches ({finished_count} finished), "
          f"{len(team_ids)} distinct teams across {len(SEASONS)} seasons")
    return set(team_ids)


def import_legacy(conn, competition_id):
    legacy = sqlite3.connect(f"file:{LEGACY_DB_PATH}?mode=ro", uri=True)
    try:
        legacy.row_factory = sqlite3.Row
        legacy_teams = {row["id"]: row["name"] for row in
                        legacy.execute("SELECT id, name FROM dashboard_team")}
        team_id_map = {}
        for legacy_id, name in sorted(legacy_teams.items()):
            team_id_map[legacy_id] = upsert_team(conn, name)

        imported_matches = 0
        for row in legacy.execute(
            "SELECT id, kickoff_utc, stage, status, home_score, away_score,"
            "       home_team_id, away_team_id FROM dashboard_match ORDER BY id"
        ):
            kickoff = row["kickoff_utc"].split(".")[0].replace(" ", "T") + "Z"
            external_id = f"wc2026:{row['id']}"
            match_id = upsert_match(conn, competition_id, external_id, "2026",
                                    kickoff, row["status"], team_id_map[row["home_team_id"]],
                                    team_id_map[row["away_team_id"]], row["home_score"],
                                    row["away_score"], None)
            has_events = conn.execute(
                "SELECT COUNT(*) FROM match_events WHERE match_id = ?", (match_id,)
            ).fetchone()[0]
            if not has_events:
                for event in legacy.execute(
                    "SELECT minute, event_type, player_name, assist_name, detail, team_id"
                    "   FROM dashboard_matchevent WHERE match_id = ? ORDER BY id",
                    (row["id"],),
                ):
                    side = "home" if event["team_id"] == row["home_team_id"] else "away"
                    detail_parts = [p for p in (event["detail"], event["assist_name"]) if p]
                    conn.execute(
                        """INSERT INTO match_events (match_id, minute, type, side, player, detail)
                           VALUES (?, ?, ?, ?, ?, ?)""",
                        (match_id, event["minute"],
                         EVENT_TYPE_MAP.get(event["event_type"], event["event_type"]),
                         side, event["player_name"], "; ".join(detail_parts) or None),
                    )
            for snap in legacy.execute(
                "SELECT minute, home_win_prob, draw_prob, away_win_prob"
                "   FROM dashboard_winprobabilitysnapshot WHERE match_id = ?",
                (row["id"],),
            ):
                conn.execute(
                    """INSERT INTO win_prob_snapshots (match_id, minute, home, draw, away)
                       VALUES (?, ?, ?, ?, ?)
                       ON CONFLICT(match_id, minute) DO UPDATE SET
                           home = excluded.home, draw = excluded.draw, away = excluded.away""",
                    (match_id, snap["minute"], snap["home_win_prob"],
                     snap["draw_prob"], snap["away_win_prob"]),
                )
            imported_matches += 1
        print(f"legacy wc2026: {imported_matches} matches imported under WC2026")
    finally:
        legacy.close()


def report_crosswalk_rate(conn, team_names, reep_index):
    total = len(team_names)
    unmatched = []
    for name in sorted(team_names):
        row = conn.execute(
            """SELECT COUNT(*) FROM id_crosswalk c
               JOIN teams t ON t.id = c.team_id
               WHERE t.name = ? AND c.provider IN ('reep_id', 'clubeleo')""",
            (name,),
        ).fetchone()
        if row[0] == 0:
            unmatched.append(name)
    matched = total - len(unmatched)
    print(f"crosswalk: {matched}/{total} teams matched to Reep "
          f"({100 * matched / total:.1f}%)")
    if unmatched:
        print("unmatched:", ", ".join(unmatched))


def report_counts(conn):
    tables = ["competitions", "teams", "id_crosswalk", "matches",
              "match_events", "win_prob_snapshots", "elo_ratings"]
    print("row counts:")
    for table in tables:
        count = conn.execute(f"SELECT COUNT(*) FROM {table}").fetchone()[0]
        print(f"  {table}: {count}")


def main():
    if not SCHEMA_PATH.exists():
        sys.exit(f"schema not found: {SCHEMA_PATH}")
    conn = sqlite3.connect(TARGET_DB_PATH)
    try:
        conn.execute("PRAGMA foreign_keys = ON")
        conn.executescript(SCHEMA_PATH.read_text(encoding="utf-8"))
        reep_index = load_reep_index()
        comp_ids = {}
        for code, name, country, enabled in COMPETITIONS:
            comp_ids[code] = upsert_competition(conn, code, name, country, enabled)
            add_comp_crosswalk(conn, code, "football_data", FOOTBALL_DATA_CODES[code])
        print(f"competitions: {', '.join(FOOTBALL_DATA_CODES[c] for c in comp_ids)} "
              f"(football_data crosswalk written)")
        team_names = seed_openfootball(conn, comp_ids["PL"], reep_index)
        import_legacy(conn, comp_ids["WC2026"])
        report_crosswalk_rate(conn, team_names, reep_index)
        conn.commit()
        report_counts(conn)
    finally:
        conn.close()


if __name__ == "__main__":
    main()
