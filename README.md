# Keep It Up

A system that coordinates periodic interactions across multiple devices and tracks when the next interaction is required.

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
  access   grant <game id> <player id>
           revoke <game id> <player id>
  player   add <name> <username> <password>
           rename <id> <name>
           passwd <id> <current password> <new password>
           passwd-force <id> <password>
           delete <id>
  auth     validate-passwd <password>
           hash-passwd <password>
           check-passwd <username> <password>
  data     games <player id>
           shared <game id>
           interactions <game id> <limit>
  session  save <game id> <player id> <duration in seconds>
           resume <game id> <player id>
           pause <game id> <player id>
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
