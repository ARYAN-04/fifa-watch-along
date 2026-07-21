# FIFA World Cup 2026 Watch-Along Dashboard

**Real-time dashboard for the 2026 World Cup** — featuring live match events, ML-powered win probability ledger, FC 26 player ratings, group standings, and pre-match context. Styled in a vintage newspaper "Broadsheet" layout.

## Architecture

```
Browser ──GET /api/*──▶ Next.js API Routes (proxy, same origin)
                              │
                              ▼ fetch() server-to-server
                         FastAPI API (SQLAlchemy-backed)
                              │
                              ▼ SQLite
```

No CORS. The browser never talks to the FastAPI backend directly. Next.js API routes act as pure pass-through proxies that forward requests to the FastAPI server.

## How It Works

- **Backend** (FastAPI + SQLAlchemy + SQLite) runs a background scheduler (APScheduler) that polls `football-data.org` every 60s for live match events. Goals and bookings are saved, and a calibrated `scikit-learn` classifier (trained on StatsBomb 2022 WC data) runs ML win/draw/loss inference snapshots after every match state update.
- **Frontend** (Next.js 16 + Recharts + Tailwind v4) polls its API routes every 30s. These routes proxy requests to the FastAPI server, returning structured JSON. The frontend is styled in the Broadsheet visual language, rendering a ticket-style scoreboard, probability ledger, goal ticker, standings grid, and roster ratings.
- **Admin** uses `sqladmin` (FastAPI-native admin panel) to configure the active match ID on match days without requiring environment variables or SSH updates.

## Stack

| Layer | Technology |
|---|---|
| Backend | FastAPI, Uvicorn, APScheduler |
| Database | SQLite via SQLAlchemy 2.0 |
| Admin | SQLAdmin (session-based authentication) |
| ML | scikit-learn (HistGradientBoosting + CalibratedClassifierCV) |
| Frontend | Next.js 16 (App Router), Recharts, Tailwind CSS v4, Outfit & Space Mono fonts |
| Data | football-data.org (live), StatsBomb Open Data (training), SoFIFA (ratings) |
| Package mgr | uv (Python 3.11) + pnpm (frontend) |

## API Endpoints

All exposed via Next.js API routes at `/api/*`; each proxies to FastAPI at `BACKEND_URL`.

| Endpoint | Description |
|---|---|
| `GET /api/health` | Backend health check status |
| `GET /api/match` | Current match state (squads, score, stage, venue) |
| `GET /api/win-probability` | Current win/draw/loss probs + historical snapshot timeline |
| `GET /api/events` | Match events (goals, cards, substitutions) |
| `GET /api/standings` | Group standings grouped by letter |
| `GET /api/players/:teamId` | FC 26 player ratings and roster details |

## Local Development

### 1. Backend Server

Make sure python dependencies are installed and run the FastAPI server:

```bash
# Start FastAPI backend (Port 8000)
uv run uvicorn api.main:app --reload
```

Open `http://localhost:8000/admin` to access the SQLAdmin panel (default: `admin` / `admin`).

### 2. Next.js Frontend

Run the Next.js development server:

```bash
# Start Next.js (Port 3000)
pnpm --dir frontend dev
```

*Note: The frontend works standalone with built-in mock fallback data if the backend is down (e.g., mock Argentina vs Canada).*

## Running Tests

The backend includes a comprehensive test suite using `pytest` and `httpx` to verify all routers, database sessions, and background polling/inference functions:

```bash
# Run backend test suite
PYTHONPATH=. uv run pytest
```
