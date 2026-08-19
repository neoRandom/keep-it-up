-- name: SaveGame :one
INSERT INTO interactions 
(
    game_id, 
    player_id, 
    action, 
    occurred_at, 
    saved_by
)
VALUES (?, ?, 'saved', ?, ?)
RETURNING id, game_id, player_id, action, occurred_at, saved_by;

-- name: ResumeGame :one
INSERT INTO interactions 
(
    game_id, 
    player_id, 
    action, 
    occurred_at
)
VALUES (?, ?, 'resumed', ?)
RETURNING id, game_id, player_id, action, occurred_at;

-- name: PauseGame :one
INSERT INTO interactions 
(
    game_id, 
    player_id, 
    action, 
    occurred_at
)
VALUES (?, ?, 'paused', ?)
RETURNING id, game_id, player_id, action, occurred_at;
