package mapping

import (
	"keep-it-up/internal/core/model"
	"keep-it-up/internal/infrastructure/database"
)

// ToDomainPlayer maps a stored player to the domain Player, dropping the
// persistence-only HashedPassword field.
func ToDomainPlayer(p database.Player) model.Player {
	return model.Player{ID: p.ID, Name: p.Name, Username: p.Username}
}
