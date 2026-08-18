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

CREATE INDEX idx_players_username
ON players (username);

CREATE TABLE access (
    game_id INTEGER NOT NULL,
    player_id INTEGER NOT NULL,

    PRIMARY KEY (game_id, player_id),
    FOREIGN KEY (game_id) REFERENCES games(id) ON DELETE CASCADE,
    FOREIGN KEY (player_id) REFERENCES players(id) ON DELETE CASCADE
);

CREATE TABLE interactions (
    id INTEGER PRIMARY KEY,
    game_id INTEGER NOT NULL,
    player_id INTEGER,
    action TEXT NOT NULL CHECK (
        action IN ('saved', 'paused', 'resumed')
    ),

    occurred_at TEXT NOT NULL, -- Uses ISO 8601 format
    saved_by INTEGER,   -- duration in seconds, not a timestamp; NULL unless action = 'saved'

    CONSTRAINT valid_occurred_at_iso
    CHECK (occurred_at = strftime('%Y-%m-%d %H:%M:%S', occurred_at)),

    CONSTRAINT saved_by_matches_action
    CHECK ((action = 'saved') = (saved_by IS NOT NULL)),

    CONSTRAINT saved_by_positive
    CHECK (saved_by IS NULL OR saved_by > 0),

    FOREIGN KEY (game_id) REFERENCES games(id) ON DELETE CASCADE,
    FOREIGN KEY (player_id) REFERENCES players(id)
) STRICT;

CREATE INDEX idx_interactions_game_occurred
ON interactions (game_id, occurred_at DESC, id DESC);

-- +goose StatementBegin
CREATE TRIGGER trg_interactions_state_machine
BEFORE INSERT ON interactions
FOR EACH ROW
BEGIN
    SELECT RAISE(ABORT, 'occurred_at precedes an existing interaction for this game')
    WHERE EXISTS (
        SELECT 1 FROM interactions
        WHERE game_id = NEW.game_id AND occurred_at > NEW.occurred_at
    );

    SELECT RAISE(ABORT, 'cannot save while paused')
    WHERE NEW.action = 'saved'
      AND (SELECT action FROM interactions
           WHERE game_id = NEW.game_id
           ORDER BY occurred_at DESC, id DESC LIMIT 1) = 'paused';

    SELECT RAISE(ABORT, 'cannot pause: game is not currently playing')
    WHERE NEW.action = 'paused'
      AND COALESCE(
            (SELECT action FROM interactions
             WHERE game_id = NEW.game_id
             ORDER BY occurred_at DESC, id DESC LIMIT 1),
            'none'
          ) NOT IN ('saved', 'resumed');

    SELECT RAISE(ABORT, 'cannot resume: game is not currently paused')
    WHERE NEW.action = 'resumed'
      AND COALESCE(
            (SELECT action FROM interactions
             WHERE game_id = NEW.game_id
             ORDER BY occurred_at DESC, id DESC LIMIT 1),
            'none'
          ) != 'paused';
END;
-- +goose StatementEnd

-- +goose Down

DROP TRIGGER IF EXISTS trg_interactions_state_machine;

DROP INDEX idx_players_username;
DROP INDEX idx_interactions_game_occurred;

DROP TABLE interactions;
DROP TABLE access;
DROP TABLE players;
DROP TABLE games;