"""
Seed wc2026.db with a completed WC 2026 match plus a realistic event
timeline and per-minute win probability snapshots for replay.

Stage 1 of the replay pipeline: this writes the legacy dashboard_* schema,
and data_pipeline/seed_football_db.py maps it read-only into football.db
via import_legacy. Run one match per invocation.

Usage:
  uv run scripts/seed_replay_match.py                        # France 4-6 England (default)
  uv run scripts/seed_replay_match.py --match-id 537390      # Final: Spain 1-0 Argentina
  uv run scripts/seed_replay_match.py --groups-only          # Just seed group teams/standings

Needs FOOTBALL_DATA_API_KEY in the environment or .env (skipped for --groups-only).
"""
import argparse
import json
import random
import sqlite3
import sys
import urllib.request
from datetime import datetime, timezone
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT / "ml"))

from features import build_game_state_features  # noqa: E402

import joblib  # noqa: E402

DB_PATH = ROOT / "wc2026.db"
MODEL_PATH = ROOT / "ml" / "win_prob_model.pkl"
FD_BASE = "https://api.football-data.org/v4"

DEFAULT_MATCH_ID = 537389

SCHEMA = """
CREATE TABLE IF NOT EXISTS dashboard_team (
    id INTEGER PRIMARY KEY,
    name TEXT NOT NULL,
    short_name TEXT DEFAULT '',
    flag_url TEXT DEFAULT '',
    "group" TEXT DEFAULT '',
    pre_match_elo REAL DEFAULT 1500.0,
    fc26_overall INTEGER NULL
);
CREATE TABLE IF NOT EXISTS dashboard_player (
    id INTEGER PRIMARY KEY,
    team_id INTEGER NOT NULL,
    name TEXT NOT NULL,
    position TEXT DEFAULT '',
    overall_rating INTEGER DEFAULT 0,
    pace INTEGER DEFAULT 0,
    shooting INTEGER DEFAULT 0,
    passing INTEGER DEFAULT 0,
    dribbling INTEGER DEFAULT 0,
    defending INTEGER DEFAULT 0,
    physical INTEGER DEFAULT 0,
    skill_moves INTEGER DEFAULT 0,
    weak_foot INTEGER DEFAULT 0,
    nationality TEXT DEFAULT ''
);
CREATE TABLE IF NOT EXISTS dashboard_match (
    id INTEGER PRIMARY KEY,
    home_team_id INTEGER NOT NULL,
    away_team_id INTEGER NOT NULL,
    kickoff_utc TEXT NOT NULL,
    stage TEXT NOT NULL,
    venue TEXT DEFAULT '',
    status TEXT DEFAULT 'SCHEDULED',
    home_score INTEGER DEFAULT 0,
    away_score INTEGER DEFAULT 0
);
CREATE TABLE IF NOT EXISTS dashboard_matchevent (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    match_id INTEGER NOT NULL,
    minute INTEGER NOT NULL,
    event_type TEXT NOT NULL,
    team_id INTEGER NULL,
    player_name TEXT NOT NULL,
    assist_name TEXT DEFAULT '',
    detail TEXT DEFAULT '',
    created_at TEXT NOT NULL,
    UNIQUE (match_id, minute, event_type, player_name)
);
CREATE TABLE IF NOT EXISTS dashboard_winprobabilitysnapshot (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    match_id INTEGER NOT NULL,
    minute INTEGER NOT NULL,
    home_win_prob REAL NOT NULL,
    draw_prob REAL NOT NULL,
    away_win_prob REAL NOT NULL,
    score_diff INTEGER NOT NULL,
    xg_diff_approx REAL NOT NULL,
    created_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS dashboard_standing (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    team_id INTEGER NOT NULL,
    "group" TEXT NOT NULL,
    position INTEGER NOT NULL,
    played INTEGER DEFAULT 0,
    won INTEGER DEFAULT 0,
    drawn INTEGER DEFAULT 0,
    lost INTEGER DEFAULT 0,
    goals_for INTEGER DEFAULT 0,
    goals_against INTEGER DEFAULT 0,
    points INTEGER DEFAULT 0,
    updated_at TEXT NOT NULL,
    UNIQUE (team_id, "group")
);
CREATE TABLE IF NOT EXISTS dashboard_matchconfig (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    current_match_id INTEGER NULL,
    updated_at TEXT NOT NULL
);
"""

