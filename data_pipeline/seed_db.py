"""
Run once before tournament: python data_pipeline/seed_db.py
Requires: migrations already applied, FOOTBALL_DATA_API_KEY in .env
Expected runtime: ~10 minutes (pacing SoFIFA requests)
"""
import os
import sys
import time
import django
import requests
from datetime import datetime, timezone

sys.path.append(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
os.environ.setdefault('DJANGO_SETTINGS_MODULE', 'wc2026.settings')
django.setup()

from dashboard.models import Team, Player, Match
from dotenv import load_dotenv
load_dotenv()

FD_KEY = os.getenv('FOOTBALL_DATA_API_KEY')
FD_HEADERS = {'X-Auth-Token': FD_KEY}
FD_BASE = 'https://api.football-data.org/v4'

FD_TO_SOFIFA = {}


def seed_teams_and_matches():
    if not FD_KEY:
        print("  FOOTBALL_DATA_API_KEY not set — skipping team/match seeding.")
        print("  Set it in .env and re-run when ready.")
        return

    print("Fetching WC 2026 schedule...")
    resp = requests.get(
        f'{FD_BASE}/competitions/WC/matches',
        headers=FD_HEADERS,
        params={'season': 2026}
    )

    remaining = resp.headers.get('x-requests-available', '?')
    print(f"  API requests remaining this window: {remaining}")

    if resp.status_code == 403:
        print("  ERROR: API returned 403 — your FOOTBALL_DATA_API_KEY is")
        print("  missing or invalid. Get a free key at")
        print("  https://www.football-data.org/client")
        print("  Then add it to .env and re-run.")
        return
    if resp.status_code == 429:
        retry_after = resp.headers.get('Retry-After', '60')
        print(f"  Rate limited — retrying after {retry_after}s")
        time.sleep(int(retry_after))
        resp = requests.get(
            f'{FD_BASE}/competitions/WC/matches',
            headers=FD_HEADERS,
            params={'season': 2026}
        )
    if resp.status_code != 200:
        print(f"  ERROR: Unexpected status {resp.status_code} — {resp.text[:200]}")
        return

    data = resp.json()

    team_ids_seen = set()

    for m in data['matches']:
        for side in ['homeTeam', 'awayTeam']:
            t = m[side]
            if t['id'] not in team_ids_seen:
                Team.objects.get_or_create(
                    id=t['id'],
                    defaults={
                        'name': t['name'],
                        'short_name': t.get('shortName', t['name'][:3].upper()),
                    }
                )
                team_ids_seen.add(t['id'])

        kickoff = datetime.fromisoformat(
            m['utcDate'].replace('Z', '+00:00')
        )
        Match.objects.get_or_create(
            id=m['id'],
            defaults={
                'home_team_id': m['homeTeam']['id'],
                'away_team_id': m['awayTeam']['id'],
                'kickoff_utc': kickoff,
                'stage': m['stage'],
                'venue': m.get('venue', ''),
                'status': m['status'],
            }
        )

    print(f"  {len(team_ids_seen)} teams, {len(data['matches'])} matches seeded.")


def seed_player_ratings():
    if not FD_KEY:
        print("  FOOTBALL_DATA_API_KEY not set — skipping player ratings.")
        print("  Set it in .env and re-run when ready.")
        return

    print("Fetching SoFIFA FC 26 ratings for all squads...")
    teams = Team.objects.all()

    for team in teams:
        sofifa_id = FD_TO_SOFIFA.get(team.id)
        if not sofifa_id:
            print(f"  Skipping {team.name} — no SoFIFA mapping")
            continue

        resp = requests.get(
            'https://sofifa.com/api/players',
            params={'team_id': sofifa_id, 'hl': 'en-US'}
        )

        if resp.status_code == 429:
            print("  Rate limited — waiting 60s")
            time.sleep(60)
            resp = requests.get(
                'https://sofifa.com/api/players',
                params={'team_id': sofifa_id, 'hl': 'en-US'}
            )

        if resp.status_code == 200:
            players = resp.json().get('data', [])
            for p in players:
                Player.objects.get_or_create(
                    id=p['id'],
                    defaults={
                        'team': team,
                        'name': p['name'],
                        'position': (p.get('positions') or [''])[0],
                        'overall_rating': p.get('overallRating', 0),
                        'pace': p.get('pace', 0),
                        'shooting': p.get('shooting', 0),
                        'passing': p.get('passing', 0),
                        'dribbling': p.get('dribbling', 0),
                        'defending': p.get('defending', 0),
                        'physical': p.get('physic', 0),
                        'skill_moves': p.get('skillMoves', 0),
                        'weak_foot': p.get('weakFoot', 0),
                    }
                )
            print(f"  {team.name}: {len(players)} players")

        time.sleep(1.5)


if __name__ == '__main__':
    seed_teams_and_matches()
    seed_player_ratings()
    print("\nDatabase seeding complete.")
