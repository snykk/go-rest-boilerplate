package postgres

import (
	"context"
	"fmt"
	"strings"

	"github.com/snykk/go-rest-boilerplate/internal/business/domain"
	repointerface "github.com/snykk/go-rest-boilerplate/internal/datasources/repositories/interface"
	"github.com/snykk/go-rest-boilerplate/internal/datasources/records"
)

// hardLimit caps List page size so a misbehaving caller can't pull
// the whole table in one request. Repeating the cap here (in addition
// to whatever clamping the handler does) is defense in depth.
const hardLimit = 200

func (r *postgreUserRepository) List(ctx context.Context, filter repointerface.UserListFilter, offset, limit int) ([]domain.User, error) {
	if limit <= 0 || limit > hardLimit {
		limit = hardLimit
	}
	if offset < 0 {
		offset = 0
	}

	// Build the WHERE clause dynamically. Each predicate appends an
	// $N placeholder + value to keep the query parameterized — never
	// concatenate filter values into the SQL string.
	var (
		where = []string{}
		args  = []interface{}{}
		idx   = 1
	)
	if !filter.IncludeDeleted {
		where = append(where, "deleted_at IS NULL")
	}
	if filter.RoleID != 0 {
		where = append(where, fmt.Sprintf("role_id = $%d", idx))
		args = append(args, filter.RoleID)
		idx++
	}
	if filter.ActiveOnly {
		where = append(where, "active = true")
	}

	query := "SELECT * FROM users"
	if len(where) > 0 {
		query += " WHERE " + strings.Join(where, " AND ")
	}
	query += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", idx, idx+1)
	args = append(args, limit, offset)

	var rows []records.Users
	if err := r.conn.SelectContext(ctx, &rows, query, args...); err != nil {
		return nil, err
	}
	out := make([]domain.User, 0, len(rows))
	for i := range rows {
		out = append(out, rows[i].ToV1Domain())
	}
	return out, nil
}
