-- name: ListPlayerGames :many
SELECT
    games.id,
    games.name
FROM games
JOIN access ON access.game_id = games.id
WHERE access.player_id = ?;

-- name: ListRecentInteractions :many
SELECT id, game_id, player_id, action, occurred_at, saved_by
FROM interactions
WHERE game_id = ?
ORDER BY occurred_at DESC, id DESC
LIMIT ?;

-- name: ListInteractionsForReplay :many
SELECT id, game_id, player_id, action, occurred_at, saved_by
FROM interactions
WHERE game_id = ?
ORDER BY occurred_at ASC, id ASC;

-- name: ListPlayerInteractions :many
SELECT id, game_id, player_id, action, occurred_at, saved_by
FROM interactions
WHERE player_id = ?
ORDER BY occurred_at DESC, id DESC;