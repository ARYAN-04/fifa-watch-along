# Casual Watcher

One Go binary that serves football live scores, league tables, head-to-head
comparisons, and minute-by-minute match replays with ML win-probability curves.

```
Browser ──GET /api/* or /──▶ Single Go binary (fifa-hub)
                               │ stdlib net/http ServeMux
                               ├── /api/* handlers → sqlc Store → SQLite (football.db)
                               ├── poller goroutine → football-data.org (openfootball fallback)
                               └── //go:embed SPA (web/dist) with index.html fallback
```

No CORS. Same-origin API. `DEV_MOCKS=1` serves canned data with no key or database.

## Run it

```bash
# Build everything (Vite SPA → embedded dist → Go binary)
make build

# Demo mode: canned data, nothing else needed
DEV_MOCKS=1 ./bin/fifa-hub        # http://localhost:8080

# Real mode: needs a free football-data.org key
FOOTBALL_DATA_API_KEY=your_key ./bin/fifa-hub
```

Knobs: `PORT` (8080), `DB_PATH` (`football.db`), `POLL_INTERVAL_SECONDS`
(15), `FOOTBALL_DATA_API_KEY`, `DEV_MOCKS`. Copy `.env.example` to `.env`
for local runs. In real mode the poller only hits leagues marked `enabled`
in the DB (Premier League by default), one request per tick, so a 60s
interval stays inside the free 10 req/min limit.

Frontend only: `pnpm --dir web dev` (Vite on :5173, proxies `/api` to :8080).
Tests: `make test` (Go) and `pnpm --dir web build` (typechecks the SPA).

## Seed the database from zero

Both DB files are gitignored build artifacts. Rebuild them with:

```bash
uv sync                                          # install Python tooling
uv run scripts/seed_replay_match.py              # replay DB: France 4-6 England
uv run scripts/seed_replay_match.py --match-id 537390   # + Final: Spain 1-0 Argentina
uv run data_pipeline/seed_football_db.py         # football.db: 5 PL seasons + WC2026 legacy import
uv run data_pipeline/compute_elo.py              # Elo ratings from finished matches
```

What lands where: 5 openfootball season files (380 matches each) become
1900 finished Premier League rows; `wc2026.db` (4 finals, 60 events,
384 snapshots) maps read-only into `football.db` as the WC2026 competition;
`compute_elo.py` rates every team from scratch (K=20, base 1500). The seeder
is idempotent, so rerunning it is safe.

Retrain or re-export the model (optional, files ship in the repo):

```bash
uv run ml/train.py            # needs data_pipeline/data/wc2022_game_states.json
uv run ml/export_model.py     # ml/win_prob_model.pkl → ml/export/model.json
```

## API

| Method | Endpoint | Notes |
|---|---|---|
| `GET` | `/api/health` | `{"status":"ok"}` |
| `GET` | `/api/scores/live` | Live matches across enabled leagues |
| `GET` | `/api/leagues/{code}/standings` | Table, lower/upper case accepted (`pl`, `wc2026`) |
| `GET` | `/api/leagues/{code}/fixtures?season=` | Season optional, defaults to latest |
| `GET` | `/api/matches/{id}` | Metadata, status, score |
| `GET` | `/api/matches/{id}/events` | Goals, cards, subs in minute order |
| `GET` | `/api/matches/{id}/win-probability` | Pre-match odds plus per-minute snapshots |
| `GET` | `/api/teams/compare?home=&away=` | H2H record, form, Elo; 400 bad id, 404 unknown team |
| `GET` | `/api/replay/matches` | Matches with probability timelines, newest first |

Bad match ids return 400, unknown ones 404.

## How the pieces fit

- **Live data:** `internal/source` polls football-data.org v4 (3 tries,
  exponential backoff, honors 429 `Retry-After`). If a league has no
  provider coverage, `internal/source/github_static.go` can read static
  openfootball season files instead.
- **Poller:** every tick, for each enabled competition, it upserts teams and
  matches, turns score diffs into `GOAL` events, and writes one
  win-probability snapshot per live minute (deduped by a targeted
  `SELECT EXISTS`). Finished matches never gain snapshots at runtime;
  history comes from the seeder.
- **ML:** the model trains offline in sklearn (scaled logistic regression
  soft-voted with a calibrated random forest, 10 game-state features) and
  exports to `ml/export/model.json`. The Go engine in `internal/inference`
  replays the math by hand, including the float32 cast sklearn applies
  before tree splits. `parity_test.go` checks 500 vectors against Python
  output; worst error so far is 2.22e-16 against a 1e-6 tolerance.
- **Replay:** any match with stored snapshots is replayable. The match page
  auto-plays from minute 0 (1 match-minute per second, 0.5-10x speeds),
  folds the score from goals up to the scrub position, and slices the chart
  and event log client-side. No server state; switching matches is a route
  change.
- **Elo:** `data_pipeline/compute_elo.py` recomputes every rating from
  scratch over finished matches in kickoff order. The server only reads.

## Status

Premier League is live-enabled. The World Cup 2026 finals ship as seeded
replays. UCL, La Liga, Serie A, and Bundesliga rows exist but stay disabled
until their provider coverage is checked. Player ratings and lineups were
cut on purpose; everything else from the original watch-along (replay
scrubbing, timelines, Broadsheet styling) is back.