KNOWN_EVENTS = {
    537389: {
        "goals": [
            {"minute": 12, "team": "away", "player": "H. Kane"},
            {"minute": 28, "team": "away", "player": "J. Bellingham"},
            {"minute": 37, "team": "away", "player": "B. Saka"},
            {"minute": 45, "stoppage": 2, "team": "away", "player": "H. Kane"},
            {"minute": 48, "team": "home", "player": "K. Mbappé"},
            {"minute": 55, "team": "away", "player": "D. Rice"},
            {"minute": 62, "team": "home", "player": "A. Griezmann"},
            {"minute": 75, "team": "home", "player": "K. Mbappé"},
            {"minute": 88, "team": "away", "player": "J. Bellingham"},
            {"minute": 90, "stoppage": 3, "team": "home", "player": "K. Mbappé"},
        ],
        "yellow_cards": [
            {"minute": 23, "team": "home", "player": "A. Tchouaméni"},
            {"minute": 40, "team": "away", "player": "D. Rice"},
            {"minute": 52, "team": "away", "player": "K. Walker"},
            {"minute": 70, "team": "home", "player": "J. Koundé"},
            {"minute": 85, "team": "home", "player": "O. Dembélé"},
        ],
        "red_cards": [],
        "substitutions": [
            {"minute": 58, "team": "home", "player_off": "O. Giroud", "player_on": "R. Kolo Muani"},
            {"minute": 65, "team": "away", "player_off": "M. Rashford", "player_on": "C. Palmer"},
            {"minute": 78, "team": "home", "player_off": "A. Griezmann", "player_on": "E. Camavinga"},
            {"minute": 80, "team": "away", "player_off": "B. Saka", "player_on": "J. Grealish"},
            {"minute": 85, "team": "home", "player_off": "K. Mbappé", "player_on": "M. Thuram"},
            {"minute": 90, "team": "away", "player_off": "H. Kane", "player_on": "I. Toney"},
        ],
    },
}

TEAM_ELOS = {
    760: 2050.0,   # Spain
    762: 2140.0,   # Argentina
    770: 2050.0,   # England
    773: 2110.0,   # France
    8872: 1840.0,  # Norway
    799: 1890.0,   # Croatia
    804: 1750.0,   # Senegal
    763: 1620.0,   # Ghana
    1836: 1650.0,  # Panama
    8062: 1580.0,  # Iraq
}

STANDINGS_TEMPLATE = {
    "I": [
        {"id": 773, "name": "France", "played": 3, "won": 3, "drawn": 0, "lost": 0, "gf": 8, "ga": 1, "pts": 9},
        {"id": 8872, "name": "Norway", "played": 3, "won": 2, "drawn": 0, "lost": 1, "gf": 5, "ga": 3, "pts": 6},
        {"id": 804, "name": "Senegal", "played": 3, "won": 1, "drawn": 0, "lost": 2, "gf": 3, "ga": 5, "pts": 3},
        {"id": 8062, "name": "Iraq", "played": 3, "won": 0, "drawn": 0, "lost": 3, "gf": 1, "ga": 8, "pts": 0},
    ],
    "L": [
        {"id": 770, "name": "England", "played": 3, "won": 2, "drawn": 1, "lost": 0, "gf": 7, "ga": 3, "pts": 7},
        {"id": 799, "name": "Croatia", "played": 3, "won": 1, "drawn": 2, "lost": 0, "gf": 4, "ga": 2, "pts": 5},
        {"id": 763, "name": "Ghana", "played": 3, "won": 0, "drawn": 2, "lost": 1, "gf": 3, "ga": 5, "pts": 2},
        {"id": 1836, "name": "Panama", "played": 3, "won": 0, "drawn": 1, "lost": 2, "gf": 2, "ga": 6, "pts": 1},
    ],
}


