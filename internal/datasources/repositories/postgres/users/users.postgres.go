package postgres

import (
	"github.com/jmoiron/sqlx"
	repointerface "github.com/snykk/go-rest-boilerplate/internal/datasources/repositories/interface"
)

// postgreUserRepository carries the sqlx handle. Each interface method
// lives in its own file (users.store.go, users.get_by_email.go, ...)
// so PR diffs touching one query stay surgical.
type postgreUserRepository struct {
	conn *sqlx.DB
}

func NewUserRepository(conn *sqlx.DB) repointerface.UserRepository {
	return &postgreUserRepository{conn: conn}
}
