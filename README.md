# Casual Watcher

A football watch-along site in a single Go binary: live scores, league
tables, head-to-head team comparisons, and minute-by-minute replays of
finished matches with machine-learned win probability curves.

## Run it

```bash
make build
DEV_MOCKS=1 ./bin/fifa-hub     # demo data, nothing else needed
```

Open http://localhost:8080. For live data instead of canned data, set a
free football-data.org key and drop `DEV_MOCKS`:

```bash
FOOTBALL_DATA_API_KEY=your_key ./bin/fifa-hub
```

See `.env.example` for all settings (`PORT`, `DB_PATH`,
`POLL_INTERVAL_SECONDS`).

## Build from zero

The databases are local build artifacts. To rebuild them:

```bash
uv sync
uv run scripts/seed_replay_match.py --match-id 537389
uv run data_pipeline/seed_football_db.py
uv run data_pipeline/compute_elo.py
```

## Docs

[docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) covers the API reference, the
ML model, the poller, the seed chain, and caching.