def generate_events(home_score: int, away_score: int, ht_home: int, ht_away: int) -> dict:
    goals = []
    yellow_cards = []
    red_cards = []
    substitutions = []

    first_half_slots = [8, 18, 25, 32, 38, 42, 45]
    second_half_slots = [48, 55, 62, 68, 75, 82, 88]

    random.shuffle(first_half_slots)
    random.shuffle(second_half_slots)

    idx = 0
    for _ in range(ht_home):
        if idx < len(first_half_slots):
            goals.append({"minute": first_half_slots[idx], "team": "home", "player": f"Player H{idx+1}"})
            idx += 1
    idx = 0
    for _ in range(ht_away):
        if idx < len(first_half_slots):
            goals.append({"minute": first_half_slots[len(first_half_slots)-1-idx], "team": "away", "player": f"Player A{idx+1}"})
            idx += 1

    idx = 0
    for _ in range(home_score - ht_home):
        if idx < len(second_half_slots):
            goals.append({"minute": second_half_slots[idx], "team": "home", "player": f"Player H{idx+1}"})
            idx += 1
    idx = 0
    for _ in range(away_score - ht_away):
        if idx < len(second_half_slots):
            goals.append({"minute": second_half_slots[len(second_half_slots)-1-idx], "team": "away", "player": f"Player A{idx+1}"})
            idx += 1

    extra_home = max(0, home_score - ht_home - len(second_half_slots))
    extra_away = max(0, away_score - ht_away - len(second_half_slots))
    for i in range(extra_home):
        goals.append({"minute": 90, "stoppage": i + 1, "team": "home", "player": f"Player H-ET{i+1}"})
    for i in range(extra_away):
        goals.append({"minute": 90, "stoppage": i + 1, "team": "away", "player": f"Player A-ET{i+1}"})

    for i in range(min(5, 6)):
        team = "home" if i % 2 == 0 else "away"
        yellow_cards.append({"minute": [15, 30, 40, 50, 65, 80][i], "team": team, "player": f"Player {team.upper()}-YC{i+1}"})

    for i in range(min(6, 6)):
        team = "home" if i % 2 == 0 else "away"
        substitutions.append({
            "minute": [55, 65, 70, 75, 80, 85][i], "team": team,
            "player_off": f"P. {team.upper()}-Off{i+1}",
            "player_on": f"P. {team.upper()}-On{i+1}",
        })

    goals.sort(key=lambda g: g["minute"] + g.get("stoppage", 0))
    return {
        "goals": goals,
        "yellow_cards": yellow_cards,
        "red_cards": red_cards,
        "substitutions": substitutions,
    }


def get_score_at_minute(minute: int, goals: list) -> tuple:
    home = 0
    away = 0
    for g in goals:
        if g["minute"] + g.get("stoppage", 0) <= minute:
            if g["team"] == "home":
                home += 1
            else:
                away += 1
    return home, away


def get_elo_diff(conn, home_team_id: int, away_team_id: int) -> float:
    home = conn.execute("SELECT pre_match_elo FROM dashboard_team WHERE id = ?", (home_team_id,)).fetchone()
    away = conn.execute("SELECT pre_match_elo FROM dashboard_team WHERE id = ?", (away_team_id,)).fetchone()
    return (home[0] if home else 1500.0) - (away[0] if away else 1500.0)


