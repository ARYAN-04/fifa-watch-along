# AGENTS.md — Project Context

## Overview

FIFA World Cup 2026 Watch-Along Dashboard. Real-time dashboard with ML-powered
win probability, live goal events, FC 26 player ratings, and pre-match context.

## Stack

- **Backend:** Django 5 + Gunicorn (1 worker), SQLite, django-apscheduler
- **ML:** scikit-learn (HistGradientBoostingClassifier + CalibratedClassifierCV)
- **Frontend:** React 18 + Vite + Recharts
- **Data:** football-data.org (live), StatsBomb Open Data (ML training), SoFIFA (ratings)
- **Deploy:** Render (backend) + Vercel (frontend)
- **Package Manager:** uv (Python 3.11 via `uv python install`)

## Directory Layout

```
├── wc2026/              # Django project config (settings, urls, wsgi)
├── dashboard/           # Django app (models, views, admin, poller, inference)
├── ml/                  # ML training pipeline (features.py, train.py, evaluate.py)
├── data_pipeline/       # Data loading scripts (load_statsbomb, fetch_sofifa, seed_db)
│   └── data/            # Static JSON data files
├── frontend/            # React SPA (Vite)
│   └── src/components/  # WinProbGraph, EventTicker, PlayerRatings, etc.
├── tests/               # Test fixtures and replay scripts
├── .venv/               # Virtual environment (Python 3.11)
├── requirements.txt     # Auto-generated via `uv export` for Docker
├── uv.lock              # Lockfile for reproducible installs
├── pyproject.toml       # Project metadata + dependencies (uv source of truth)
├── Dockerfile
├── .env.example
└── AGENTS.md            # ← This file — update after every meaningful change
```

## Implementation Phases

1. **Phase 1 — ML Model** (offline): StatsBomb training data → win_prob_model.pkl
2. **Phase 2 — Data Pipeline** (offline): Seed SQLite with teams, matches, players
3. **Phase 3 — Django Backend**: Models, admin, views, poller, inference
4. **Phase 4 — React Frontend**: Components, API client, graph
5. **Phase 5 — Deployment**: Docker, Render, Vercel

## Rules for This Agent

- Update this file after every meaningful change (new files, structural changes,
  dependency changes, configuration changes).
- Keep directory layout in sync with the actual project structure.
- Log decisions and rationale so future sessions pick up context instantly.

## Change Log

| Date | Change |
|---|---|
| 2026-06-07 | Initial project bootstrap. Created directory structure, AGENTS.md, .gitignore, .env.example. Installed deps via `uv add`. `statsbombpy` unpinned (v1.0.3 doesn't exist on PyPI; resolved to 1.19.0). Python 3.11 managed by uv. Removed placeholder `main.py`. |
