-- name: GetGame :one
SELECT
    id,
    name
FROM games
WHERE id = ?;

-- name: CreateGame :one
INSERT INTO games (name)
VALUES (?)
RETURNING id, name;