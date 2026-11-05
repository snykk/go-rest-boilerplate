package postgres

import (
	"github.com/jmoiron/sqlx"
	repointerface "github.com/go-rest-boilerplate/internal/datasources/repositories/interface"
)

type postgresPlayersRepository struct {
	conn *sqlx.DB
}

func NewPlayersRepository(conn *sqlx.DB) repointerface.PlayerRepository {
	return &postgresPlayersRepository{conn: conn}
}
