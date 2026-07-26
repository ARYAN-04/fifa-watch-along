"""
Seed database with one completed WC 2026 match and generate a realistic
event timeline + per-minute win probability snapshots for replay.

Usage:
  uv run python scripts/seed_replay_match.py                                  # France 4-6 England (default)
  uv run python scripts/seed_replay_match.py --match-id 537390                 # Final: Spain 1-0 Argentina
  uv run python scripts/seed_replay_match.py --groups-only                     # Just seed group teams/standings
"""
import os
import sys
import argparse
import requests
import random
from datetime import datetime, timezone

sys.path.insert(0, os.path.join(os.path.dirname(__file__), ".."))

from dotenv import load_dotenv
dotenv_path = os.path.join(os.path.dirname(__file__), "..", ".env")
load_dotenv(dotenv_path)

from api.db import SessionLocal, engine
from api.models import Base
from api.models import Team, Player, Match, MatchEvent, MatchConfig, Standing, WinProbabilitySnapshot
from api.services.inference import get_model
from ml.features import build_game_state_features

FD_API_KEY = os.getenv("FOOTBALL_DATA_API_KEY")
FD_HEADERS = {"X-Auth-Token": FD_API_KEY}
FD_BASE = "https://api.football-data.org/v4"

DEFAULT_MATCH_ID = 537389

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
    home_name_short = "H"
    away_name_short = "A"

    goals = []
    yellow_cards = []
    red_cards = []
    substitutions = []

    first_half_goals_home = ht_home
    first_half_goals_away = ht_away
    second_half_goals_home = home_score - ht_home
    second_half_goals_away = away_score - ht_away

    first_half_slots = [8, 18, 25, 32, 38, 42, 45]
    second_half_slots = [48, 55, 62, 68, 75, 82, 88]
    stoppage_slots = [90]

    random.shuffle(first_half_slots)
    random.shuffle(second_half_slots)

    home_first = first_half_goals_home + second_half_goals_home + (1 if home_score > 4 else 0)
    away_first = first_half_goals_away + second_half_goals_away + (1 if away_score > 4 else 0)

    idx = 0
    for _ in range(first_half_goals_home):
        if idx < len(first_half_slots):
            goals.append({"minute": first_half_slots[idx], "team": "home", "player": f"Player {home_name_short}{idx+1}"})
            idx += 1
    idx = 0
    for _ in range(first_half_goals_away):
        if idx < len(first_half_slots):
            goals.append({"minute": first_half_slots[len(first_half_slots)-1-idx], "team": "away", "player": f"Player {away_name_short}{idx+1}"})
            idx += 1

    idx = 0
    for _ in range(second_half_goals_home):
        if idx < len(second_half_slots):
            goals.append({"minute": second_half_slots[idx], "team": "home", "player": f"Player {home_name_short}{idx+1}"})
            idx += 1
    idx = 0
    for _ in range(second_half_goals_away):
        if idx < len(second_half_slots):
            goals.append({"minute": second_half_slots[len(second_half_slots)-1-idx], "team": "away", "player": f"Player {away_name_short}{idx+1}"})
            idx += 1

    extra_goals_home = max(0, second_half_goals_home - len(second_half_slots))
    extra_goals_away = max(0, second_half_goals_away - len(second_half_slots))
    for i in range(extra_goals_home):
        goals.append({"minute": 90, "stoppage": i + 1, "team": "home", "player": f"Player H-ET{i+1}"})
    for i in range(extra_goals_away):
        goals.append({"minute": 90, "stoppage": i + 1, "team": "away", "player": f"Player A-ET{i+1}"})

    card_slots = [15, 30, 40, 50, 65, 80]
    num_yellows = min(5, len(card_slots))
    for i in range(num_yellows):
        team = "home" if i % 2 == 0 else "away"
        yellow_cards.append({"minute": card_slots[i], "team": team, "player": f"Player {team.upper()}-YC{i+1}"})

    sub_slots = [55, 65, 70, 75, 80, 85]
    num_subs = min(6, len(sub_slots))
    for i in range(num_subs):
        team = "home" if i % 2 == 0 else "away"
        substitutions.append({
            "minute": sub_slots[i], "team": team,
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


def get_score_at_minute(minute: int, goals: list) -> tuple[int, int]:
    home = 0
    away = 0
    for g in goals:
        m = g["minute"] + g.get("stoppage", 0)
        if m <= minute:
            if g["team"] == "home":
                home += 1
            else:
                away += 1
    return home, away


def get_elo_diff_for_teams(home_team_id: int, away_team_id: int, db) -> float:
    home_team = db.query(Team).filter(Team.id == home_team_id).first()
    away_team = db.query(Team).filter(Team.id == away_team_id).first()
    home_elo = home_team.pre_match_elo if home_team else 1500.0
    away_elo = away_team.pre_match_elo if away_team else 1500.0
    return home_elo - away_elo


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
        {"id": 76011, "name": "A. Morata", "position": "ST", "overall_rating": 83, "pace": 81, "shooting": 82, "passing": 72, "dribbling": 78, "defending": 35, "physical": 76},
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


def seed_standings(db):
    for group_letter, entries in STANDINGS_TEMPLATE.items():
        for pos, entry in enumerate(entries, 1):
            existing = db.query(Team).filter(Team.id == entry["id"]).first()
            elo_val = TEAM_ELOS.get(entry["id"], 1800.0)
            if not existing:
                db.add(Team(
                    id=entry["id"],
                    name=entry["name"],
                    short_name=entry["name"][:3].upper(),
                    group=f"GROUP_{group_letter}",
                    pre_match_elo=elo_val,
                ))
            else:
                existing.pre_match_elo = elo_val

            st = db.query(Standing).filter(Standing.team_id == entry["id"]).first()
            if not st:
                db.add(Standing(
                    team_id=entry["id"],
                    group=group_letter,
                    position=pos,
                    played=entry["played"],
                    won=entry["won"],
                    drawn=entry["drawn"],
                    lost=entry["lost"],
                    goals_for=entry["gf"],
                    goals_against=entry["ga"],
                    points=entry["pts"],
                ))
            else:
                st.group = group_letter
                st.position = pos
                st.played = entry["played"]
                st.won = entry["won"]
                st.drawn = entry["drawn"]
                st.lost = entry["lost"]
                st.goals_for = entry["gf"]
                st.goals_against = entry["ga"]
                st.points = entry["pts"]
    db.commit()
    print("Standings seeded.")


def seed_players(db):
    for team_id, players in PLAYER_ROSTERS.items():
        for pdata in players:
            existing = db.query(Player).filter(Player.id == pdata["id"]).first()
            if existing:
                continue
            db.add(Player(
                id=pdata["id"],
                team_id=team_id,
                name=pdata["name"],
                position=pdata["position"],
                overall_rating=pdata["overall_rating"],
                pace=pdata["pace"],
                shooting=pdata["shooting"],
                passing=pdata["passing"],
                dribbling=pdata["dribbling"],
                defending=pdata["defending"],
                physical=pdata["physical"],
            ))
    db.commit()
    print("Player ratings seeded.")


def main(match_id: int, groups_only: bool):
    if not FD_API_KEY:
        print("FOOTBALL_DATA_API_KEY not set in .env")
        return

    Base.metadata.create_all(bind=engine)
    db = SessionLocal()

    try:
        if not groups_only:
            db.query(WinProbabilitySnapshot).filter(WinProbabilitySnapshot.match_id == match_id).delete()
            db.query(MatchEvent).filter(MatchEvent.match_id == match_id).delete()
            db.query(Match).filter(Match.id == match_id).delete()
            db.commit()

            existing_config = db.query(MatchConfig).first()
            if existing_config:
                existing_config.current_match_id = match_id
            else:
                db.add(MatchConfig(current_match_id=match_id))
            db.commit()

        seed_players(db)
        seed_standings(db)

        if groups_only:
            print("\nDone! Groups and standings seeded.")
            print("  Run the script again without --groups-only to seed a specific match.")
            return

        print(f"Fetching match {match_id} from football-data.org...")
        resp = requests.get(f"{FD_BASE}/matches/{match_id}", headers=FD_HEADERS)
        if resp.status_code != 200:
            print(f"API error: {resp.status_code} — {resp.text[:200]}")
            return

        m = resp.json()
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
        
        # Resolve venue (API or official WC 2026 venue mapping)
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
        kickoff_utc_str = m.get("utcDate", "2026-07-19T19:00:00Z")
        kickoff_utc = datetime.fromisoformat(kickoff_utc_str.replace("Z", "+00:00"))

        print(f"Match: {home_team_data['name']} {home_score} vs {away_score} {away_team_data['name']}")
        print(f"  Stage: {stage}  |  HT: {ht_home}-{ht_away}  |  Venue: {venue}")

        if match_id in KNOWN_EVENTS:
            events = KNOWN_EVENTS[match_id]
            print("  Using predefined event timeline.")
        else:
            events = generate_events(home_score, away_score, ht_home, ht_away)
            print("  Generating event timeline from scoreline.")

        for td in [home_team_data, away_team_data]:
            existing = db.query(Team).filter(Team.id == td["id"]).first()
            elo_val = TEAM_ELOS.get(td["id"], 1800.0)
            if existing:
                existing.pre_match_elo = elo_val
                continue
            db.add(Team(
                id=td["id"],
                name=td["name"],
                short_name=td.get("shortName", td["name"][:3].upper()),
                flag_url="",
                group=m.get("group", ""),
                pre_match_elo=elo_val,
                fc26_overall=None,
            ))
        db.commit()
        print("Teams seeded.")

        match = Match(
            id=match_id,
            home_team_id=home_team_data["id"],
            away_team_id=away_team_data["id"],
            kickoff_utc=kickoff_utc,
            stage=stage,
            venue=venue or "MetLife Stadium",
            status="FINISHED",
            home_score=home_score,
            away_score=away_score,
        )
        db.add(match)
        db.commit()
        print("Match seeded.")

        for evt in events["yellow_cards"]:
            team_id = home_team_data["id"] if evt["team"] == "home" else away_team_data["id"]
            db.add(MatchEvent(
                match_id=match_id,
                minute=evt["minute"],
                event_type="YELLOW_CARD",
                team_id=team_id,
                player_name=evt["player"],
                assist_name="",
                detail="Yellow Card",
            ))

        for evt in events["red_cards"]:
            team_id = home_team_data["id"] if evt["team"] == "home" else away_team_data["id"]
            db.add(MatchEvent(
                match_id=match_id,
                minute=evt["minute"],
                event_type="RED_CARD",
                team_id=team_id,
                player_name=evt["player"],
                assist_name="",
                detail="Red Card",
            ))

        for evt in events["substitutions"]:
            team_id = home_team_data["id"] if evt["team"] == "home" else away_team_data["id"]
            db.add(MatchEvent(
                match_id=match_id,
                minute=evt["minute"],
                event_type="SUBSTITUTION",
                team_id=team_id,
                player_name=f"{evt['player_on']} ← {evt['player_off']}",
                assist_name="",
                detail="Substitution",
            ))

        for evt in events["goals"]:
            team_id = home_team_data["id"] if evt["team"] == "home" else away_team_data["id"]
            minute = evt["minute"] + evt.get("stoppage", 0)
            db.add(MatchEvent(
                match_id=match_id,
                minute=minute,
                event_type="GOAL",
                team_id=team_id,
                player_name=evt["player"],
                assist_name="",
                detail="Goal",
            ))

        db.commit()
        print(f"Events seeded: {len(events['goals'])} goals, {len(events['yellow_cards'])} yellows, "
              f"{len(events['red_cards'])} reds, {len(events['substitutions'])} subs.")

        elo_diff = get_elo_diff_for_teams(home_team_data["id"], away_team_data["id"], db)

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
            model = get_model()
            probs_batch = model.predict_proba(features_list)
            classes = model.classes_.tolist()
        except FileNotFoundError:
            probs_batch = [[0.45, 0.10, 0.45] for _ in range(96)]
            classes = [-1, 0, 1]

        for i, (minute, h, a, score_diff, xg_diff) in enumerate(minute_list):
            probs = dict(zip(classes, probs_batch[i].tolist()))
            db.add(WinProbabilitySnapshot(
                match_id=match_id,
                minute=minute,
                home_win_prob=probs.get(1, 0.45),
                draw_prob=probs.get(0, 0.10),
                away_win_prob=probs.get(-1, 0.45),
                score_diff=score_diff,
                xg_diff_approx=xg_diff,
            ))

        db.commit()
        print("96 win probability snapshots seeded.")

        print("\nDone! Replay match is ready.")
        print(f"  {home_team_data['name']} vs {away_team_data['name']}")
        print(f"  Final score: {home_score} - {away_score}")

    except Exception as e:
        db.rollback()
        print(f"Error: {e}")
        raise
    finally:
        db.close()


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description="Seed a WC 2026 replay match")
    parser.add_argument("--match-id", type=int, default=DEFAULT_MATCH_ID,
                        help=f"football-data.org match ID (default: {DEFAULT_MATCH_ID}, France 4-6 England)")
    parser.add_argument("--groups-only", action="store_true",
                        help="Only seed group teams and standings, skip match seeding")
    args = parser.parse_args()
    main(match_id=args.match_id, groups_only=args.groups_only)
