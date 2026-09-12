-- name: InsertWinProbSnapshot :exec
INSERT INTO win_prob_snapshots (match_id, minute, home, draw, away)
VALUES (?, ?, ?, ?, ?)
ON CONFLICT(match_id, minute) DO UPDATE SET
    home = excluded.home,
    draw = excluded.draw,
    away = excluded.away;

-- name: ListWinProbSnapshotsByMatch :many
SELECT * FROM win_prob_snapshots
WHERE match_id = ?
ORDER BY minute ASC;
