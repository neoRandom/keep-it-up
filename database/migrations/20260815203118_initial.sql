-- +goose Up

CREATE TABLE games (
    id INTEGER PRIMARY KEY,
    name TEXT NOT NULL
);

CREATE TABLE players (
    id INTEGER PRIMARY KEY,
    name TEXT NOT NULL,
    username TEXT NOT NULL UNIQUE,
    hashed_password TEXT NOT NULL
);

CREATE TABLE access (
    game_id INTEGER NOT NULL,
    player_id INTEGER NOT NULL,

    PRIMARY KEY (game_id, player_id),
    FOREIGN KEY (game_id) REFERENCES games(id),
    FOREIGN KEY (player_id) REFERENCES players(id)
);

CREATE TABLE interactions (
    id INTEGER PRIMARY KEY,
    game_id INTEGER NOT NULL,
    player_id INTEGER NOT NULL,
    action TEXT NOT NULL CHECK (
        action IN ('saved', 'paused', 'resumed')
    ),
    occurred_at TEXT NOT NULL,

    FOREIGN KEY (game_id) REFERENCES games(id),
    FOREIGN KEY (player_id) REFERENCES players(id)
);

-- +goose Down

DROP TABLE interactions;
DROP TABLE access;
DROP TABLE players;
DROP TABLE games;