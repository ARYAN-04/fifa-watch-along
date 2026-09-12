-- name: GetStandingsByCompetition :many
WITH finished AS (
    SELECT home_team_id AS team_id, home_goals AS gf, away_goals AS ga
    FROM matches
    WHERE competition_id = @competition_id
      AND status IN ('FINISHED', 'LIVE')
      AND home_goals IS NOT NULL AND away_goals IS NOT NULL
    UNION ALL
    SELECT away_team_id AS team_id, away_goals AS gf, home_goals AS ga
    FROM matches
    WHERE competition_id = @competition_id
      AND status IN ('FINISHED', 'LIVE')
      AND home_goals IS NOT NULL AND away_goals IS NOT NULL
),
agg AS (
    SELECT team_id,
           COUNT(*) AS played,
           SUM(gf) AS goals_for,
           SUM(ga) AS goals_against,
           SUM(CASE WHEN gf > ga THEN 1 ELSE 0 END) AS won,
           SUM(CASE WHEN gf = ga THEN 1 ELSE 0 END) AS drawn,
           SUM(CASE WHEN gf < ga THEN 1 ELSE 0 END) AS lost
    FROM finished
    GROUP BY team_id
)
SELECT
    t.id AS team_id,
    t.name AS team_name,
    t.short_name,
    COALESCE(a.played, 0) AS played,
    COALESCE(a.won, 0) AS won,
    COALESCE(a.drawn, 0) AS drawn,
    COALESCE(a.lost, 0) AS lost,
    COALESCE(a.goals_for, 0) AS goals_for,
    COALESCE(a.goals_against, 0) AS goals_against,
    COALESCE(a.goals_for, 0) - COALESCE(a.goals_against, 0) AS goal_diff,
    COALESCE(a.won, 0) * 3 + COALESCE(a.drawn, 0) AS points
FROM teams t
LEFT JOIN agg a ON a.team_id = t.id
WHERE EXISTS (
    SELECT 1 FROM matches m
    WHERE (m.home_team_id = t.id OR m.away_team_id = t.id)
      AND m.competition_id = @competition_id
)
ORDER BY points DESC, goal_diff DESC, goals_for DESC, team_name ASC;
