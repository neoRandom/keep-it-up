# Keep It Up

A simple system that keeps an activity alive through periodic interaction and triggers an alert when it is left unattended for too long.

## Tech Stack

```text
Go + SQLite

Echo + SQLC + Goose
```

## CLI Management

```text
Usage:
  keepitup <noun> <verb> [args...]

Nouns:
  game     add <name>
           update <id> <name>
           delete <id>
  access   grant <gameId> <playerId>
           revoke <gameId> <playerId>
  player   add <name> <username> <password>
           rename <id> <name>
           passwd <id> <currentPassword> <newPassword>
           passwd-force <id> <password>
           delete <id>
  auth     validate-passwd <password>
           hash-passwd <password>
           check-passwd <username> <password>
  data     games <playerId>
           shared <gameId>
           interactions <gameId> <count>
  session  save <gameId> <playerId> <RFC3339 timestamp>
           resume <gameId> <playerId>
           pause <gameId> <playerId>
```

## HTTP API

```text
POST  /login        username + password → set cookies
GET   /games        game access → list accessible games
GET   /shared       game ID → current shared state
GET   /interactions game ID → latest interactions
POST  /save         game ID → add save interaction
POST  /play         game ID → start or resume the game
POST  /pause        game ID → pause the game
```
