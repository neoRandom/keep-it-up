-- name: GetPlayer :one
SELECT
    id,
    name,
    username,
    hashed_password
FROM players
WHERE id = ?;

-- name: GetPlayerByUsername :one
SELECT
    id,
    name,
    username,
    hashed_password
FROM players
WHERE username = ?;

-- name: CreatePlayer :one
INSERT INTO players (name, username, hashed_password)
VALUES (?, ?, ?)
RETURNING id, name, username, hashed_password;

-- name: UpdatePlayerName :exec
UPDATE players
SET name = ?
WHERE id = ?;

-- name: UpdatePlayerPassword :exec
UPDATE players
SET hashed_password = ?
WHERE id = ?;

-- name: DeletePlayer :exec
DELETE FROM players
WHERE id = ?;
