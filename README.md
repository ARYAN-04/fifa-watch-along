# FIFA World Cup 2026 Watch-Along Dashboard

**Work in progress.**

Real-time dashboard for the 2026 World Cup — live goal events, ML-powered win probability graph, FC 26 player ratings, and pre-match context. Built for a small audience of friends and LinkedIn visitors.

## How It Works

- **Backend** (Django 5 + SQLite) polls football-data.org every 60s for live match events. Goals are stored in SQLite, and a trained scikit-learn model (trained on StatsBomb 2022 WC data) computes win/draw/loss probabilities after each goal.
- **Frontend** (React 18 + Recharts) polls the backend every 30s and renders a live scoreboard, win probability line graph, goal ticker, and player ratings.
- **Admin** sets the current match via Django Admin — no SSH or env var changes needed on match day.

## Stack

| Layer | What |
|---|---|
| Backend | Django 5, Gunicorn (1 worker), django-apscheduler |
| Database | SQLite via Django ORM |
| ML | scikit-learn (HistGradientBoosting + CalibratedClassifierCV) |
| Frontend | React 18, Vite, Recharts |
| Data | football-data.org (live), StatsBomb Open Data (training), SoFIFA (ratings) |
| Package mgr | uv (Python 3.11) |
| Deploy | Render (backend) + Vercel (frontend) |

## Status

Early development. Phases being built in order:
1. ML model training pipeline
2. Pre-match data pipeline
3. Django backend
4. React frontend
5. Deployment


