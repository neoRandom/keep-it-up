-- name: GrantPlayerAccess :one
INSERT INTO access (game_id, player_id)
VALUES (?, ?)
RETURNING game_id, player_id;

-- name: RevokePlayerAccess :exec
DELETE FROM access
WHERE game_id = ? AND player_id = ?;
