-- name: AddInteraction :one
INSERT INTO interactions (
    game_id,
    player_id,
    action,
    occurred_at
)
VALUES (?, ?, ?, ?)
RETURNING id, game_id, player_id, action, occurred_at;