PLAYER_ROSTERS = {
    773: [  # France
        {"id": 77301, "name": "M. Maignan", "position": "GK", "overall_rating": 87, "pace": 83, "shooting": 80, "passing": 85, "dribbling": 84, "defending": 85, "physical": 86},
        {"id": 77302, "name": "J. Koundé", "position": "RB", "overall_rating": 85, "pace": 84, "shooting": 45, "passing": 75, "dribbling": 78, "defending": 86, "physical": 78},
        {"id": 77303, "name": "W. Saliba", "position": "CB", "overall_rating": 88, "pace": 82, "shooting": 38, "passing": 72, "dribbling": 74, "defending": 89, "physical": 85},
        {"id": 77304, "name": "I. Konaté", "position": "CB", "overall_rating": 84, "pace": 78, "shooting": 35, "passing": 65, "dribbling": 68, "defending": 85, "physical": 86},
        {"id": 77305, "name": "T. Hernández", "position": "LB", "overall_rating": 86, "pace": 93, "shooting": 72, "passing": 78, "dribbling": 82, "defending": 79, "physical": 83},
        {"id": 77306, "name": "A. Tchouaméni", "position": "CDM", "overall_rating": 86, "pace": 75, "shooting": 74, "passing": 81, "dribbling": 80, "defending": 85, "physical": 84},
        {"id": 77307, "name": "E. Camavinga", "position": "CM", "overall_rating": 83, "pace": 80, "shooting": 68, "passing": 80, "dribbling": 84, "defending": 80, "physical": 81},
        {"id": 77308, "name": "O. Dembélé", "position": "RW", "overall_rating": 86, "pace": 93, "shooting": 78, "passing": 81, "dribbling": 90, "defending": 36, "physical": 60},
        {"id": 77309, "name": "A. Griezmann", "position": "CAM", "overall_rating": 88, "pace": 78, "shooting": 87, "passing": 88, "dribbling": 87, "defending": 58, "physical": 72},
        {"id": 77310, "name": "K. Mbappé", "position": "ST", "overall_rating": 91, "pace": 97, "shooting": 90, "passing": 80, "dribbling": 92, "defending": 36, "physical": 78},
        {"id": 77311, "name": "R. Kolo Muani", "position": "ST", "overall_rating": 82, "pace": 88, "shooting": 80, "passing": 73, "dribbling": 82, "defending": 40, "physical": 76},
    ],
    770: [  # England
        {"id": 77001, "name": "J. Pickford", "position": "GK", "overall_rating": 83, "pace": 81, "shooting": 78, "passing": 84, "dribbling": 80, "defending": 81, "physical": 82},
        {"id": 77002, "name": "K. Walker", "position": "RB", "overall_rating": 84, "pace": 90, "shooting": 63, "passing": 76, "dribbling": 77, "defending": 83, "physical": 82},
        {"id": 77003, "name": "J. Stones", "position": "CB", "overall_rating": 86, "pace": 72, "shooting": 52, "passing": 80, "dribbling": 80, "defending": 87, "physical": 80},
        {"id": 77004, "name": "M. Guéhi", "position": "CB", "overall_rating": 82, "pace": 75, "shooting": 40, "passing": 68, "dribbling": 70, "defending": 83, "physical": 81},
        {"id": 77005, "name": "L. Shaw", "position": "LB", "overall_rating": 82, "pace": 79, "shooting": 60, "passing": 79, "dribbling": 79, "defending": 80, "physical": 78},
        {"id": 77006, "name": "D. Rice", "position": "CDM", "overall_rating": 87, "pace": 76, "shooting": 68, "passing": 82, "dribbling": 80, "defending": 86, "physical": 85},
        {"id": 77007, "name": "J. Bellingham", "position": "CAM", "overall_rating": 90, "pace": 80, "shooting": 86, "passing": 85, "dribbling": 88, "defending": 78, "physical": 84},
        {"id": 77008, "name": "B. Saka", "position": "RW", "overall_rating": 87, "pace": 86, "shooting": 83, "passing": 83, "dribbling": 87, "defending": 65, "physical": 75},
        {"id": 77009, "name": "P. Foden", "position": "CAM", "overall_rating": 88, "pace": 85, "shooting": 85, "passing": 87, "dribbling": 90, "defending": 56, "physical": 62},
        {"id": 77010, "name": "C. Palmer", "position": "CAM", "overall_rating": 85, "pace": 82, "shooting": 84, "passing": 85, "dribbling": 86, "defending": 48, "physical": 68},
        {"id": 77011, "name": "H. Kane", "position": "ST", "overall_rating": 90, "pace": 69, "shooting": 93, "passing": 84, "dribbling": 83, "defending": 47, "physical": 82},
    ],
    760: [  # Spain
        {"id": 76001, "name": "U. Simón", "position": "GK", "overall_rating": 84, "pace": 82, "shooting": 79, "passing": 81, "dribbling": 81, "defending": 83, "physical": 81},
        {"id": 76002, "name": "D. Carvajal", "position": "RB", "overall_rating": 86, "pace": 81, "shooting": 58, "passing": 78, "dribbling": 80, "defending": 84, "physical": 82},
        {"id": 76003, "name": "R. Le Normand", "position": "CB", "overall_rating": 82, "pace": 70, "shooting": 42, "passing": 68, "dribbling": 68, "defending": 84, "physical": 82},
        {"id": 76004, "name": "A. Laporte", "position": "CB", "overall_rating": 84, "pace": 68, "shooting": 50, "passing": 74, "dribbling": 72, "defending": 85, "physical": 81},
        {"id": 76005, "name": "M. Cucurella", "position": "LB", "overall_rating": 82, "pace": 80, "shooting": 62, "passing": 76, "dribbling": 78, "defending": 80, "physical": 78},
        {"id": 76006, "name": "Rodri", "position": "CDM", "overall_rating": 91, "pace": 66, "shooting": 75, "passing": 86, "dribbling": 84, "defending": 87, "physical": 85},
        {"id": 76007, "name": "Pedri", "position": "CM", "overall_rating": 86, "pace": 78, "shooting": 70, "passing": 88, "dribbling": 89, "defending": 68, "physical": 66},
        {"id": 76008, "name": "Dani Olmo", "position": "CAM", "overall_rating": 85, "pace": 78, "shooting": 82, "passing": 84, "dribbling": 86, "defending": 52, "physical": 66},
        {"id": 76009, "name": "Lamine Yamal", "position": "RW", "overall_rating": 87, "pace": 90, "shooting": 82, "passing": 85, "dribbling": 91, "defending": 42, "physical": 60},
        {"id": 76010, "name": "N. Williams", "position": "LW", "overall_rating": 85, "pace": 93, "shooting": 78, "passing": 79, "dribbling": 87, "defending": 45, "physical": 68},
        {"id": 76011, "name": "A. Morata", "position": "ST", "overall_rating": 83, "pace": 81, "shooting": 82, "passing": 72, "dribbling": 78, "defending": 35, "physical": 78},
    ],
    762: [  # Argentina
        {"id": 76201, "name": "E. Martínez", "position": "GK", "overall_rating": 87, "pace": 85, "shooting": 82, "passing": 85, "dribbling": 85, "defending": 86, "physical": 87},
        {"id": 76202, "name": "N. Molina", "position": "RB", "overall_rating": 82, "pace": 85, "shooting": 64, "passing": 74, "dribbling": 78, "defending": 79, "physical": 76},
        {"id": 76203, "name": "C. Romero", "position": "CB", "overall_rating": 85, "pace": 76, "shooting": 45, "passing": 65, "dribbling": 68, "defending": 86, "physical": 85},
        {"id": 76204, "name": "N. Otamendi", "position": "CB", "overall_rating": 82, "pace": 60, "shooting": 52, "passing": 64, "dribbling": 62, "defending": 83, "physical": 82},
        {"id": 76205, "name": "M. Acuña", "position": "LB", "overall_rating": 81, "pace": 78, "shooting": 70, "passing": 78, "dribbling": 80, "defending": 79, "physical": 81},
        {"id": 76206, "name": "R. De Paul", "position": "CM", "overall_rating": 84, "pace": 78, "shooting": 76, "passing": 82, "dribbling": 82, "defending": 79, "physical": 83},
        {"id": 76207, "name": "E. Fernández", "position": "CM", "overall_rating": 84, "pace": 74, "shooting": 76, "passing": 84, "dribbling": 82, "defending": 78, "physical": 78},
        {"id": 76208, "name": "A. Mac Allister", "position": "CM", "overall_rating": 85, "pace": 74, "shooting": 78, "passing": 84, "dribbling": 84, "defending": 77, "physical": 76},
        {"id": 76209, "name": "L. Messi", "position": "RW", "overall_rating": 88, "pace": 79, "shooting": 87, "passing": 90, "dribbling": 92, "defending": 33, "physical": 64},
        {"id": 76210, "name": "J. Álvarez", "position": "ST", "overall_rating": 85, "pace": 85, "shooting": 84, "passing": 78, "dribbling": 85, "defending": 55, "physical": 78},
        {"id": 76211, "name": "L. Martínez", "position": "ST", "overall_rating": 89, "pace": 82, "shooting": 88, "passing": 75, "dribbling": 85, "defending": 48, "physical": 84},
    ]
}


