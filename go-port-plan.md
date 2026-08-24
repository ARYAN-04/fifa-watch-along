# Football Hub and ML Match Predictor: Go Port and Platform Architecture Plan

This document defines the unified architectural design, data strategy, and implementation plan for transforming `fifa-watch-along` into a high-performance football data platform and match prediction hub. The system ports the legacy Python FastAPI and Next.js backend to a single Go binary with an embedded React 19 and TanStack Router frontend.

## 1. Product direction and scope adjustments

The platform shifts from a single-match World Cup watchalong tool to a multi-league football data website with three primary pillars:

1. **Global League Hub and Score Center:** Real-time score strip, active match tracker, league standings, and historical results archive across the Premier League, Champions League, La Liga, Serie A, Bundesliga, and World Cup 2026.
2. **Head-to-Head Team Comparison Engine:** Historical match results, goal differential stats, form guides, and pre-match Elo ratings between any two clubs or national teams.
3. **ML Match Predictor and Streamlined Live View:** Pre-match win odds and live in-game win/draw/loss probability curves powered by a pure Go machine learning inference engine, accompanied by a clean chronological game events feed (goals, cards, substitutions).

### What is removed
- Granular player ratings, EA FC 26 player cards, and lineup rosters are removed to eliminate unmaintained data dependencies and simplify schema requirements.
- Complex replay scrubbing simulation controls are streamlined into an on-demand historical match inspector with probability curves and event logs.

## 2. System architecture and caller contract

The entire system compiles into a single static Go binary (`fifa-hub`) with zero external runtime dependencies.

```
+-----------------------------------------------------------------------------------+
|                        Client Layer (React 19 + TanStack Router)                  |
|          Same-origin fetch("/api/...") • Recharts • Tailwind v4 • Lucide           |
+-----------------------------------------------------------------------------------+
                                         │
                                         ▼
+-----------------------------------------------------------------------------------+
|                            Go Server (cmd/server/main.go)                         |
|  +-----------------------------------------------------------------------------+  |
|  | HTTP Router (stdlib net/http ServeMux with path value routing)              |  |
|  +-----------------------------------------------------------------------------+  |
|  | Store Layer (modernc.org/sqlite + sqlc typed queries on football.db)        |  |
|  +-----------------------------------------------------------------------------+  |
|  | ML Inference Engine (m2cgen generated Random Forest + Platt calibration)     |  |
|  +-----------------------------------------------------------------------------+  |
|  | Background Poller (Multi-tier cache: sync.Map -> football-data.org -> GH)   |  |
|  +-----------------------------------------------------------------------------+  |
|  | Embedded Web Assets (//go:embed frontend/dist with SPA fallback)            |  |
|  +-----------------------------------------------------------------------------+  |
+-----------------------------------------------------------------------------------+
```

### Server initialization contract (`cmd/server/main.go`)

```go
func main() {
	cfg := config.Load() // env: PORT, DB_PATH, POLL_INTERVAL_SECONDS, FOOTBALL_DATA_API_KEY, DEV_MOCKS
	st := store.Open(cfg.DBPath)
	inf := inference.New("ml/export/model.json")

	mux := http.NewServeMux()
	api.Register(mux, api.Deps{
		Store:   st,
		Predict: inf.Predict,
		Mocks:   cfg.DevMocks,
	})
	web.Mount(mux) // go:embed frontend/dist (SPA fallback to index.html)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	go poller.Run(ctx, poller.Deps{
		Store:   st,
		Source:  source.NewFootballData(cfg.APIKey),
		Predict: inf.Predict,
		Every:   cfg.PollInterval,
	})

	http.ListenAndServe(":"+cfg.Port, mux)
}
```

## 3. Directory layout and module map

