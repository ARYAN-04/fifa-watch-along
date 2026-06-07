# FIFA World Cup 2026 Watch-Along Dashboard
## Complete Implementation & Deployment Guide (Django)

---

## Table of Contents

1. [Project Overview](#1-project-overview)
2. [Architecture Overview](#2-architecture-overview)
3. [Repository & Environment Setup](#3-repository--environment-setup)
4. [Data Sources Reference](#4-data-sources-reference)
5. [Phase 1 — ML Win Probability Model](#5-phase-1--ml-win-probability-model)
6. [Phase 2 — Pre-Match Data Pipeline](#6-phase-2--pre-match-data-pipeline)
7. [Phase 3 — Backend (Django)](#7-phase-3--backend-django)
8. [Phase 4 — Frontend (React)](#8-phase-4--frontend-react)
9. [Phase 5 — Deployment](#9-phase-5--deployment)
10. [Match Day Operations](#10-match-day-operations)
11. [Fallback & Error Handling](#11-fallback--error-handling)
12. [Testing Strategy](#12-testing-strategy)

---

## 1. Project Overview

### What You're Building

A real-time watch-along dashboard for the FIFA World Cup 2026, serving a small
audience of friends and LinkedIn visitors. The dashboard shows:

- Live goal events (polled from football-data.org, ~1–2 min delay)
- A **ML-powered win probability graph** that updates on every goal, driven by
  a model trained on StatsBomb 2022 World Cup data
- Player FC 26 ratings (SoFIFA, pre-loaded before kickoff)
- Pre-match context panel: group standings, team ELO ratings, squad profiles
- Tournament schedule and bracket tracker
- **Django Admin** for managing match state without touching the server

### Constraints & Decisions

| Constraint | Decision |
|---|---|
| Free APIs only | football-data.org (free tier), StatsBomb Open Data, SoFIFA API |
| Only goals needed live | Poll football-data.org every 60s — well within 10 req/min free limit |
| World Cup 2026 only | StatsBomb 2022 WC data for ML training; no league data needed |
| Shared publicly | Backend on Render (free), Frontend on Vercel (free) |
| Persistent storage | SQLite via Django ORM with migrations |
| No Polymarket | Win probability entirely from the trained ML model |
| Framework | Django 5 + django-apscheduler for background polling |

### Technology Stack

| Layer | Technology | Why |
|---|---|---|
| Backend | Python 3.11, Django 5 | Familiar, batteries-included, great admin panel |
| Background tasks | django-apscheduler | Runs polling inside Django process, no Redis/Celery needed |
| Database | SQLite + Django ORM | Zero-config, migrations handled by Django |
| ML | scikit-learn | Lightweight, serialisable, no GPU needed |
| Data loading | statsbombpy, requests | Official StatsBomb Python client + sync HTTP |
| Production server | Gunicorn | Standard Django production server |
| Frontend | React 18, Recharts | Component-based; Recharts for the probability graph |
| Deployment | Render (backend), Vercel (frontend) | Both free, both deploy from GitHub |

### Why Django Over FastAPI for This Project

Django gives you one thing FastAPI doesn't: a free, fully-featured admin UI at
`/admin`. Before each match you set `CURRENT_MATCH_ID` by clicking through the
admin rather than SSH-ing into Render to update an environment variable. You can
also browse every stored goal event, inspect win probability snapshots, and edit
team ELO values without writing a single extra line of code. For a project you're
operating manually, this is a genuine operational advantage.

The tradeoff is that Django is synchronous by default, so background polling uses
`django-apscheduler` (a `BackgroundScheduler` thread) rather than native async
tasks. For a single-match polling use case this is entirely fine.

---

## 2. Architecture Overview

```
┌─────────────────────────────────────────────────────────┐
│                    VERCEL (Frontend)                     │
│  React SPA — polls /api/state every 30s                 │
│  Panels: WinProb Graph │ Event Ticker │ Player Ratings  │
│          Pre-Match Context │ Standings                  │
└────────────────────────┬────────────────────────────────┘
                         │ HTTP (REST — plain JsonResponse)
┌────────────────────────▼────────────────────────────────┐
│                   RENDER (Backend)                       │
│  Django 5 + Gunicorn (1 worker)                         │
│                                                         │
│  ┌──────────────────────────────────────────────────┐   │
│  │ APScheduler (BackgroundScheduler thread)         │   │
│  │ → fires poll_match() every 60s                  │   │
│  │ → calls football-data.org                       │   │
│  │ → writes new goals to SQLite                    │   │
│  │ → runs model inference                          │   │
│  │ → updates in-memory current_state dict          │   │
│  └──────────────────────────────────────────────────┘   │
│                                                         │
│  ┌──────────────────────────────────────────────────┐   │
│  │ Django Views (dashboard/views.py)                │   │
│  │ GET /api/state        → current_state dict       │   │
│  │ GET /api/prematch/    → team + player data       │   │
│  │ GET /api/standings/   → group tables             │   │
│  │ GET /api/matches/     → full schedule            │   │
│  │ GET /health/          → uptime check             │   │
│  └──────────────────────────────────────────────────┘   │
│                                                         │
│  ┌──────────────────────────────────────────────────┐   │
│  │ Django Admin (/admin)                            │   │
│  │ → Set CURRENT_MATCH_ID before each match        │   │
│  │ → Browse goals, probability snapshots           │   │
│  │ → Edit team ELO ratings                         │   │
│  └──────────────────────────────────────────────────┘   │
│                                                         │
│  ┌──────────────────────────────────────────────────┐   │
│  │ SQLite (wc2026.db)                               │   │
│  │ tables: dashboard_team, dashboard_player,        │   │
│  │   dashboard_match, dashboard_matchevent,         │   │
│  │   dashboard_winprobabilitysnapshot,              │   │
│  │   dashboard_standing, dashboard_matchconfig      │   │
│  └──────────────────────────────────────────────────┘   │
│                                                         │
│  ml/win_prob_model.pkl  (loaded at startup)             │
└─────────────────────────────────────────────────────────┘
              │                   │               │
    football-data.org        SoFIFA API     StatsBomb
    (live goals, free)    (FC26 ratings,   Open Data
                           pre-match)      (GitHub,
                                           downloaded once)
```

### Data Flow During a Live Match

1. APScheduler fires `poll_match()` every 60 seconds in a background thread.
2. Calls `api.football-data.org/v4/matches/{match_id}` using the match ID
   stored in the `MatchConfig` table (set via Django Admin before kickoff).
3. Compares returned goals against existing `MatchEvent` rows — inserts any new
   ones, skips duplicates.
4. Runs model inference: feeds (score_diff, minute, xg_diff_approx,
   pre_match_elo_diff, red_card_diff) into the loaded `.pkl` model.
5. Writes a new `WinProbabilitySnapshot` row to SQLite.
6. Updates the module-level `current_state` dict.
7. React frontend polls `/api/state` every 30 seconds, updates graph and ticker.

### Important: Single Worker Constraint

The in-memory `current_state` dict must live in exactly one process. Run Gunicorn
with `--workers 1` on Render's free tier. Multiple workers would each maintain
their own copy of `current_state`, causing stale data on some requests. One worker
is fine for the expected audience size (tens of concurrent visitors).

---

## 3. Repository & Environment Setup

### Directory Structure

```
wc2026-dashboard/
├── wc2026/                        # Django project config
│   ├── __init__.py
│   ├── settings.py
│   ├── urls.py
│   └── wsgi.py
├── dashboard/                     # Main Django app
│   ├── migrations/
│   ├── __init__.py
│   ├── admin.py                   # Admin registrations
│   ├── apps.py                    # Starts scheduler on ready()
│   ├── models.py                  # All ORM models
│   ├── views.py                   # All API endpoints
│   ├── urls.py                    # App URL patterns
│   ├── poller.py                  # Background polling + in-memory state
│   └── inference.py               # Model loading + prediction
├── ml/
│   ├── train.py                   # Run offline to produce .pkl
│   ├── features.py                # Feature engineering (shared by train + inference)
│   ├── evaluate.py                # Calibration checks
│   └── win_prob_model.pkl         # Committed to repo after training
├── data_pipeline/
│   ├── load_statsbomb.py          # Build ML training dataset
│   ├── fetch_sofifa.py            # Fetch player ratings pre-tournament
│   ├── seed_db.py                 # Populate SQLite before tournament
│   └── data/
│       ├── wc2022_game_states.json
│       └── wc2026_squads.json
├── frontend/                      # React app (separate deploy to Vercel)
│   ├── src/
│   │   ├── App.jsx
│   │   ├── api.js
│   │   └── components/
│   │       ├── WinProbGraph.jsx
│   │       ├── EventTicker.jsx
│   │       ├── PlayerRatings.jsx
│   │       ├── PreMatchPanel.jsx
│   │       └── Standings.jsx
│   ├── package.json
│   └── .env.production
├── wc2026.db                      # Seeded SQLite — committed to repo
├── manage.py
├── requirements.txt
├── Dockerfile
├── .env.example
└── .gitignore
```

### Python Environment

```bash
python3.11 -m venv venv
source venv/bin/activate          # Windows: venv\Scripts\activate

pip install django==5.0.6 \
            django-cors-headers==4.3.1 \
            django-apscheduler==0.6.2 \
            gunicorn==22.0.0 \
            requests==2.32.3 \
            python-dotenv==1.0.1 \
            scikit-learn==1.5.0 \
            pandas==2.2.2 \
            numpy==1.26.4 \
            joblib==1.4.2 \
            statsbombpy==1.0.3
```

**`requirements.txt`:**
```
django==5.0.6
django-cors-headers==4.3.1
django-apscheduler==0.6.2
gunicorn==22.0.0
requests==2.32.3
python-dotenv==1.0.1
scikit-learn==1.5.0
pandas==2.2.2
numpy==1.26.4
joblib==1.4.2
statsbombpy==1.0.3
```

### Environment Variables

Create `.env` in the project root (never commit this file):

```bash
# Django core
SECRET_KEY=your-long-random-secret-key-here
DEBUG=False
ALLOWED_HOSTS=your-backend.onrender.com,localhost

# football-data.org — register free at football-data.org/client
FOOTBALL_DATA_API_KEY=your_key_here

# Poll interval in seconds (60s stays well within 10 req/min free limit)
POLL_INTERVAL_SECONDS=60

# Frontend URL for CORS — fill in after Vercel deploy
FRONTEND_URL=http://localhost:5173
```

Note: `CURRENT_MATCH_ID` is **not** an environment variable in the Django version.
You set it via the Django Admin's `MatchConfig` table before each match. This is
the key operational improvement over the FastAPI version.

### Django Project Bootstrap

```bash
# Create the Django project
django-admin startproject wc2026 .

# Create the main app
python manage.py startapp dashboard

# After writing models (Phase 3), run migrations
python manage.py makemigrations dashboard
python manage.py migrate

# Create admin superuser (do this before deploying)
python manage.py createsuperuser
```

---

## 4. Data Sources Reference

### football-data.org

**What it provides:** Live scores, goal events, cards, lineups, standings, bracket.

**Free tier:** 10 requests/minute, no daily cap, covers the World Cup. Goal events
have a short delay (typically 1–3 minutes behind broadcast).

**Key endpoints:**

```bash
# Full WC 2026 schedule (run once during seeding)
GET https://api.football-data.org/v4/competitions/WC/matches?season=2026
Header: X-Auth-Token: YOUR_KEY

# Live match — poll this every 60s
GET https://api.football-data.org/v4/matches/{match_id}
Header: X-Auth-Token: YOUR_KEY

# Group standings
GET https://api.football-data.org/v4/competitions/WC/standings?season=2026

# Knockout bracket by stage
GET https://api.football-data.org/v4/competitions/WC/matches?stage=LAST_16
# Other stages: QUARTER_FINALS, SEMI_FINALS, FINAL
```

**Goal event shape (inside `goals` array on match response):**
```json
{
  "minute": 34,
  "injuryTime": null,
  "type": "REGULAR",
  "team": { "id": 759, "name": "Germany" },
  "scorer": { "id": 3322, "name": "Kai Havertz" },
  "assist": { "id": 3321, "name": "Florian Wirtz" }
}
```

**Rate budget:** Polling once per 60s = 1 request/min. You have 9 requests/min
remaining for standings refreshes and other calls. You will not hit the cap.

---

### StatsBomb Open Data

**What it provides:** Event-level data for every match in their open set — shots
with xG values, passes, pressures, player locations. Covers WC 2022 fully.

**Access:** No API key. Free GitHub download. Use `statsbombpy`:

```bash
pip install statsbombpy
```

```python
from statsbombpy import sb

# WC 2022: competition_id=43, season_id=106
matches = sb.matches(competition_id=43, season_id=106)

# All events for one match (shots have 'shot_statsbomb_xg' field)
events = sb.events(match_id=3869685)
shots = events[events['type'] == 'Shot']
```

You download this once during Phase 1 to build the ML training dataset. It never
needs to be called again during live matches.

---

### SoFIFA API

**What it provides:** FC 26 player ratings — overall, pace, shooting, passing,
dribbling, defending, physical, skill moves, weak foot.

**Access:** Free, register at sofifa.com/api. Rate limit: 60 requests/minute.

```bash
# Team roster with ratings
GET https://sofifa.com/api/players?team_id=21&hl=en-US

# Individual player
GET https://sofifa.com/api/player?id=239085&hl=en-US
```

You call this once in the pre-match pipeline to populate the `Player` table for
all 32 World Cup squads. Never called during live matches.

**Manual step required:** Build a mapping from football-data.org team IDs to
SoFIFA team IDs. There is no automated way to do this — spend ~1 hour looking up
each of the 32 teams on both sites and building a Python dict. This dict goes in
`data_pipeline/fetch_sofifa.py`.

---

## 5. Phase 1 — ML Win Probability Model

**Do this phase entirely offline before writing any Django code.** The model is a
small `.pkl` file (~100KB) you commit to your repository. Django loads it at
startup.

### What the Model Does

Given a snapshot of the current game state, it outputs three probabilities:
P(home win), P(draw), P(away win), summing to 1.0. These update on every new
goal event and drive the live graph on the frontend.

This is a multi-class classification problem. You use
`HistGradientBoostingClassifier` wrapped in `CalibratedClassifierCV` — the
calibration wrapper is critical for the probabilities to be meaningful (you want
P=0.70 to actually occur 70% of the time).

### Feature Engineering

**`ml/features.py`**

```python
def build_game_state_features(
    score_diff: int,
    minute: int,
    xg_diff: float,
    pre_match_elo_diff: float,
    red_card_diff: int = 0
) -> list:
    """
    Builds a 7-element feature vector for model training and inference.

    score_diff:          home goals minus away goals (e.g. +1, -2, 0)
    minute:              current match minute (1–95)
    xg_diff:             cumulative xG home minus away
                         At inference: approximated as score_diff * 0.9
                         since football-data.org free tier has no live xG
    pre_match_elo_diff:  home ELO minus away ELO, computed once pre-match
    red_card_diff:       home red cards minus away red cards
    """
    minute_norm = min(minute, 95) / 95.0
    time_remaining = 1.0 - minute_norm

    return [
        score_diff,
        minute_norm,
        time_remaining,
        xg_diff,
        pre_match_elo_diff / 400.0,      # normalise ELO scale
        red_card_diff,
        score_diff * time_remaining,     # interaction: lead size × urgency
    ]
```

### Building the Training Dataset

**`data_pipeline/load_statsbomb.py`**

```python
"""
Run once offline to generate data/wc2022_game_states.json.
Each row = one minute of one WC 2022 match, labelled with final outcome.
Expected runtime: ~5 minutes.
"""
import json
from statsbombpy import sb


def extract_game_states(match_id, home_team, away_team,
                        home_score, away_score):
    events = sb.events(match_id=match_id)
    shots = events[events['type'] == 'Shot'].copy()
    goals = shots[shots['shot_outcome'] == 'Goal'].copy()

    # Final outcome label from home team perspective
    if home_score > away_score:
        label = 1       # home win
    elif home_score < away_score:
        label = -1      # away win
    else:
        label = 0       # draw

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
            'pre_match_elo_diff': 0,   # populate if you have ELO data
            'red_card_diff': 0,
            'label': label
        })
    return rows


def build_dataset():
    matches = sb.matches(competition_id=43, season_id=106)  # WC 2022
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

    with open('data/wc2022_game_states.json', 'w') as f:
        json.dump(all_rows, f)

    print(f"\nTotal rows: {len(all_rows)} from {len(matches)} matches")


if __name__ == '__main__':
    build_dataset()
```

### Training the Model

**`ml/train.py`**

```python
"""
Run offline after load_statsbomb.py has generated the training data.
Output: ml/win_prob_model.pkl — commit this file to your repository.
"""
import json
import joblib
import numpy as np
from sklearn.ensemble import HistGradientBoostingClassifier
from sklearn.calibration import CalibratedClassifierCV
from sklearn.model_selection import train_test_split
from sklearn.metrics import log_loss
from features import build_game_state_features


def load_data(path='../data_pipeline/data/wc2022_game_states.json'):
    with open(path) as f:
        rows = json.load(f)
    X, y = [], []
    for r in rows:
        X.append(build_game_state_features(
            r['score_diff'], r['minute'], r['xg_diff'],
            r['pre_match_elo_diff'], r['red_card_diff']
        ))
        y.append(r['label'])
    return np.array(X), np.array(y)


def train():
    X, y = load_data()
    X_train, X_test, y_train, y_test = train_test_split(
        X, y, test_size=0.2, random_state=42, stratify=y
    )

    base = HistGradientBoostingClassifier(
        max_iter=200, learning_rate=0.05, max_depth=4, random_state=42
    )
    # Isotonic calibration: ensures probabilities are accurate, not just ranked
    model = CalibratedClassifierCV(base, method='isotonic', cv=5)
    model.fit(X_train, y_train)

    probs = model.predict_proba(X_test)
    print(f"Log loss on held-out set: {log_loss(y_test, probs):.4f}")
    # A good model should score below 0.85 log loss on this dataset

    joblib.dump(model, 'win_prob_model.pkl')
    print("Saved to win_prob_model.pkl")

    # Sanity checks
    sanity = [
        (0,  1,  0.0,   0, 0, "Kickoff, equal teams"),
        (1, 45,  0.5,   0, 0, "1-0 up at HT, slight xG edge"),
        (2, 85,  1.2,  50, 0, "2-0 up at 85min, stronger team"),
        (0, 80, -0.8, 100, 0, "0-0 at 80min, losing xG battle"),
        (-1, 60, 0.3,   0, 0, "1-0 down at 60, winning xG"),
    ]
    print("\nSanity checks (home_win / draw / away_win):")
    for sd, mn, xg, elo, rc, desc in sanity:
        f = build_game_state_features(sd, mn, xg, elo, rc)
        p = model.predict_proba([f])[0]
        pm = dict(zip(model.classes_.tolist(), p.tolist()))
        print(f"  {desc}")
        print(f"    Win {pm.get(1,0):.1%} | Draw {pm.get(0,0):.1%}"
              f" | Loss {pm.get(-1,0):.1%}")


if __name__ == '__main__':
    train()
```

### Expected Sanity Check Output

```
Sanity checks (home_win / draw / away_win):
  Kickoff, equal teams
    Win 40.2% | Draw 24.8% | Loss 35.0%
  1-0 up at HT, slight xG edge
    Win 68.3% | Draw 17.1% | Loss 14.6%
  2-0 up at 85min, stronger team
    Win 93.1% | Draw  4.2% | Loss  2.7%
  0-0 at 80min, losing xG battle
    Win 22.4% | Draw 31.6% | Loss 46.0%
  1-0 down at 60, winning xG
    Win 28.9% | Draw 20.5% | Loss 50.6%
```

If numbers look wildly off, verify your label encoding is consistent:
`1 = home win`, `0 = draw`, `-1 = away win`.

---

## 6. Phase 2 — Pre-Match Data Pipeline

These scripts run once before the tournament and populate SQLite with everything
the frontend needs to render without hitting any external API during a live match.
You run them locally, then commit the resulting `wc2026.db` to your repository
so Render has it baked into the Docker image at deploy time.

The Django ORM handles table creation via migrations — you do **not** call
`Base.metadata.create_all()` like in SQLAlchemy. Instead:

```bash
python manage.py makemigrations dashboard
python manage.py migrate
# Tables now exist in wc2026.db
python data_pipeline/seed_db.py
# Tables are now populated
```

### Seed Script

**`data_pipeline/seed_db.py`**

```python
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

# Bootstrap Django outside manage.py
sys.path.append(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
os.environ.setdefault('DJANGO_SETTINGS_MODULE', 'wc2026.settings')
django.setup()

from dashboard.models import Team, Player, Match
from dotenv import load_dotenv
load_dotenv()

FD_KEY = os.getenv('FOOTBALL_DATA_API_KEY')
FD_HEADERS = {'X-Auth-Token': FD_KEY}
FD_BASE = 'https://api.football-data.org/v4'

# Manual mapping: football-data.org team ID → SoFIFA team ID
# Build this dict by looking up each team on both sites (~1hr of work)
FD_TO_SOFIFA = {
    # Examples — fill in all 32:
    # 759: 21,   # Germany
    # 768: 27,   # Brazil
    # 773: 45,   # Spain
}


def seed_teams_and_matches():
    print("Fetching WC 2026 schedule...")
    resp = requests.get(
        f'{FD_BASE}/competitions/WC/matches',
        headers=FD_HEADERS,
        params={'season': 2026}
    )
    resp.raise_for_status()
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

        time.sleep(1.5)   # well within 60 req/min


if __name__ == '__main__':
    seed_teams_and_matches()
    seed_player_ratings()
    print("\nDatabase seeded. Commit wc2026.db to your repository.")
```

---

## 7. Phase 3 — Backend (Django)

### Settings

**`wc2026/settings.py`** — key additions to the default Django settings:

```python
import os
from pathlib import Path
from dotenv import load_dotenv

load_dotenv()

BASE_DIR = Path(__file__).resolve().parent.parent

SECRET_KEY = os.getenv('SECRET_KEY', 'dev-insecure-key-change-in-production')
DEBUG = os.getenv('DEBUG', 'False') == 'True'
ALLOWED_HOSTS = os.getenv('ALLOWED_HOSTS', 'localhost').split(',')

INSTALLED_APPS = [
    'django.contrib.admin',
    'django.contrib.auth',
    'django.contrib.contenttypes',
    'django.contrib.sessions',
    'django.contrib.messages',
    'django.contrib.staticfiles',
    'corsheaders',          # pip: django-cors-headers
    'django_apscheduler',   # pip: django-apscheduler
    'dashboard',            # our app
]

MIDDLEWARE = [
    'corsheaders.middleware.CorsMiddleware',   # Must be first in this list
    'django.middleware.security.SecurityMiddleware',
    'django.contrib.sessions.middleware.SessionMiddleware',
    'django.middleware.common.CommonMiddleware',
    'django.middleware.csrf.CsrfViewMiddleware',
    'django.contrib.auth.middleware.AuthenticationMiddleware',
    'django.contrib.messages.middleware.MessageMiddleware',
]

ROOT_URLCONF = 'wc2026.urls'

DATABASES = {
    'default': {
        'ENGINE': 'django.db.backends.sqlite3',
        'NAME': BASE_DIR / 'wc2026.db',
    }
}

# CORS — allow the Vercel frontend to call this backend
CORS_ALLOWED_ORIGINS = [
    os.getenv('FRONTEND_URL', 'http://localhost:5173'),
    'https://*.vercel.app',
]
CORS_ALLOW_METHODS = ['GET', 'OPTIONS']

STATIC_URL = '/static/'
STATIC_ROOT = BASE_DIR / 'staticfiles'

DEFAULT_AUTO_FIELD = 'django.db.models.BigAutoField'
```

### URL Configuration

**`wc2026/urls.py`:**
```python
from django.contrib import admin
from django.urls import path, include

urlpatterns = [
    path('admin/', admin.site.urls),
    path('', include('dashboard.urls')),
]
```

**`dashboard/urls.py`:**
```python
from django.urls import path
from . import views

urlpatterns = [
    path('health/',                              views.health,     name='health'),
    path('api/state/',                           views.state,      name='state'),
    path('api/prematch/<int:home_id>/<int:away_id>/',
                                                 views.prematch,   name='prematch'),
    path('api/standings/',                       views.standings,  name='standings'),
    path('api/matches/',                         views.all_matches, name='matches'),
    path('api/players/<int:team_id>/',           views.players,    name='players'),
]
```

### Models

**`dashboard/models.py`**

```python
from django.db import models


class Team(models.Model):
    id = models.IntegerField(primary_key=True)   # football-data.org ID
    name = models.CharField(max_length=100)
    short_name = models.CharField(max_length=10, blank=True)
    flag_url = models.URLField(blank=True)
    group = models.CharField(max_length=2, blank=True)
    pre_match_elo = models.FloatField(default=1500.0)
    fc26_overall = models.IntegerField(null=True, blank=True)

    def __str__(self):
        return self.name

    class Meta:
        ordering = ['name']


class Player(models.Model):
    id = models.IntegerField(primary_key=True)   # SoFIFA player ID
    team = models.ForeignKey(Team, on_delete=models.CASCADE,
                             related_name='players')
    name = models.CharField(max_length=100)
    position = models.CharField(max_length=10, blank=True)
    overall_rating = models.IntegerField(default=0)
    pace = models.IntegerField(default=0)
    shooting = models.IntegerField(default=0)
    passing = models.IntegerField(default=0)
    dribbling = models.IntegerField(default=0)
    defending = models.IntegerField(default=0)
    physical = models.IntegerField(default=0)
    skill_moves = models.IntegerField(default=0)
    weak_foot = models.IntegerField(default=0)
    nationality = models.CharField(max_length=50, blank=True)

    def __str__(self):
        return f"{self.name} ({self.team.name})"

    class Meta:
        ordering = ['-overall_rating']


class Match(models.Model):
    STATUS_CHOICES = [
        ('SCHEDULED', 'Scheduled'),
        ('IN_PLAY', 'In Play'),
        ('PAUSED', 'Paused'),
        ('FINISHED', 'Finished'),
        ('POSTPONED', 'Postponed'),
    ]

    id = models.IntegerField(primary_key=True)   # football-data.org match ID
    home_team = models.ForeignKey(Team, related_name='home_matches',
                                  on_delete=models.CASCADE)
    away_team = models.ForeignKey(Team, related_name='away_matches',
                                  on_delete=models.CASCADE)
    kickoff_utc = models.DateTimeField()
    stage = models.CharField(max_length=50)
    venue = models.CharField(max_length=100, blank=True)
    status = models.CharField(max_length=20, choices=STATUS_CHOICES,
                              default='SCHEDULED')
    home_score = models.IntegerField(default=0)
    away_score = models.IntegerField(default=0)

    def __str__(self):
        return (f"{self.home_team.short_name} vs "
                f"{self.away_team.short_name} — {self.kickoff_utc:%b %d}")

    class Meta:
        ordering = ['kickoff_utc']


class MatchEvent(models.Model):
    EVENT_CHOICES = [
        ('GOAL', 'Goal'),
        ('YELLOW_CARD', 'Yellow Card'),
        ('RED_CARD', 'Red Card'),
        ('SUBSTITUTION', 'Substitution'),
    ]

    match = models.ForeignKey(Match, on_delete=models.CASCADE,
                              related_name='events')
    minute = models.IntegerField()
    event_type = models.CharField(max_length=20, choices=EVENT_CHOICES)
    team = models.ForeignKey(Team, on_delete=models.SET_NULL, null=True)
    player_name = models.CharField(max_length=100)
    assist_name = models.CharField(max_length=100, blank=True)
    detail = models.CharField(max_length=50, blank=True)   # "Own Goal", "Penalty"
    created_at = models.DateTimeField(auto_now_add=True)

    def __str__(self):
        return f"{self.event_type} {self.minute}' — {self.player_name}"

    class Meta:
        ordering = ['minute']
        # Prevents duplicate inserts for the same goal event
        unique_together = ['match', 'minute', 'event_type', 'player_name']


class WinProbabilitySnapshot(models.Model):
    match = models.ForeignKey(Match, on_delete=models.CASCADE,
                              related_name='win_prob_snapshots')
    minute = models.IntegerField()
    home_win_prob = models.FloatField()
    draw_prob = models.FloatField()
    away_win_prob = models.FloatField()
    score_diff = models.IntegerField()
    xg_diff_approx = models.FloatField()
    created_at = models.DateTimeField(auto_now_add=True)

    def __str__(self):
        return (f"{self.match} min {self.minute} — "
                f"HW:{self.home_win_prob:.2f}")

    class Meta:
        ordering = ['minute']


class Standing(models.Model):
    team = models.ForeignKey(Team, on_delete=models.CASCADE)
    group = models.CharField(max_length=2)
    position = models.IntegerField()
    played = models.IntegerField(default=0)
    won = models.IntegerField(default=0)
    drawn = models.IntegerField(default=0)
    lost = models.IntegerField(default=0)
    goals_for = models.IntegerField(default=0)
    goals_against = models.IntegerField(default=0)
    points = models.IntegerField(default=0)
    updated_at = models.DateTimeField(auto_now=True)

    def __str__(self):
        return f"Group {self.group} P{self.position}: {self.team.name}"

    class Meta:
        ordering = ['group', 'position']
        unique_together = ['team', 'group']


class MatchConfig(models.Model):
    """
    Single-row config table. Edit via Django Admin to set which match
    is currently being tracked live. Only one row should exist.
    """
    current_match = models.ForeignKey(
        Match, on_delete=models.SET_NULL, null=True, blank=True,
        help_text="The match to poll live. Set this ~30 min before kickoff."
    )
    updated_at = models.DateTimeField(auto_now=True)

    def __str__(self):
        return f"Config — current match: {self.current_match}"

    class Meta:
        verbose_name = "Match Config"
        verbose_name_plural = "Match Config"
```

### Admin Registration

**`dashboard/admin.py`**

```python
from django.contrib import admin
from .models import (Team, Player, Match, MatchEvent,
                     WinProbabilitySnapshot, Standing, MatchConfig)


@admin.register(Team)
class TeamAdmin(admin.ModelAdmin):
    list_display = ['name', 'short_name', 'group', 'pre_match_elo', 'fc26_overall']
    list_editable = ['pre_match_elo', 'group']   # Edit ELO ratings inline
    search_fields = ['name']


@admin.register(Player)
class PlayerAdmin(admin.ModelAdmin):
    list_display = ['name', 'team', 'position', 'overall_rating',
                    'pace', 'shooting', 'passing']
    list_filter = ['team', 'position']
    search_fields = ['name', 'team__name']


@admin.register(Match)
class MatchAdmin(admin.ModelAdmin):
    list_display = ['__str__', 'stage', 'status',
                    'home_score', 'away_score', 'kickoff_utc']
    list_filter = ['stage', 'status']


@admin.register(MatchEvent)
class MatchEventAdmin(admin.ModelAdmin):
    list_display = ['match', 'minute', 'event_type', 'player_name', 'assist_name']
    list_filter = ['event_type', 'match']
    ordering = ['-match', 'minute']


@admin.register(WinProbabilitySnapshot)
class WinProbAdmin(admin.ModelAdmin):
    list_display = ['match', 'minute', 'home_win_prob', 'draw_prob', 'away_win_prob']
    list_filter = ['match']


@admin.register(Standing)
class StandingAdmin(admin.ModelAdmin):
    list_display = ['group', 'position', 'team', 'played', 'points']
    list_editable = ['position', 'points']


@admin.register(MatchConfig)
class MatchConfigAdmin(admin.ModelAdmin):
    """
    This is the key operational screen. Before each match:
    1. Go to /admin/dashboard/matchconfig/
    2. Set 'current match' to today's match
    3. Save — the poller picks it up within 60 seconds
    """
    list_display = ['current_match', 'updated_at']
```

### Model Inference

**`dashboard/inference.py`**

```python
import joblib
import numpy as np
from pathlib import Path

MODEL_PATH = Path(__file__).resolve().parent.parent / 'ml' / 'win_prob_model.pkl'

_model = None


def get_model():
    global _model
    if _model is None:
        try:
            _model = joblib.load(MODEL_PATH)
            print(f"[inference] Model loaded from {MODEL_PATH}")
        except FileNotFoundError:
            print(f"[inference] WARNING: Model not found at {MODEL_PATH}. "
                  "Using naive fallback.")
    return _model


def predict_win_probability(
    score_diff: int,
    minute: int,
    xg_diff: float,
    pre_match_elo_diff: float,
    red_card_diff: int = 0
) -> dict:
    """
    Returns {'home_win': float, 'draw': float, 'away_win': float}.

    Falls back to a simple formula if the model file is missing.
    xg_diff at inference time is approximated as score_diff * 0.9
    since football-data.org free tier provides no live xG.
    """
    model = get_model()

    if model is None:
        # Naive fallback: score-based logistic approximation
        time_left = max(0.0, (90 - minute) / 90.0)
        base = 0.40 + score_diff * 0.15 * (1 - time_left * 0.5)
        hw = max(0.02, min(0.95, base))
        aw = max(0.02, min(0.95, 0.80 - hw))
        dr = max(0.02, 1.0 - hw - aw)
        return {'home_win': round(hw, 4),
                'draw': round(dr, 4),
                'away_win': round(aw, 4)}

    # Import here to avoid issues if features.py isn't on the path at startup
    import sys
    sys.path.insert(0, str(Path(__file__).resolve().parent.parent / 'ml'))
    from features import build_game_state_features

    features = build_game_state_features(
        score_diff, minute, xg_diff, pre_match_elo_diff, red_card_diff
    )
    probs = model.predict_proba([features])[0]
    prob_map = dict(zip(model.classes_.tolist(), probs.tolist()))

    return {
        'home_win': round(prob_map.get(1, 0.0), 4),
        'draw': round(prob_map.get(0, 0.0), 4),
        'away_win': round(prob_map.get(-1, 0.0), 4),
    }
```

### Background Poller

**`dashboard/poller.py`**

```python
"""
Background polling using APScheduler's BackgroundScheduler.
Runs in a daemon thread alongside the Django/Gunicorn process.
Uses synchronous `requests` (not async httpx) — fine for a background thread.
"""
import os
import requests as http
from datetime import datetime, timezone

from apscheduler.schedulers.background import BackgroundScheduler

from .inference import predict_win_probability

# Module-level state dict — what /api/state/ returns instantly
current_state: dict = {
    'match_id': None,
    'status': 'NO_MATCH',
    'home_team': None,
    'away_team': None,
    'home_team_id': None,
    'away_team_id': None,
    'home_score': 0,
    'away_score': 0,
    'minute': 0,
    'events': [],
    'win_probability': {'home_win': 0.40, 'draw': 0.25, 'away_win': 0.35},
    'win_prob_history': [],    # [{minute, home_win, draw, away_win}, ...]
    'last_updated': None,
    'pre_match_elo_diff': 0.0,
}

FD_BASE = 'https://api.football-data.org/v4'


def poll_match():
    """
    Called by APScheduler every POLL_INTERVAL_SECONDS seconds.
    Reads current match from MatchConfig, polls football-data.org,
    updates SQLite and in-memory state.
    """
    # Import models inside function to avoid AppRegistryNotReady at module load
    from .models import Match, MatchEvent, WinProbabilitySnapshot, MatchConfig, Team

    config = MatchConfig.objects.select_related('current_match').first()
    if not config or not config.current_match:
        return   # No match configured — nothing to poll

    match_obj = config.current_match
    match_id = match_obj.id
    api_key = os.getenv('FOOTBALL_DATA_API_KEY', '')

    try:
        resp = http.get(
            f'{FD_BASE}/matches/{match_id}',
            headers={'X-Auth-Token': api_key},
            timeout=10
        )
    except http.exceptions.RequestException as e:
        print(f'[poller] Network error: {e}')
        return

    if resp.status_code == 429:
        print('[poller] Rate limited — skipping this cycle')
        return
    if resp.status_code != 200:
        print(f'[poller] Unexpected status {resp.status_code}')
        return

    data = resp.json()
    score = data.get('score', {})
    full_time = score.get('fullTime', {})
    home_score = full_time.get('home') or 0
    away_score = full_time.get('away') or 0
    minute = data.get('minute') or 0
    status = data.get('status', 'SCHEDULED')

    # Update match status and score in DB
    Match.objects.filter(id=match_id).update(
        status=status, home_score=home_score, away_score=away_score
    )

    # Process goal events — unique_together constraint prevents duplicates
    goal_events = []
    for goal in data.get('goals', []):
        team_id = goal.get('team', {}).get('id')
        player_name = goal.get('scorer', {}).get('name', 'Unknown')
        assist_name = (goal.get('assist') or {}).get('name', '')
        event_minute = goal.get('minute', 0)

        obj, created = MatchEvent.objects.get_or_create(
            match_id=match_id,
            minute=event_minute,
            event_type='GOAL',
            player_name=player_name,
            defaults={'team_id': team_id, 'assist_name': assist_name or ''}
        )
        if created:
            print(f'[poller] New goal: {player_name} {event_minute}\'')

        goal_events.append({
            'minute': event_minute,
            'type': 'GOAL',
            'team_id': team_id,
            'player': player_name,
            'assist': assist_name,
        })

    # Run model inference
    home_team = match_obj.home_team
    away_team = match_obj.away_team
    elo_diff = (home_team.pre_match_elo or 1500) - (away_team.pre_match_elo or 1500)
    score_diff = home_score - away_score
    actual_minute = max(1, min(minute, 95))
    xg_diff_approx = score_diff * 0.9  # approximation — no live xG on free tier

    probs = predict_win_probability(
        score_diff=score_diff,
        minute=actual_minute,
        xg_diff=xg_diff_approx,
        pre_match_elo_diff=elo_diff,
    )

    # Persist probability snapshot
    WinProbabilitySnapshot.objects.create(
        match_id=match_id,
        minute=actual_minute,
        home_win_prob=probs['home_win'],
        draw_prob=probs['draw'],
        away_win_prob=probs['away_win'],
        score_diff=score_diff,
        xg_diff_approx=xg_diff_approx,
    )

    # Update in-memory state (the dict /api/state/ serves)
    history = current_state.get('win_prob_history', [])
    history.append({'minute': actual_minute, **probs})

    current_state.update({
        'match_id': match_id,
        'status': status,
        'home_team': home_team.name,
        'away_team': away_team.name,
        'home_team_id': home_team.id,
        'away_team_id': away_team.id,
        'home_score': home_score,
        'away_score': away_score,
        'minute': actual_minute,
        'events': goal_events,
        'win_probability': probs,
        'win_prob_history': history[-95:],   # cap at 95 data points
        'last_updated': datetime.now(timezone.utc).isoformat(),
        'pre_match_elo_diff': elo_diff,
    })


_scheduler = None


def start_scheduler():
    global _scheduler
    if _scheduler is not None:
        return   # Already started

    interval = int(os.getenv('POLL_INTERVAL_SECONDS', 60))
    _scheduler = BackgroundScheduler()
    _scheduler.add_job(
        poll_match,
        trigger='interval',
        seconds=interval,
        id='poll_match',
        replace_existing=True,
        max_instances=1   # Prevent overlapping polls if one runs long
    )
    _scheduler.start()
    print(f'[scheduler] Started — polling every {interval}s')
```

### App Config — Scheduler Startup

**`dashboard/apps.py`**

```python
import os
from django.apps import AppConfig


class DashboardConfig(AppConfig):
    default_auto_field = 'django.db.models.BigAutoField'
    name = 'dashboard'

    def ready(self):
        """
        Called once when Django is fully initialised.
        Starts the APScheduler background thread.

        Guards prevent the scheduler starting twice:
        - Django's dev server runs ready() twice (one for the reloader process)
        - RUN_MAIN=true is set only on the main process
        - In production (Gunicorn), RUN_MAIN is not set, so we check
          we're not in a management command context (migrate, etc.)
        """
        is_dev_reloader = os.environ.get('RUN_MAIN') == 'true'
        is_production = not os.environ.get('RUN_MAIN')
        running_management_command = any(
            cmd in os.sys.argv for cmd in
            ['migrate', 'makemigrations', 'createsuperuser',
             'collectstatic', 'shell']
        )

        if running_management_command:
            return

        if is_dev_reloader or is_production:
            from .poller import start_scheduler
            start_scheduler()
```

### Views

**`dashboard/views.py`**

```python
from django.http import JsonResponse
from .models import Team, Player, Match, Standing
from .poller import current_state


def health(request):
    return JsonResponse({'status': 'ok'})


def state(request):
    """
    Primary endpoint — React polls this every 30 seconds.
    Returns full live match state including ML win probability history.
    Reads directly from in-memory dict: zero DB queries, instant response.
    """
    return JsonResponse(current_state)


def prematch(request, home_id, away_id):
    """Returns pre-loaded team profiles and FC 26 squad ratings."""
    try:
        home = Team.objects.get(id=home_id)
        away = Team.objects.get(id=away_id)
    except Team.DoesNotExist:
        return JsonResponse({'error': 'Team not found'}, status=404)

    def squad(team):
        return list(
            team.players
            .order_by('-overall_rating')[:23]
            .values('name', 'position', 'overall_rating',
                    'pace', 'shooting', 'passing',
                    'dribbling', 'defending', 'physical',
                    'skill_moves', 'weak_foot')
        )

    return JsonResponse({
        'home_team': {
            'id': home.id, 'name': home.name, 'group': home.group,
            'elo': home.pre_match_elo, 'fc26_overall': home.fc26_overall,
            'squad': squad(home),
        },
        'away_team': {
            'id': away.id, 'name': away.name, 'group': away.group,
            'elo': away.pre_match_elo, 'fc26_overall': away.fc26_overall,
            'squad': squad(away),
        },
        'elo_diff': (home.pre_match_elo or 1500) - (away.pre_match_elo or 1500),
    })


def players(request, team_id):
    """FC 26 ratings for a single team — used by the player ratings panel."""
    qs = (Player.objects
          .filter(team_id=team_id)
          .order_by('-overall_rating')
          .values('name', 'position', 'overall_rating', 'pace',
                  'shooting', 'passing', 'dribbling', 'defending',
                  'physical', 'skill_moves', 'weak_foot'))
    return JsonResponse(list(qs), safe=False)


def standings(request):
    """Group stage standings, grouped by group letter."""
    rows = (Standing.objects
            .select_related('team')
            .order_by('group', 'position'))
    result = {}
    for row in rows:
        g = row.group
        if g not in result:
            result[g] = []
        result[g].append({
            'position': row.position,
            'team': row.team.name,
            'team_id': row.team_id,
            'played': row.played,
            'won': row.won,
            'drawn': row.drawn,
            'lost': row.lost,
            'gf': row.goals_for,
            'ga': row.goals_against,
            'gd': row.goals_for - row.goals_against,
            'points': row.points,
        })
    return JsonResponse(result)


def all_matches(request):
    """Full tournament schedule with scores and status."""
    qs = (Match.objects
          .select_related('home_team', 'away_team')
          .order_by('kickoff_utc'))
    return JsonResponse([{
        'id': m.id,
        'home': m.home_team.name,
        'home_id': m.home_team_id,
        'away': m.away_team.name,
        'away_id': m.away_team_id,
        'home_score': m.home_score,
        'away_score': m.away_score,
        'kickoff': m.kickoff_utc.isoformat() if m.kickoff_utc else None,
        'stage': m.stage,
        'status': m.status,
        'venue': m.venue,
    } for m in qs], safe=False)
```

---

## 8. Phase 4 — Frontend (React)

### Setup

```bash
cd frontend
npm create vite@latest . -- --template react
npm install recharts
```

### API Client

**`frontend/src/api.js`**
```javascript
const BASE = import.meta.env.VITE_API_URL || 'http://localhost:8000';

export const api = {
  getState:     () => fetch(`${BASE}/api/state/`).then(r => r.json()),
  getPrematch:  (h, a) => fetch(`${BASE}/api/prematch/${h}/${a}/`).then(r => r.json()),
  getStandings: () => fetch(`${BASE}/api/standings/`).then(r => r.json()),
  getMatches:   () => fetch(`${BASE}/api/matches/`).then(r => r.json()),
  getPlayers:   (tid) => fetch(`${BASE}/api/players/${tid}/`).then(r => r.json()),
};
```

### Win Probability Graph

**`frontend/src/components/WinProbGraph.jsx`**

```jsx
import { LineChart, Line, XAxis, YAxis, CartesianGrid,
         Tooltip, ReferenceLine, ResponsiveContainer } from 'recharts';

const Tip = ({ active, payload, label }) => {
  if (!active || !payload?.length) return null;
  return (
    <div style={{ background: '#1a1a2e', border: '1px solid #333',
                  padding: '8px 12px', borderRadius: 6, fontSize: 12 }}>
      <div style={{ color: '#888', marginBottom: 4 }}>Minute {label}'</div>
      {payload.map(p => (
        <div key={p.name} style={{ color: p.color }}>
          {p.name}: {(p.value * 100).toFixed(1)}%
        </div>
      ))}
    </div>
  );
};

export default function WinProbGraph({ history, homeTeam, awayTeam }) {
  if (!history?.length) return (
    <div style={{ textAlign: 'center', color: '#555', padding: 40, fontSize: 13 }}>
      Win probability graph appears once the match starts
    </div>
  );

  return (
    <>
      <div style={{ display: 'flex', justifyContent: 'space-between',
                    fontSize: 12, color: '#aaa', marginBottom: 8 }}>
        <span style={{ color: '#4fc3f7' }}>◆ {homeTeam}</span>
        <span style={{ color: '#888' }}>— Draw</span>
        <span style={{ color: '#f06292' }}>◆ {awayTeam}</span>
      </div>
      <ResponsiveContainer width="100%" height={250}>
        <LineChart data={history}
                   margin={{ top: 4, right: 8, left: -24, bottom: 0 }}>
          <CartesianGrid strokeDasharray="3 3" stroke="#222" />
          <XAxis dataKey="minute" stroke="#444"
                 tick={{ fontSize: 10, fill: '#666' }} />
          <YAxis domain={[0, 1]} stroke="#444"
                 tick={{ fontSize: 10, fill: '#666' }}
                 tickFormatter={v => `${(v*100).toFixed(0)}%`} />
          <Tooltip content={<Tip />} />
          <ReferenceLine y={0.5}   stroke="#333" strokeDasharray="4 4" />
          <ReferenceLine x={45}    stroke="#333" strokeDasharray="4 4"
                         label={{ value: 'HT', fill: '#555', fontSize: 10 }} />
          <Line type="monotone" dataKey="home_win" name={homeTeam}
                stroke="#4fc3f7" strokeWidth={2.5} dot={false} />
          <Line type="monotone" dataKey="draw" name="Draw"
                stroke="#666" strokeWidth={1.5} dot={false}
                strokeDasharray="4 4" />
          <Line type="monotone" dataKey="away_win" name={awayTeam}
                stroke="#f06292" strokeWidth={2.5} dot={false} />
        </LineChart>
      </ResponsiveContainer>
    </>
  );
}
```

### Event Ticker

**`frontend/src/components/EventTicker.jsx`**

```jsx
const ICONS = { GOAL: '⚽', YELLOW_CARD: '🟨', RED_CARD: '🟥', SUBSTITUTION: '🔄' };

export default function EventTicker({ events }) {
  if (!events?.length)
    return <p style={{ color: '#555', fontSize: 12, textAlign: 'center' }}>
      No events yet
    </p>;

  return (
    <div style={{ maxHeight: 280, overflowY: 'auto' }}>
      {[...events].reverse().map((e, i) => (
        <div key={i} style={{ display: 'flex', gap: 10, padding: '7px 0',
                              borderBottom: '1px solid #1e1e2e', fontSize: 13 }}>
          <span>{ICONS[e.type] || '•'}</span>
          <span style={{ color: '#888', minWidth: 28 }}>{e.minute}'</span>
          <div>
            <span style={{ fontWeight: 600 }}>{e.player}</span>
            {e.assist && (
              <span style={{ color: '#555', fontSize: 11 }}>
                {' '}(assist: {e.assist})
              </span>
            )}
          </div>
        </div>
      ))}
    </div>
  );
}
```

### Main App

**`frontend/src/App.jsx`**

```jsx
import { useState, useEffect } from 'react';
import { api } from './api';
import WinProbGraph from './components/WinProbGraph';
import EventTicker from './components/EventTicker';

export default function App() {
  const [state, setState] = useState(null);
  const [lastRefresh, setLastRefresh] = useState(null);

  useEffect(() => {
    const fetch = () =>
      api.getState()
         .then(d => { setState(d); setLastRefresh(new Date()); })
         .catch(console.error);

    fetch();
    const t = setInterval(fetch, 30_000);
    return () => clearInterval(t);
  }, []);

  const isLive = state?.status === 'IN_PLAY' || state?.status === 'PAUSED';
  const isFinished = state?.status === 'FINISHED';

  return (
    <div style={{ background: '#0d0d1a', minHeight: '100vh', color: '#fff',
                  fontFamily: 'Inter, system-ui, sans-serif',
                  padding: '16px 20px', maxWidth: 1100, margin: '0 auto' }}>

      {/* Header */}
      <div style={{ display: 'flex', justifyContent: 'space-between',
                    alignItems: 'center', marginBottom: 20 }}>
        <div>
          <h1 style={{ margin: 0, fontSize: 20 }}>🏆 World Cup 2026</h1>
          <span style={{ fontSize: 11, color: '#555' }}>Watch-Along Dashboard</span>
        </div>
        <div style={{ textAlign: 'right' }}>
          {isLive && (
            <span style={{ background: '#c0392b', color: '#fff',
                           padding: '3px 10px', borderRadius: 12,
                           fontSize: 11, fontWeight: 700 }}>
              ● LIVE
            </span>
          )}
          {lastRefresh && (
            <div style={{ fontSize: 10, color: '#444', marginTop: 4 }}>
              Updated {lastRefresh.toLocaleTimeString()}
            </div>
          )}
        </div>
      </div>

      {/* Scoreboard */}
      {state?.home_team && (
        <div style={{ background: '#13132a', borderRadius: 10,
                      padding: '18px 24px', marginBottom: 16,
                      textAlign: 'center', border: '1px solid #1e1e3a' }}>
          <div style={{ display: 'flex', justifyContent: 'center',
                        alignItems: 'center', gap: 36 }}>
            <span style={{ fontSize: 20, fontWeight: 700 }}>{state.home_team}</span>
            <span style={{ fontSize: 48, fontWeight: 800, letterSpacing: 6,
                           color: isLive ? '#fff' : '#666' }}>
              {state.home_score} – {state.away_score}
            </span>
            <span style={{ fontSize: 20, fontWeight: 700 }}>{state.away_team}</span>
          </div>
          <div style={{ color: '#555', fontSize: 12, marginTop: 6 }}>
            {isLive ? `${state.minute}'` : isFinished ? 'Full Time' : 'Pre-Match'}
          </div>
        </div>
      )}

      {/* Main grid */}
      <div style={{ display: 'grid', gridTemplateColumns: '1fr 280px', gap: 14 }}>

        {/* Left: graph */}
        <div style={{ background: '#13132a', borderRadius: 10,
                      padding: 18, border: '1px solid #1e1e3a' }}>
          <h3 style={{ margin: '0 0 14px', fontSize: 12, color: '#888',
                       textTransform: 'uppercase', letterSpacing: 1 }}>
            ML Win Probability
          </h3>
          <WinProbGraph
            history={state?.win_prob_history || []}
            homeTeam={state?.home_team || 'Home'}
            awayTeam={state?.away_team || 'Away'}
          />
        </div>

        {/* Right: events + probability bars */}
        <div style={{ display: 'flex', flexDirection: 'column', gap: 14 }}>

          {/* Event ticker */}
          <div style={{ background: '#13132a', borderRadius: 10,
                        padding: 18, border: '1px solid #1e1e3a' }}>
            <h3 style={{ margin: '0 0 12px', fontSize: 12, color: '#888',
                         textTransform: 'uppercase', letterSpacing: 1 }}>
              Goals
            </h3>
            <EventTicker events={state?.events || []} />
          </div>

          {/* Current probability summary */}
          {state?.win_probability && (
            <div style={{ background: '#13132a', borderRadius: 10,
                          padding: 18, border: '1px solid #1e1e3a' }}>
              <h3 style={{ margin: '0 0 14px', fontSize: 12, color: '#888',
                           textTransform: 'uppercase', letterSpacing: 1 }}>
                Current Odds
              </h3>
              {[
                [state.home_team, state.win_probability.home_win, '#4fc3f7'],
                ['Draw',          state.win_probability.draw,     '#666'],
                [state.away_team, state.win_probability.away_win, '#f06292'],
              ].map(([label, val, color]) => (
                <div key={label} style={{ marginBottom: 12 }}>
                  <div style={{ display: 'flex', justifyContent: 'space-between',
                                fontSize: 12, marginBottom: 3 }}>
                    <span style={{ color: '#ccc' }}>{label}</span>
                    <span style={{ color, fontWeight: 600 }}>
                      {(val * 100).toFixed(1)}%
                    </span>
                  </div>
                  <div style={{ background: '#1e1e3a', borderRadius: 3, height: 5 }}>
                    <div style={{ background: color, borderRadius: 3,
                                  height: 5, width: `${val * 100}%`,
                                  transition: 'width 1s ease' }} />
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
      </div>

      {/* No match state */}
      {(!state || state.status === 'NO_MATCH') && (
        <div style={{ textAlign: 'center', color: '#555',
                      marginTop: 60, fontSize: 14 }}>
          No match configured. Check back on match day.
        </div>
      )}
    </div>
  );
}
```

### Vite Config

**`frontend/vite.config.js`:**
```javascript
import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';

export default defineConfig({
  plugins: [react()],
  server: {
    proxy: { '/api': 'http://localhost:8000' }
  }
});
```

**`frontend/.env`** (local dev):
```
VITE_API_URL=http://localhost:8000
```

**`frontend/.env.production`** (Vercel):
```
VITE_API_URL=https://your-backend.onrender.com
```

---

## 9. Phase 5 — Deployment

### Pre-Deployment Checklist

Before deploying, do these locally:

```bash
# 1. Run all migrations
python manage.py makemigrations dashboard
python manage.py migrate

# 2. Seed the database with teams, matches, player ratings
python data_pipeline/seed_db.py

# 3. Create the admin superuser
python manage.py createsuperuser

# 4. Collect static files (for Django admin CSS)
python manage.py collectstatic --noinput

# 5. Verify wc2026.db has data
python manage.py shell -c "from dashboard.models import Team; print(Team.objects.count(), 'teams')"

# 6. Commit wc2026.db and win_prob_model.pkl to the repository
git add wc2026.db ml/win_prob_model.pkl
git commit -m "Add seeded database and trained ML model"
git push
```

Both `wc2026.db` and `win_prob_model.pkl` are committed deliberately. Render's
free tier has no persistent disk — baking them into the Docker image is the
cleanest solution at this scale.

### Dockerfile

**`Dockerfile`** (in project root):
```dockerfile
FROM python:3.11-slim

WORKDIR /app

# Install dependencies
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# Copy the entire project
COPY . .

# Collect static files (Django admin needs these)
RUN python manage.py collectstatic --noinput

EXPOSE 8000

# Single worker: in-memory current_state must live in one process
CMD ["gunicorn", "wc2026.wsgi:application", \
     "--bind", "0.0.0.0:8000", \
     "--workers", "1", \
     "--timeout", "120"]
```

### Deploy Backend to Render

1. Push your repository to GitHub.
2. Go to **render.com** → New → Web Service.
3. Connect your GitHub repository.
4. Set **Runtime** to Docker (Render detects your Dockerfile automatically).
5. Add environment variables in Render's dashboard:

   | Key | Value |
   |---|---|
   | `SECRET_KEY` | A long random string (generate at djecrety.ir) |
   | `DEBUG` | `False` |
   | `ALLOWED_HOSTS` | `your-app-name.onrender.com` |
   | `FOOTBALL_DATA_API_KEY` | Your football-data.org API key |
   | `POLL_INTERVAL_SECONDS` | `60` |
   | `FRONTEND_URL` | Your Vercel URL (fill in after frontend deploy) |

6. Click **Deploy**. Your backend URL will be
   `https://your-app-name.onrender.com`.

7. After deploy, verify: `curl https://your-app-name.onrender.com/health/`
   should return `{"status": "ok"}`.

8. Check the Django admin is working:
   `https://your-app-name.onrender.com/admin/` — log in with the superuser
   credentials you set before seeding.

### Keep Render Warm

Render's free tier spins down after 15 minutes of inactivity. Set up a free
cron job at **cron-job.org** to ping your health endpoint every 10 minutes:

```
URL: https://your-app-name.onrender.com/health/
Schedule: */10 * * * *
```

This keeps cold start latency out of your friends' and LinkedIn visitors' experience.

### Deploy Frontend to Vercel

1. Go to **vercel.com** → New Project → Import your GitHub repo.
2. Set **Root Directory** to `frontend`.
3. Framework preset: Vite.
4. Add environment variable:
   - `VITE_API_URL` = `https://your-app-name.onrender.com`
5. Deploy. You'll get a URL like `your-app.vercel.app`.
6. Go back to Render → Environment → update `FRONTEND_URL` to your Vercel URL.
   Redeploy the Render service to apply the CORS update.

### Final Verification

```
[ ] https://your-backend.onrender.com/health/ → {"status": "ok"}
[ ] https://your-backend.onrender.com/api/state/ → valid JSON
[ ] https://your-backend.onrender.com/admin/ → Django admin login works
[ ] https://your-app.vercel.app → dashboard loads, no CORS errors in console
[ ] cron-job.org ping is active
[ ] .env is in .gitignore
[ ] wc2026.db and win_prob_model.pkl are committed
```

---

## 10. Match Day Operations

### Before Each Match (~30 Minutes Before Kickoff)

1. Find today's match ID on football-data.org:
   ```bash
   curl "https://api.football-data.org/v4/competitions/WC/matches?status=SCHEDULED" \
     -H "X-Auth-Token: YOUR_KEY" | python -m json.tool | grep '"id"'
   ```

2. Go to **Django Admin** → Dashboard → Match Config.
   - If no row exists, click **Add Match Config**.
   - Set **Current match** to today's match from the dropdown.
   - Save.

3. The poller picks up the new match within 60 seconds (next scheduler tick).

4. Verify on `https://your-app.vercel.app` that the correct team names appear
   in the scoreboard.

### During the Match

Nothing to do. The scheduler runs automatically every 60 seconds.
Goals appear on the dashboard within 1–3 minutes of happening on screen
(football-data.org free tier delay).

If you want an immediate refresh after a goal you just saw on TV,
add a manual trigger view:

```python
# In dashboard/views.py — add this endpoint for your own use
from django.views.decorators.csrf import csrf_exempt
from .poller import poll_match

@csrf_exempt
def force_refresh(request):
    if request.method == 'POST':
        poll_match()
        return JsonResponse({'refreshed': True})
    return JsonResponse({'error': 'POST only'}, status=405)
```

Then hit it with: `curl -X POST https://your-backend.onrender.com/api/refresh/`

### After the Match

Match status will show `FINISHED`. The win probability graph shows the full 90-minute
arc. The event log is persisted in SQLite.

Clear the `MatchConfig.current_match` in Django Admin (set it to blank) to stop
unnecessary polling until the next match.

---

## 11. Fallback & Error Handling

### football-data.org Unavailable

The poller wraps all HTTP calls in try/except. On network failure it logs the
error and returns without updating state. The frontend serves the last known
`current_state`, showing a "last updated X minutes ago" indicator if
`last_updated` is stale by more than 3 minutes:

```javascript
// In App.jsx — add this computed value
const minutesSinceUpdate = state?.last_updated
  ? Math.floor((Date.now() - new Date(state.last_updated)) / 60000)
  : null;

// Render a banner if stale
{minutesSinceUpdate > 3 && (
  <div style={{ background: '#2a1a1a', color: '#f06292',
                padding: '6px 12px', borderRadius: 6,
                fontSize: 12, marginBottom: 12 }}>
    ⚠ Live data delayed — last updated {minutesSinceUpdate} min ago
  </div>
)}
```

### Model File Missing

`inference.py` catches `FileNotFoundError` and falls back to a simple
score-based logistic formula. The probability graph still works, just with
less accurate values. A console warning is printed at startup.

### Django Admin Locked Out

If you forget your superuser password:
```bash
# Locally or via Render shell
python manage.py changepassword your_username
```

### Rate Limit Hit (429 from football-data.org)

The poller detects 429 responses and skips that cycle entirely (no retry).
The next scheduled poll runs 60 seconds later. With 10 req/min available and
polling at 1 req/min, hitting the rate limit would require a bug. If it
happens, increase `POLL_INTERVAL_SECONDS` to `90` in Render's environment
variables.

---

## 12. Testing Strategy

### Test the ML Model Offline

```bash
cd ml
python train.py
# Verify sanity check output makes intuitive sense
# Log loss below 0.85 on held-out set is good
```

### Test the Backend Locally

```bash
# Start Django dev server
python manage.py runserver

# In another terminal — check all endpoints
curl http://localhost:8000/health/
curl http://localhost:8000/api/state/
curl http://localhost:8000/api/standings/
curl http://localhost:8000/api/matches/

# Set a test match in the admin
open http://localhost:8000/admin/
```

### Replay a Historical Match

To test the poller without waiting for a live game, download a WC 2022 match
response from football-data.org as a JSON file and write a simple replay script:

```python
# tests/replay.py
"""
Simulates a match by calling poll_match() with mock responses.
Replace the HTTP call in poller.py with a mock for testing.
"""
import os, django
os.environ['DJANGO_SETTINGS_MODULE'] = 'wc2026.settings'
django.setup()

import json
from unittest.mock import patch, MagicMock
from dashboard.poller import poll_match

# Load a saved API response for a finished match
with open('tests/fixtures/match_sample.json') as f:
    sample_response = json.load(f)

mock_resp = MagicMock()
mock_resp.status_code = 200
mock_resp.json.return_value = sample_response

with patch('dashboard.poller.http.get', return_value=mock_resp):
    poll_match()

from dashboard.poller import current_state
print("State after poll:", current_state)
```

### Test CORS Before Sharing Publicly

```bash
curl -H "Origin: https://your-app.vercel.app" \
     -H "Access-Control-Request-Method: GET" \
     -X OPTIONS \
     https://your-backend.onrender.com/api/state/
# Response should include: Access-Control-Allow-Origin: https://your-app.vercel.app
```

### Load Test Before LinkedIn Post

Render's free tier (512MB RAM, 0.1 vCPU, single worker) handles ~30–50
simultaneous read-only requests comfortably since `/api/state/` serves an
in-memory dict. Use `ab` (Apache Bench) for a quick check:

```bash
ab -n 100 -c 20 https://your-backend.onrender.com/api/state/
# Look for: "Failed requests: 0" and mean response time under 500ms
```

---

## Quick Reference

```bash
# Local development
python manage.py runserver        # Django backend on :8000
cd frontend && npm run dev        # React frontend on :5173

# Database
python manage.py makemigrations dashboard
python manage.py migrate
python manage.py createsuperuser
python data_pipeline/seed_db.py   # Run after migrate

# ML model
python ml/train.py                # Run once after load_statsbomb.py

# Production build test
docker build -t wc2026 .
docker run -p 8000:8000 --env-file .env wc2026

# Check today's live matches
curl "https://api.football-data.org/v4/competitions/WC/matches?status=IN_PLAY" \
  -H "X-Auth-Token: YOUR_KEY"

# Django admin (local)
open http://localhost:8000/admin/

# Force poller to run immediately (local)
python manage.py shell -c "from dashboard.poller import poll_match; poll_match()"
```

---

*Built for FIFA World Cup 2026 (June 11 – July 19, 2026).*
*All data sources free tier: football-data.org, StatsBomb Open Data, SoFIFA API.*
*Stack: Django 5, APScheduler, SQLite, scikit-learn, React 18, Recharts.*
*Deployed: Render (backend) + Vercel (frontend). Total cost: £0.*