def seed_standings(conn):
    now = datetime.now(timezone.utc).isoformat()
    for group_letter, entries in STANDINGS_TEMPLATE.items():
        for pos, entry in enumerate(entries, 1):
            elo_val = TEAM_ELOS.get(entry["id"], 1800.0)
            conn.execute(
                """INSERT INTO dashboard_team (id, name, short_name, "group", pre_match_elo)
                   VALUES (?, ?, ?, ?, ?)
                   ON CONFLICT(id) DO UPDATE SET pre_match_elo = excluded.pre_match_elo""",
                (entry["id"], entry["name"], entry["name"][:3].upper(), f"GROUP_{group_letter}", elo_val),
            )
            conn.execute(
                """INSERT INTO dashboard_standing
                   (team_id, "group", position, played, won, drawn, lost, goals_for, goals_against, points, updated_at)
                   VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
                   ON CONFLICT(team_id, "group") DO UPDATE SET
                       position = excluded.position, played = excluded.played, won = excluded.won,
                       drawn = excluded.drawn, lost = excluded.lost, goals_for = excluded.goals_for,
                       goals_against = excluded.goals_against, points = excluded.points""",
                (entry["id"], group_letter, pos, entry["played"], entry["won"], entry["drawn"],
                 entry["lost"], entry["gf"], entry["ga"], entry["pts"], now),
            )
    conn.commit()
    print("Standings seeded.")


