package postgres

import (
	"context"

	"github.com/jmoiron/sqlx"
	"github.com/snykk/go-rest-boilerplate/internal/business/domains"
	"github.com/snykk/go-rest-boilerplate/internal/datasources/records"
)

type postgreUserRepository struct {
	conn *sqlx.DB
}

func NewUserRepository(conn *sqlx.DB) domains.UserRepository {
	return &postgreUserRepository{
		conn: conn,
	}
}

func (r *postgreUserRepository) Store(ctx context.Context, inDom *domains.UserDomain) (err error) {
	userRecord := records.FromUsersDomain(inDom)

	_, err = r.conn.NamedQueryContext(ctx, `INSERT INTO users(id, username, email, password, active, role_id, created_at) VALUES (uuid_generate_v4(), :username, :email, :password, false, :role_id, :created_at)`, userRecord)
	if err != nil {
		return err
	}

	return nil
}

func (r *postgreUserRepository) GetByEmail(ctx context.Context, inDom *domains.UserDomain) (outDomain domains.UserDomain, err error) {
	userRecord := records.FromUsersDomain(inDom)

	err = r.conn.GetContext(ctx, &userRecord, `SELECT * FROM users WHERE "email" = $1`, userRecord.Email)
	if err != nil {
		return domains.UserDomain{}, err
	}

	return userRecord.ToDomain(), nil
}

func (r *postgreUserRepository) ChangeActiveUser(ctx context.Context, inDom *domains.UserDomain) (err error) {
	userRecord := records.FromUsersDomain(inDom)

	_, err = r.conn.NamedQueryContext(ctx, `UPDATE users SET active = :active WHERE id = :id`, userRecord)

	return
}
