package mapping

import (
	"keep-it-up/internal/core/model"
	"keep-it-up/internal/infrastructure/database"
)

// ToDomainGame maps a stored game to the domain Game.
func ToDomainGame(g database.Game) model.Game {
	return model.Game{ID: g.ID, Name: g.Name}
}

// ToDomainGames maps a slice of stored games to domain Games.
func ToDomainGames(gs []database.Game) []model.Game {
	out := make([]model.Game, 0, len(gs))
	for _, g := range gs {
		out = append(out, ToDomainGame(g))
	}
	return out
}
