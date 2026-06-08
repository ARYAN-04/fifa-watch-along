# FIFA World Cup 2026 Watch-Along Dashboard

**Work in progress.**

Real-time dashboard for the 2026 World Cup — live goal events, ML-powered win probability graph, FC 26 player ratings, and pre-match context. Built for a small audience of friends and LinkedIn visitors.

## Architecture

```
Browser ──GET /api/*──▶ Next.js API Routes (proxy, same origin)
                              │
                              ▼ fetch() server-to-server
                         Django API (ORM-backed)
                              │
                              ▼ SQLite
```

No CORS. The browser never talks to Django directly. Next.js API routes are pure pass-through proxies that forward requests to Django server-to-server.

## How It Works

- **Backend** (Django 5 + SQLite) polls football-data.org every 60s for live match events. Goals are stored in SQLite, and a trained scikit-learn model (trained on StatsBomb 2022 WC data) computes win/draw/loss probabilities after each goal.
- **Frontend** (Next.js 16 + Recharts) polls its own API routes every 30s. These routes proxy to Django, which queries the ORM and returns JSON. The frontend renders a live scoreboard, win probability line graph, goal ticker, and player ratings.
- **Admin** sets the current match via Django Admin — no SSH or env var changes needed on match day.

## Stack

| Layer | What |
|---|---|
| Backend | Django 5, Gunicorn (1 worker), django-apscheduler |
| Database | SQLite via Django ORM |
| ML | scikit-learn (HistGradientBoosting + CalibratedClassifierCV) |
| Frontend | Next.js 16, Recharts |
| Data | football-data.org (live), StatsBomb Open Data (training), SoFIFA (ratings) |
| Package mgr | uv (Python 3.11) + pnpm (frontend) |
| Deploy | Render (backend) + Vercel (frontend) |

## API Endpoints

All exposed via Next.js API routes at `/api/*`; each proxies to Django at `BACKEND_URL`.

| Endpoint | Description |
|---|---|
| `GET /api/health` | Backend health check |
| `GET /api/match` | Current match state (teams, score, stage, venue) |
| `GET /api/win-probability` | Current win/draw/loss probs + history |
| `GET /api/events` | Match events (goals, cards, subs) ordered by minute |
| `GET /api/standings` | Group standings grouped by group letter |
| `GET /api/players/:teamId` | FC 26 player ratings for a given team |

## Status

| Phase | Status |
|---|---|
| 1. ML Model — trained on StatsBomb WC 2022 (log loss 0.43) | ✅ Complete |
| 2. Data Pipeline — Django bootstrap, schema, seed script | ✅ Complete |
| 3. Django Backend — API views, admin, poller, inference | ✅ API views done; poller + inference TBD |
| 4. Next.js Frontend — BFF proxy routes, components | 🚧 Proxy layer done; components TBD |
| 5. Deployment — Docker, Render, Vercel | ⬜ Not started |

## Local Development

```bash
# Terminal 1 — Django API
uv run python manage.py runserver

# Terminal 2 — Next.js (BFF proxy + frontend)
pnpm --dir frontend dev
```