```
fifa-watch-along/
├── cmd/
│   └── server/
│       └── main.go
├── internal/
│   ├── config/
│   │   └── config.go             // Environment configuration
│   ├── domain/
│   │   └── types.go              // Match, Standing, HeadToHead, Event, Probabilities
│   ├── store/
│   │   ├── store.go              // modernc SQLite store initialization and queries
│   │   └── queries/
│   │       ├── matches.sql       // Named sqlc queries
│   │       ├── standings.sql
│   │       └── teams.sql
│   ├── inference/
│   │   ├── inference.go          // Predict(GameState) Probabilities
│   │   ├── model_gen.go          // Generated Go code via m2cgen (Random Forest trees)
│   │   └── calib.go              // StandardScaler, Logistic Regression, Platt Sigmoids
│   ├── source/
│   │   ├── source.go             // DataSource interface
│   │   ├── football_data.go      // football-data.org v4 client (retry x3, 429 backoff)
│   │   └── github_static.go      // openfootball/football.json fallback client
│   ├── poller/
│   │   └── poller.go             // Ticker poller, in-memory TTL cache, event diffs
│   ├── api/
│   │   ├── register.go           // Route registration and trailing-slash middleware
│   │   ├── handlers_matches.go   // Live scores, match history, and events
│   │   ├── handlers_teams.go     // Team comparison and Head-to-Head stats
│   │   ├── handlers_predict.go   // ML win probability snapshots
│   │   └── mocks.go              // Canned datasets for DEV_MOCKS=1
│   └── web/
│       └── embed.go              // go:embed frontend/dist with SPA fallback handler
├── frontend/
│   ├── src/
│   │   ├── routes/
│   │   │   ├── __root.tsx        // Root layout with navigation header and league tabs
│   │   │   ├── index.tsx         // Live score strip, active match spotlight, and standings
│   │   │   ├── compare.tsx       // Head-to-Head team comparison view
│   │   │   ├── fixtures.tsx      // Comprehensive league schedule and history
│   │   │   └── match.$id.tsx     // Match detail: ML probability curve + live event feed
│   │   ├── components/
│   │   │   ├── ScoreStrip.tsx    // Real-time horizontal score carousel
│   │   │   ├── WinProbChart.tsx  // Recharts win/draw/loss probability timeline
│   │   │   ├── EventFeed.tsx     // Clean chronological game events
│   │   │   └── H2HComparison.tsx // Team comparison radar and head-to-head records
│   │   └── lib/
│   │       └── api.ts            // Typed fetchers for TanStack Query
├── ml/
│   └── export_model.py           // Python script exporting scikit-learn model to model.json
├── go.mod
└── go.sum
```

## 4. API endpoints specification

All endpoints are served under `/api/*` on the same origin without CORS overhead.

| Method | Endpoint | Description |
|---|---|---|
| `GET` | `/api/health` | Service health status. |
| `GET` | `/api/scores/live` | Current live scores across all tracked leagues. |
| `GET` | `/api/leagues/{leagueId}/standings` | League table with rank, points, goal diff, and form. |
| `GET` | `/api/leagues/{leagueId}/fixtures` | Full season fixture list and past match results. |
| `GET` | `/api/teams/compare?home={id}&away={id}` | Head-to-Head history, average goals, and Elo ratings. |
| `GET` | `/api/matches/{id}` | Match metadata, status, venue, and current score. |
| `GET` | `/api/matches/{id}/events` | Chronological game events (goals, cards, substitutions). |
| `GET` | `/api/matches/{id}/win-probability` | Pre-match odds and in-game probability curve snapshots. |

## 5. Free football data strategy

To maintain zero recurring operational costs, the data layer uses a three-tier fallback architecture:

```
[ Tier 1: In-Memory Cache (Go sync.Map, TTL 15s) ]
                        │
                        ▼ (Cache miss)
[ Tier 2: football-data.org Free Tier (10 req/min, 6 major leagues) ]
                        │
                        ▼ (Rate limited or network error)
[ Tier 3: GitHub Static JSON (openfootball/football.json & withqwerty/reep) ]
```

