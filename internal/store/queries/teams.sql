-- name: GetTeamByID :one
SELECT * FROM teams WHERE id = ? LIMIT 1;

-- name: GetOrCreateTeam :one
INSERT INTO teams (name, short_name, country)
VALUES (@name, @short_name, @country)
ON CONFLICT(name) DO UPDATE SET
    short_name = excluded.short_name,
    country = excluded.country
RETURNING *;

-- name: GetTeamCrosswalk :many
SELECT * FROM id_crosswalk WHERE team_id = ?;

-- name: UpsertCrosswalk :one
INSERT INTO id_crosswalk (team_id, comp_code, provider, provider_id)
VALUES (?, ?, ?, ?)
ON CONFLICT(team_id, provider) DO UPDATE SET
    provider_id = excluded.provider_id,
    comp_code = excluded.comp_code
RETURNING *;

-- name: GetEloRating :one
SELECT rating, updated_at FROM elo_ratings WHERE team_id = ?;

-- name: ListFinishedBetweenTeams :many
SELECT m.id, m.season, m.utc_kickoff, m.home_team_id, m.away_team_id,
       m.home_goals, m.away_goals,
       h.name AS home_name, a.name AS away_name
FROM matches m
JOIN teams h ON h.id = m.home_team_id
JOIN teams a ON a.id = m.away_team_id
WHERE m.status = 'FINISHED'
  AND ((m.home_team_id = @team_a AND m.away_team_id = @team_b)
    OR (m.home_team_id = @team_b AND m.away_team_id = @team_a))
ORDER BY m.utc_kickoff DESC, m.id DESC;

-- name: ListLastFinishedByTeam :many
SELECT m.id, m.season, m.utc_kickoff, m.home_team_id, m.away_team_id,
       m.home_goals, m.away_goals,
       h.name AS home_name, a.name AS away_name
FROM matches m
JOIN teams h ON h.id = m.home_team_id
JOIN teams a ON a.id = m.away_team_id
WHERE m.status = 'FINISHED'
  AND (m.home_team_id = @team OR m.away_team_id = @team)
ORDER BY m.utc_kickoff DESC, m.id DESC
LIMIT 5;

-- name: UpsertEloRating :exec
INSERT INTO elo_ratings (team_id, rating, updated_at)
VALUES (?, ?, ?)
ON CONFLICT(team_id) DO UPDATE SET
    rating = excluded.rating,
    updated_at = excluded.updated_at;
