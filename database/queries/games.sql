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

-- name: UpdateGame :exec
UPDATE games
SET name = ?
WHERE id = ?;

-- name: DeleteGame :exec
DELETE FROM games
WHERE id = ?;