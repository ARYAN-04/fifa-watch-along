-- name: ListFixturesByLeagueAndSeason :many
SELECT m.*, h.name AS home_team_name, a.name AS away_team_name
FROM matches m
JOIN teams h ON h.id = m.home_team_id
JOIN teams a ON a.id = m.away_team_id
WHERE m.competition_id = @competition_id AND m.season = @season
ORDER BY m.utc_kickoff ASC, m.id ASC;

-- name: ListFinishedByLeagueAndSeason :many
SELECT m.*, h.name AS home_team_name, a.name AS away_team_name
FROM matches m
JOIN teams h ON h.id = m.home_team_id
JOIN teams a ON a.id = m.away_team_id
WHERE m.competition_id = @competition_id AND m.season = @season AND m.status = 'FINISHED'
ORDER BY m.utc_kickoff DESC, m.id DESC;
