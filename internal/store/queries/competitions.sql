-- name: UpsertCompetition :one
INSERT INTO competitions (code, name, country, enabled)
VALUES (@code, @name, @country, @enabled)
ON CONFLICT(code) DO UPDATE SET
    name = excluded.name,
    country = excluded.country,
    enabled = excluded.enabled
RETURNING *;

-- name: GetCompetitionByCode :one
SELECT * FROM competitions WHERE code = ? LIMIT 1;

-- name: GetEnabledCompetitions :many
SELECT * FROM competitions WHERE enabled = 1 ORDER BY code ASC;

-- name: GetCompProviderCode :one
SELECT provider_id FROM id_crosswalk
WHERE comp_code = @comp_code AND provider = @provider
LIMIT 1;
