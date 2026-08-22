# Keep It Up

A system that coordinates periodic interactions across multiple devices and tracks when the next interaction is required.

>Disclimer: This system is not a game, nor was it created for a game. The project's design merely happens to coincide with the world of gaming; for this reason, I decided to use similar names to make it easier to understand and visualize.

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
  game     add    <name>
           update <id> <name>
           delete <id>
  access   grant  <game id> <player id>
           revoke <game id> <player id>
           check  <game id> <player id>
  player   add          <name> <username> <password>
           rename       <id> <name>
           passwd       <username> <current password> <new password>
           passwd-force <username> <password>
           delete       <id>
  auth     validate-passwd <password>
           hash-passwd     <password>
           check-passwd    <username> <password>
  fetch    games        <player id>
           shared       <game id>
           interactions <game id> <limit>
  command  save   <game id> <player id> <duration in seconds>
           resume <game id> <player id>
           pause  <game id> <player id>
```

## HTTP API

```text
no auth required:
POST  /login        username + password → set cookies

auth/cookies required:
GET   /games        game access → list accessible games
GET   /shared       game ID → current shared state
GET   /interactions game ID → latest interactions
POST  /save         game ID + duration in seconds → add save interaction
POST  /play         game ID → start or resume the game
POST  /pause        game ID → pause the game
```
