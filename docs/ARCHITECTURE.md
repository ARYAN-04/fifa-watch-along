# Architecture

How Casual Watcher fits together. For the five-minute version, run it and
click through: front page, a replay, compare, fixtures.

## Layout

```
Browser ──GET /api/* or /──▶  fifa-hub (one process)
                               ├─ stdlib net/http ServeMux
                               ├─ /api/* handlers ──▶ sqlc Store ──▶ football.db
                               ├─ poller goroutine ──▶ football-data.org
                               └─ embedded Vite + React SPA (go:embed, index fallback)
```

One static binary, one SQLite file, same-origin API, no CORS. Routes that
are not `/api/*` or hashed assets fall back to `index.html`, so deep links
like `/match/1903` survive a refresh. `internal/web/dist` is copied from
`web/dist` at build time, so the binary always carries the UI it built with.

## API

League codes accept any case (`pl`, `wc2026`). Bad ids return 400, unknown
rows return 404.

| Method | Endpoint | Returns |
|---|---|---|
| `GET` | `/api/health` | Service heartbeat |
| `GET` | `/api/scores/live` | Live matches across enabled leagues |
| `GET` | `/api/leagues/{code}/standings` | Table with played, won, drawn, lost, goals, points |
| `GET` | `/api/leagues/{code}/fixtures?season=` | Fixtures and results, season optional |
| `GET` | `/api/matches/{id}` | Metadata, status, score, minute |
| `GET` | `/api/matches/{id}/events` | Goals, cards, subs in minute order |
| `GET` | `/api/matches/{id}/win-probability` | Pre-match odds plus per-minute snapshots |
| `GET` | `/api/teams/compare?home=&away=` | H2H record, average goals, form guides, Elo |
| `GET` | `/api/replay/matches` | Matches with stored timelines, newest first |

`DEV_MOCKS=1` answers all nine from canned data: a Premier League slice,
the four real World Cup finals, and empty responses for leagues with no data.

## The ML model

Training happens offline in scikit-learn on StatsBomb WC2022 game states
(`data_pipeline/data/wc2022_game_states.json`, one row per match-minute).
`uv run ml/train.py` writes `ml/win_prob_model.pkl`.

Each game state becomes 10 features (`ml/features.py`, mirrored exactly in
`internal/inference/features.go`): score difference, normalized minute,
time remaining, xG difference, Elo difference over 400, red-card difference,
and four lead-versus-time interactions. The time-decay terms are the point:
a 1-0 lead at minute 5 reads about 45/17/38, not 90%.

The model is a scaled logistic regression soft-voted 50/50 with a 5-fold
calibrated random forest. `uv run ml/export_model.py` flattens it to
`ml/export/model.json` (coefficients, tree nodes, sigmoids). The Go engine
replays the math by hand, including the float32 cast sklearn applies before
tree splits. `parity_test.go` checks 500 vectors against stored Python
output; worst error to date is 2.22e-16 against a 1e-6 tolerance.

## Live data and seeding

The poller ticks every `POLL_INTERVAL_SECONDS` over competitions marked
`enabled` (Premier League by default). It upserts teams and matches, turns
score diffs into `GOAL` rows, and writes one probability snapshot per live
minute, skipping existing ones with a targeted `SELECT EXISTS`. The
football-data.org client retries 3 times with exponential backoff and honors
429 `Retry-After`. Our league codes stay stable in the DB while an
`id_crosswalk` table translates them to provider codes at poll time. A
second client reads static openfootball files for leagues without coverage.

Finished matches never gain snapshots at runtime. History comes from the
seeder, which is idempotent:

1. `uv run scripts/seed_replay_match.py [--match-id]` writes one finished
   final plus its event log and 96 per-minute snapshots into `wc2026.db`.
2. `uv run data_pipeline/seed_football_db.py` loads 5 openfootball seasons
   (380 matches each, 1900 finished PL rows), matches team names against the
   frozen Reep register, and imports the wc2026 legacy rows read-only.
3. `uv run data_pipeline/compute_elo.py` rates every team from scratch
   (K=20, base 1500, goal-margin multiplier).

## Replay

Any match with stored snapshots is replayable; the match page auto-plays
from minute 0 at one match-minute per second (0.5-10x speeds). The score
refolds from goals up to the scrub position, the chart draws a marker at
that minute, and the event log filters to everything so far. Scrubbing
re-slices already-fetched data, so it fires zero requests. Switching matches
is a route change; the Replay Archive lists everything replayable.

## Caching

No Redis, no CDN. Hashed JS/CSS under `/assets/` are immutable for a year;
`index.html` is never cached. React Query refetches live scores every 15
seconds, match data goes stale after 30 seconds, and tables, fixtures,
compare, and the replay list go stale after 5-10 minutes. The poller keeps
last-seen scores in a process-lifetime map (a restart just skips goal
detection for one tick), and `external_id` plus `(match_id, minute)`
uniqueness constraints make retries converge instead of duplicating. The
provider client caches nothing; the poll interval is the throttle.

## Status

Premier League is live-enabled and the World Cup 2026 finals ship as seeded
replays. UCL, La Liga, Serie A, and Bundesliga rows exist but stay disabled
until their provider coverage is checked. Player ratings and lineups were
cut on purpose; replay scrubbing, timelines, and the Broadsheet styling are
back.
