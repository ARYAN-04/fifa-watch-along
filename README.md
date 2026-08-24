# FIFA Hub

Multi-league football data platform in a **single static Go binary** — live scores, league standings & fixtures, head-to-head team comparison with Elo ratings, and ML-powered win-probability curves.

## Architecture

```
Browser ──GET /api/* or /──▶ Single Go binary (fifa-hub)
                               │ stdlib net/http ServeMux
                               ├── /api/* handlers → sqlc Store → SQLite (football.db)
                               ├── poller goroutine → football-data.org (→ openfootball fallback)
                               └── //go:embed SPA (web/dist) with index.html fallback
```

No CORS. Same-origin API. `DEV_MOCKS=1` serves canned datasets without any data source.

## Stack

| Layer | Technology |
|---|---|
| Backend | Go 1.27, stdlib `net/http`, sqlc + modernc.org/sqlite |
| ML inference | Pure-Go engine over exported scikit-learn ensemble (golden parity ≤1e-6) |
| Frontend | Vite + React 19 + TanStack Router/Query + Tailwind CSS v4 + Recharts (embedded via go:embed) |
| Live data | football-data.org v4 (retry ×3, 429 backoff), openfootball/football.json fallback |
| Offline tooling | uv (Python): training, model export, DB seeding, Elo computation |

## API Endpoints

| Method | Endpoint | Description |
|---|---|---|
| `GET` | `/api/health` | Service health status |
| `GET` | `/api/scores/live` | Live scores across enabled leagues |
| `GET` | `/api/leagues/{code}/standings` | League table (points, GD, form) |
| `GET` | `/api/leagues/{code}/fixtures?season=` | Season fixtures/results (season optional) |
| `GET` | `/api/matches/{id}` | Match metadata, status, score |
| `GET` | `/api/matches/{id}/events` | Chronological goals/cards/subs |
| `GET` | `/api/matches/{id}/win-probability` | Pre-match odds + in-game probability snapshots |
| `GET` | `/api/teams/compare?home=&away=` | H2H record, avg goals, form guides, Elo |

## Local Development

```bash
# Build everything (frontend → embedded dist → Go binary)
make build

# Run tests
make test

# Dev mode: canned mock data, no API key needed
DEV_MOCKS=1 ./bin/fifa-hub

# Live mode: needs football-data.org key (free at football-data.org/client)
FOOTBALL_DATA_API_KEY=your_key ./bin/fifa-hub
```

Environment variables: `PORT` (8080), `DB_PATH` (`football.db`), `POLL_INTERVAL_SECONDS` (15), `FOOTBALL_DATA_API_KEY`, `DEV_MOCKS`. See `.env.example`.

### Web SPA development

```bash
cd web && pnpm install && pnpm dev   # Vite dev server on :5173, proxies /api → :8080
```

### Offline Python tooling (uv)

```bash
uv run python data_pipeline/seed_football_db.py   # seed multi-league football.db
uv run python data_pipeline/compute_elo.py        # recompute Elo from finished matches
uv run python ml/export_model.py                  # re-export sklearn model → ml/export/model.json
```

## Data Pipeline

- **Seeding:** 5 Premier League seasons from [openfootball/football.json](https://github.com/openfootball/football.json) + legacy WC2026 knockout data, with team name-matching against the frozen [Reep v0 register](https://github.com/withqwerty/reep) for provider ID crosswalks.
- **Elo:** computed offline from all finished matches (World-Football-Elo style, K=20, goal-diff multiplier); stored in `elo_ratings`.
- **ML:** soft-voting ensemble (Scaled Logistic Regression + Calibrated Random Forest) trained offline on StatsBomb WC2022 game states; exported to JSON and evaluated by a hand-rolled Go tree/sigmoid engine verified to machine precision against Python.

## Project Status

Go port complete. Only the Premier League is enabled pending live provider verification; UCL / La Liga / Serie A / Bundesliga / WC2026 are seeded but disabled until their football-data.org coverage is validated.
