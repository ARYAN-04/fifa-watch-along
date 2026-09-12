-- name: GetLiveMatches :many
SELECT * FROM matches
WHERE status = 'LIVE'
ORDER BY utc_kickoff ASC, id ASC;

-- name: GetLatestSeasonByCompetition :one
SELECT season FROM matches
WHERE competition_id = ?
ORDER BY season DESC
LIMIT 1;

-- name: GetMatchByID :one
SELECT * FROM matches WHERE id = ? LIMIT 1;

-- name: GetMatchEvents :many
SELECT * FROM match_events
WHERE match_id = ?
ORDER BY minute ASC, id ASC;

-- name: UpsertMatch :one
INSERT INTO matches (
    competition_id, external_id, season, utc_kickoff, status,
    home_team_id, away_team_id, home_goals, away_goals, minute
) VALUES (
    ?, ?, ?, ?, ?, ?, ?, ?, ?, ?
)
ON CONFLICT(external_id) DO UPDATE SET
    status = excluded.status,
    minute = excluded.minute,
    home_goals = excluded.home_goals,
    away_goals = excluded.away_goals
RETURNING *;

-- name: UpdateMatchScore :exec
UPDATE matches
SET home_goals = ?, away_goals = ?, minute = ?
WHERE id = ?;

-- name: InsertMatchEvent :one
INSERT INTO match_events (match_id, minute, type, side, player, detail)
VALUES (?, ?, ?, ?, ?, ?)
RETURNING id;
