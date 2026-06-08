# AGENTS.md — Project Context

## Overview

FIFA World Cup 2026 Watch-Along Dashboard. Real-time dashboard with ML-powered
win probability, live goal events, FC 26 player ratings, and pre-match context.

## Stack

- **Backend:** Django 5 + Gunicorn (1 worker), SQLite, django-apscheduler
- **ML:** scikit-learn (HistGradientBoostingClassifier + CalibratedClassifierCV)
- **Frontend:** Next.js 14 + App Router + Recharts
- **Data:** football-data.org (live), StatsBomb Open Data (ML training), SoFIFA (ratings)
- **Deploy:** Render (backend) + Vercel (frontend)
- **Package Manager:** uv (Python 3.11 via `uv python install`) + pnpm (frontend)

## Directory Layout

```
├── wc2026/              # Django project config (settings, urls, wsgi)
├── dashboard/           # Django app (models, views, admin, poller, inference)
├── ml/                  # ML training pipeline (features.py, train.py, evaluate.py)
│   └── win_prob_model.pkl   # Trained model (~4.7MB, committed)
├── data_pipeline/       # Data loading scripts (load_statsbomb, fetch_sofifa, seed_db)
│   └── data/            # Static JSON data files
│       └── wc2022_game_states.json   # 6,080 rows from StatsBomb WC 2022
├── frontend/            # Next.js app (App Router)
│   └── src/components/  # WinProbGraph, EventTicker, PlayerRatings, etc.
├── tests/               # Test fixtures and replay scripts
├── manage.py            # Django CLI entrypoint
├── .venv/               # Virtual environment (Python 3.11)
├── requirements.txt     # Auto-generated via `uv export` for Docker
├── uv.lock              # Lockfile for reproducible installs
├── pyproject.toml       # Project metadata + dependencies (uv source of truth)
├── Dockerfile
├── .env                 # Local secrets (gitignored)
├── .env.example         # Template for .env
└── AGENTS.md            # ← This file — update after every meaningful change
```

## Implementation Phases

1. **Phase 1 — ML Model** (offline): StatsBomb training data → win_prob_model.pkl
2. **Phase 2 — Data Pipeline** (offline): Seed SQLite with teams, matches, players
3. **Phase 3 — Django Backend**: Models, admin, views, poller, inference
4. **Phase 4 — Next.js Frontend**: Components, API client, graph
5. **Phase 5 — Deployment**: Docker, Render, Vercel

## Rules for This Agent

- Update this file after every meaningful change (new files, structural changes,
  dependency changes, configuration changes).
- Keep directory layout in sync with the actual project structure.
- Log decisions and rationale so future sessions pick up context instantly.
- **PNPM ONLY**: Use `pnpm` for ALL frontend package operations. Never `npm install`,
  `npm add`, `npm create`, or any npm command. Every frontend operation must use pnpm
  (`pnpm add`, `pnpm dev`, `pnpm build`, `pnpm create next-app`, etc.).
- **MINIMUM RELEASE AGE**: The `minReleaseAge` or equivalent setting in any config
  file (including `.npmrc`, `.pnpmrc`, or platform settings) must never be altered,
  removed, or circumvented.

## Change Log

| Date | Change |
|---|---|
| 2026-06-07 | Initial project bootstrap. Created directory structure, AGENTS.md, .gitignore, .env.example. Installed deps via `uv add`. `statsbombpy` unpinned (v1.0.3 doesn't exist on PyPI; resolved to 1.19.0). Python 3.11 managed by uv. Removed placeholder `main.py`. |
| 2026-06-08 | **Phase 1 complete.** Created `ml/features.py` (7-feature vector), `data_pipeline/load_statsbomb.py` (StatsBomb dataset builder), `ml/train.py` (HGB+CalibratedClassifierCV training), `ml/evaluate.py` (calibration checks). Ran load_statsbomb.py → `data/wc2022_game_states.json` (6,080 rows, 64 WC 2022 matches). Ran train.py → `ml/win_prob_model.pkl` (log loss 0.4310 on held-out set). Model slightly overconfident on sanity checks because `pre_match_elo_diff=0` in training data — expected, real ELO diffs will improve calibration at inference time. |
| 2026-06-08 | **Phase 2 started (minimal bootstrap + schema).** Bootstrapped Django project: `manage.py`, `wc2026/settings.py`, `wc2026/urls.py`, `wc2026/wsgi.py`. Created all 7 ORM models in `dashboard/models.py`, admin registrations, `apps.py`, stub `views.py` + `poller.py` + `urls.py`. Created `data_pipeline/seed_db.py` (skips API calls if `FOOTBALL_DATA_API_KEY` is unset). Ran `makemigrations` + `migrate` → `wc2026.db` schema created (232KB, 16 tables). `.env` template written with placeholder key. |