def seed_players(conn):
    for team_id, players in PLAYER_ROSTERS.items():
        for pdata in players:
            conn.execute(
                """INSERT INTO dashboard_player
                   (id, team_id, name, position, overall_rating, pace, shooting, passing,
                    dribbling, defending, physical)
                   VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
                   ON CONFLICT(id) DO NOTHING""",
                (pdata["id"], team_id, pdata["name"], pdata["position"], pdata["overall_rating"],
                 pdata["pace"], pdata["shooting"], pdata["passing"], pdata["dribbling"],
                 pdata["defending"], pdata["physical"]),
            )
    conn.commit()
    print("Player ratings seeded.")


def load_dotenv(path: Path):
    import os

    try:
        lines = path.read_text(encoding="utf-8").splitlines()
    except OSError:
        return
    for line in lines:
        line = line.strip()
        if not line or line.startswith("#") or "=" not in line:
            continue
        key, _, value = line.partition("=")
        os.environ.setdefault(key.strip(), value.strip().strip("'\""))


def fd_get(api_key: str, path: str):
    req = urllib.request.Request(
        f"{FD_BASE}{path}", headers={"X-Auth-Token": api_key}
    )
    with urllib.request.urlopen(req, timeout=30) as resp:
        return json.load(resp)


def main(match_id: int, groups_only: bool):
    import os

    load_dotenv(ROOT / ".env")
    api_key = os.getenv("FOOTBALL_DATA_API_KEY")
    if not api_key and not groups_only:
        print("FOOTBALL_DATA_API_KEY not set in environment or .env")
        return

    conn = sqlite3.connect(DB_PATH)
    try:
        conn.executescript(SCHEMA)

        if not groups_only:
            conn.execute("DELETE FROM dashboard_winprobabilitysnapshot WHERE match_id = ?", (match_id,))
            conn.execute("DELETE FROM dashboard_matchevent WHERE match_id = ?", (match_id,))
            conn.execute("DELETE FROM dashboard_match WHERE id = ?", (match_id,))
            conn.commit()

            row = conn.execute("SELECT id FROM dashboard_matchconfig LIMIT 1").fetchone()
            now = datetime.now(timezone.utc).isoformat()
            if row:
                conn.execute("UPDATE dashboard_matchconfig SET current_match_id = ?, updated_at = ? WHERE id = ?",
                             (match_id, now, row[0]))
            else:
                conn.execute("INSERT INTO dashboard_matchconfig (current_match_id, updated_at) VALUES (?, ?)",
                             (match_id, now))
            conn.commit()

        seed_players(conn)
        seed_standings(conn)

        if groups_only:
            print("\nDone! Groups and standings seeded.")
            print("  Run the script again without --groups-only to seed a specific match.")
            return

        print(f"Fetching match {match_id} from football-data.org...")
        try:
            m = fd_get(api_key, f"/matches/{match_id}")
        except Exception as e:
            print(f"API error: {e}")
            return

        home_team_data = m["homeTeam"]
        away_team_data = m["awayTeam"]
        score = m["score"]
        ft = score.get("fullTime", {})
        ht = score.get("halfTime", {})
        home_score = ft.get("home", 0)
        away_score = ft.get("away", 0)
        ht_home = ht.get("home", 0)
        ht_away = ht.get("away", 0)
        stage = m.get("stage", "GROUP_STAGE")

        venue_api = m.get("venue")
        if venue_api:
            venue = venue_api
        elif match_id == 537390 or stage == "FINAL":
            venue = "MetLife Stadium (New York / New Jersey)"
        elif match_id == 537389 or stage == "THIRD_PLACE":
            venue = "Hard Rock Stadium (Miami)"
        elif match_id == 537388:
            venue = "AT&T Stadium (Dallas)"
        elif match_id == 537387:
            venue = "Mercedes-Benz Stadium (Atlanta)"
        elif stage == "SEMI_FINALS":
            venue = "AT&T Stadium (Dallas)"
        else:
            venue = "MetLife Stadium"
        kickoff_utc = m.get("utcDate", "2026-07-19T19:00:00Z").replace("Z", "").replace("T", " ")

        print(f"Match: {home_team_data['name']} {home_score} vs {away_score} {away_team_data['name']}")
        print(f"  Stage: {stage}  |  HT: {ht_home}-{ht_away}  |  Venue: {venue}")

        if match_id in KNOWN_EVENTS:
            events = KNOWN_EVENTS[match_id]
            print("  Using predefined event timeline.")
        else:
            events = generate_events(home_score, away_score, ht_home, ht_away)
            print("  Generating event timeline from scoreline.")

        for td in [home_team_data, away_team_data]:
            elo_val = TEAM_ELOS.get(td["id"], 1800.0)
            conn.execute(
                """INSERT INTO dashboard_team (id, name, short_name, pre_match_elo)
                   VALUES (?, ?, ?, ?)
                   ON CONFLICT(id) DO UPDATE SET pre_match_elo = excluded.pre_match_elo""",
                (td["id"], td["name"], td.get("shortName", td["name"][:3].upper()), elo_val),
            )
        conn.commit()
        print("Teams seeded.")

        conn.execute(
            """INSERT INTO dashboard_match
               (id, home_team_id, away_team_id, kickoff_utc, stage, venue, status, home_score, away_score)
               VALUES (?, ?, ?, ?, ?, ?, 'FINISHED', ?, ?)""",
            (match_id, home_team_data["id"], away_team_data["id"], kickoff_utc,
             stage, venue or "MetLife Stadium", home_score, away_score),
        )
        conn.commit()
        print("Match seeded.")

        now = datetime.now(timezone.utc).isoformat()
        for evt in events["yellow_cards"]:
            team_id = home_team_data["id"] if evt["team"] == "home" else away_team_data["id"]
            conn.execute(
                """INSERT OR IGNORE INTO dashboard_matchevent
                   (match_id, minute, event_type, team_id, player_name, assist_name, detail, created_at)
                   VALUES (?, ?, 'YELLOW_CARD', ?, ?, '', 'Yellow Card', ?)""",
                (match_id, evt["minute"], team_id, evt["player"], now),
            )

        for evt in events["red_cards"]:
            team_id = home_team_data["id"] if evt["team"] == "home" else away_team_data["id"]
            conn.execute(
                """INSERT OR IGNORE INTO dashboard_matchevent
                   (match_id, minute, event_type, team_id, player_name, assist_name, detail, created_at)
                   VALUES (?, ?, 'RED_CARD', ?, ?, '', 'Red Card', ?)""",
                (match_id, evt["minute"], team_id, evt["player"], now),
            )

        for evt in events["substitutions"]:
            team_id = home_team_data["id"] if evt["team"] == "home" else away_team_data["id"]
            conn.execute(
                """INSERT OR IGNORE INTO dashboard_matchevent
                   (match_id, minute, event_type, team_id, player_name, assist_name, detail, created_at)
                   VALUES (?, ?, 'SUBSTITUTION', ?, ?, '', 'Substitution', ?)""",
                (match_id, evt["minute"], team_id,
                 f"{evt['player_on']} ← {evt['player_off']}", now),
            )

        for evt in events["goals"]:
            team_id = home_team_data["id"] if evt["team"] == "home" else away_team_data["id"]
            minute = evt["minute"] + evt.get("stoppage", 0)
            conn.execute(
                """INSERT OR IGNORE INTO dashboard_matchevent
                   (match_id, minute, event_type, team_id, player_name, assist_name, detail, created_at)
                   VALUES (?, ?, 'GOAL', ?, ?, '', 'Goal', ?)""",
                (match_id, minute, team_id, evt["player"], now),
            )

        conn.commit()
        print(f"Events seeded: {len(events['goals'])} goals, {len(events['yellow_cards'])} yellows, "
              f"{len(events['red_cards'])} reds, {len(events['substitutions'])} subs.")

        elo_diff = get_elo_diff(conn, home_team_data["id"], away_team_data["id"])

        print("Pre-computing win probability snapshots for minutes 0-95...")
        features_list = []
        minute_list = []
        for minute in range(0, 96):
            h, a = get_score_at_minute(minute, events["goals"])
            score_diff = h - a
            xg_diff = float(score_diff) * 0.75 + (elo_diff / 1000.0)
            features_list.append(
                build_game_state_features(
                    score_diff=score_diff,
                    minute=minute,
                    xg_diff=xg_diff,
                    pre_match_elo_diff=elo_diff,
                    red_card_diff=0,
                )
            )
            minute_list.append((minute, h, a, score_diff, xg_diff))

        try:
            model = joblib.load(MODEL_PATH)
            probs_batch = model.predict_proba(features_list)
            classes = model.classes_.tolist()
        except FileNotFoundError:
            probs_batch = [[0.45, 0.10, 0.45] for _ in range(96)]
            classes = [-1, 0, 1]

        for i, (minute, h, a, score_diff, xg_diff) in enumerate(minute_list):
            probs = dict(zip(classes, probs_batch[i].tolist()))
            conn.execute(
                """INSERT INTO dashboard_winprobabilitysnapshot
                   (match_id, minute, home_win_prob, draw_prob, away_win_prob,
                    score_diff, xg_diff_approx, created_at)
                   VALUES (?, ?, ?, ?, ?, ?, ?, ?)""",
                (match_id, minute, probs.get(1, 0.45), probs.get(0, 0.10),
                 probs.get(-1, 0.45), score_diff, xg_diff, now),
            )

        conn.commit()
        print("96 win probability snapshots seeded.")

        print("\nDone! Replay match is ready.")
        print(f"  {home_team_data['name']} vs {away_team_data['name']}")
        print(f"  Final score: {home_score} - {away_score}")
    finally:
        conn.close()


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description="Seed a WC 2026 replay match")
    parser.add_argument("--match-id", type=int, default=DEFAULT_MATCH_ID,
                        help=f"football-data.org match ID (default: {DEFAULT_MATCH_ID}, France 4-6 England)")
    parser.add_argument("--groups-only", action="store_true",
                        help="Only seed group teams and standings, skip match seeding")
    args = parser.parse_args()
    main(match_id=args.match_id, groups_only=args.groups_only)