1. **`football-data.org` (Free Tier):** Primary source for live scores, active minute updates, and standings.
2. **`openfootball/football.json` (GitHub Repository):** Zero rate-limit static backup for seasonal fixture schedules and historical match outcomes.
3. **`withqwerty/reep` (The Reep Register):** Crosswalk dataset mapping team/competition IDs across ~30 providers (Transfermarkt, FBref, Opta, Sofascore, ClubElo, …), CC0-licensed. **Verified 2026-08-24:** the GitHub repo's v0 register is **frozen** (last release 2026.25, 21 June 2026; API unwritten since 25 April 2026); active maintenance moved to reep.football with non-interchangeable v1 IDs. Crucially, it has **no football-data.org namespace**, so it cannot directly translate between our Tier-2 and Tier-3 sources. Usage policy: treat the frozen v0 CSVs (`teams.csv`, `competitions.csv`) as a one-time offline seed-time asset downloaded into `data_pipeline/data/`; never fetch Reep at runtime; never adopt `reep_id` as a primary key. Provider alignment between football-data.org and openfootball must be done by team-name matching at seed time, with Reep assisting via canonical names and its `key_clubeleo` mapping (used in Phase 6 to validate computed Elo).
4. **`statsbomb/open-data` (GitHub Repository):** Gold standard event-level match vectors used to train and calibrate the machine learning models.
5. **Offline Elo job (Python, seed-time only):** Computes team Elo ratings from loaded historical results and writes them as a stored column. The Go runtime never computes or fetches Elo.

## 6. Machine learning export and parity testing

The Python training pipeline exports the trained scikit-learn ensemble (Logistic Regression + Calibrated Random Forest) into structured JSON containing:
- StandardScaler `mean_` and `scale_` vectors.
- Logistic Regression coefficients and intercept.
- Random Forest decision trees converted into nested Go condition blocks via `m2cgen`.
- Platt scaling calibration parameters ($A, B$) from `CalibratedClassifierCV`.

### Mandatory golden parity test
A test suite in Go evaluates 500 game state vectors sampled from `data_pipeline/data/wc2022_game_states.json`. It asserts that the probability predictions generated by the Go engine match Python `pickle.predict_proba` within $10^{-6}$ precision across all features.

### 7.1 Locked build decisions (resolved deltas)

Forks left open by the architecture draft were resolved as follows:

1. **Elo ratings are computed offline.** No free API provides Elo; scraping eloratings.net is fragile and licensing-unclear. A Python offline job computes Elo ratings from loaded historical results during seeding and writes them to a stored column. The Go binary reads precomputed values only — zero runtime computation or dependency.
2. **New `football.db` schema, not the legacy WC2026 tables.** Multi-league standings, fixtures archives, and H2H cannot be expressed in the WC2026-specific schema. The Go port targets a clean multi-league schema (`competitions`, `teams`, `matches`, `match_events`, `win_prob_snapshots`). Legacy `wc2026.db` becomes seed *input* only, consumed by offline Python jobs. The original "existing table names" constraint is superseded by this decision.
3. **Staged league rollout, Premier League first.** Every pipeline and UI phase is built and proven against the Premier League before additional leagues are switched on via configuration (Phase 7). This de-risks rate limits, ID-crosswalk gaps, and provider quirks one league at a time.
4. **Reep is a frozen offline seed asset, not a runtime dependency.** Verified 2026-08-24: the GitHub v0 register is frozen (final release 2026.25) and lacks any football-data.org namespace. football-data.org ↔ openfootball team alignment therefore happens by name matching in the seed jobs; Reep CSVs are downloaded once into `data_pipeline/data/` for canonical names and ClubElo cross-references only. Own IDs live in `football.db`; provider mappings go in an `id_crosswalk` table.

## 7. Ordered implementation phases

Phases are **vertical slices**, not horizontal layers: each phase after the first lands something demonstrable end-to-end through the running binary. Phase 2 runs in parallel with Phases 3–5 (independent workstream).

