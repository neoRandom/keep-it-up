# Keep It Up

A simple system that keeps an activity alive through periodic interaction and triggers an alert when it is left unattended for too long.

## Tech Stack

```text
Go + SQLite

Echo + SQLC + Goose
```

## API

```text
POST  /login        username + password → set cookies
GET   /games        game access → list accessible games
GET   /shared       game ID → current shared state
GET   /interactions game ID → latest interactions
POST  /save         game ID → add save interaction
POST  /play         game ID → start or resume the game
POST  /pause        game ID → pause the game
```