| Phase | Description | Deliverable / Verification Criteria |
|---|---|---|
| **0. Correct & Scaffold** | Apply the locked decisions above to this document; download the frozen Reep v0 CSVs into `data_pipeline/data/` (one-time, offline); scaffold Go module, `cmd/server`, `internal/config`, Makefile targets (`go build ./...`, `sqlc generate`, `pnpm build`). | `go build ./...` passes; config loads from env (`PORT`, `DB_PATH`, `POLL_INTERVAL_SECONDS`, `FOOTBALL_DATA_API_KEY`, `DEV_MOCKS`); Reep CSVs present locally. |
| **1. Schema & Store** | Design `football.db` multi-league schema (incl. `id_crosswalk` table for provider ID mappings); write embedded migration SQL; configure sqlc; implement typed store queries; extend offline seed jobs (`uv` scripts) to load legacy WC2026 data + openfootball historical results, aligning football-data.org ↔ openfootball teams by name matching (Reep canonical names assist). | Store tests pass against a temp DB; seeded DB contains teams with crosswalk mappings, PL fixtures/results, and events. |
| **2. ML Export & Golden Parity** *(parallel workstream)* | Write `ml/export_model.py`; export scaler, logreg coefficients, RF trees (m2cgen), Platt params; generate `model_gen.go` + `calib.go`. | Golden test: 500 sampled game-state vectors match Python `predict_proba` within 1e-6. |
| **3. PL Vertical Slice — Live Scores** | `DataSource` interface + football-data.org client (retry ×3, 429 backoff, PL only); ticker poller with TTL cache and event diffing; `GET /api/health`, `GET /api/scores/live`; `DEV_MOCKS=1` canned datasets; Vite + React 19 + TanStack Router + Tailwind v4 shell; `ScoreStrip` on the index route. | `httptest` handler suite passes; single dev-mode binary serves a live (or mocked) PL score strip through the embedded-style same-origin API. |
| **4. Slice — League Hub** | `GET /api/leagues/{id}/standings` and `/fixtures`; GitHub static fallback source (openfootball/football.json) wired behind `DataSource`; `fixtures.tsx` route with season history archive. | Handler tests pass; UI renders PL standings and historical results from the seeded DB. |
| **5. Slice — Match Detail & Win Probability** | `GET /api/matches/{id}`, `/events`, `/win-probability`; poller persists inference snapshots via the Phase 2 engine; `WinProbChart` + `EventFeed` components; on-demand historical match inspector (replaces replay scrubbing). | Parity test still green; probability curves render for both live-tracked and historical matches. |
| **6. Slice — Head-to-Head Engine** | Extend offline seed job with Elo computation over historical results; `GET /api/teams/compare?home=&away=` returning H2H record, avg goals, form guide, Elo; `compare.tsx` + `H2HComparison` component. | Computed Elo sanity-checks against published references (ClubElo via Reep's `key_clubeleo`, or eloratings.net) within a small tolerance; handler tests pass. |
| **7. Multi-League Switch-On** | Config-driven league enablement; map remaining competitions (UCL, La Liga, Serie A, Bundesliga, WC2026) through the ID crosswalk; verify provider coverage per league before enabling. | Integration test returns correct standings + fixtures for every enabled league. |
| **8. Embed & Binary Assembly** | `//go:embed frontend/dist` with SPA fallback in `internal/web`; single-binary production smoke test. | One static binary serves API + frontend on a single port with no external dependencies. |
| **9. Cutover & Cleanup** | Delete FastAPI app, Next.js proxy/UI, and their dependencies. **Python stays strictly as offline tooling** (train/export/seed/Elo). Update README and AGENTS.md; final end-to-end verification of live tracking + H2H. | Repo has zero Python *runtime* dependencies; full E2E passes against the single binary. |

### 7.2 Phase dependency notes

- Phase 2 has no upstream dependency and can proceed concurrently with Phases 0–1; its output is first consumed in Phase 5.
- Phases 3–6 each independently extend the same vertical spine (source → poller → store → API → UI); they must be sequential but each ships a demoable increment.
- Phase 7 is deliberately last among feature phases so that league-specific provider quirks never block core pillar delivery.
- Rollback posture: until Phase 9, the legacy Python/Next.js stack remains runnable, so any phase can be abandoned without losing the shipping product.
