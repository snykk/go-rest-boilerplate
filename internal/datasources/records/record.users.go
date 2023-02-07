package records

import (
	"time"
)

type Users struct {
	Id        string     `db:"id"`
	Username  string     `db:"username"`
	Email     string     `db:"email"`
	Password  string     `db:"password"`
	Active    bool       `db:"active"`
	RoleId    int        `db:"role_id"`
	CreatedAt time.Time  `db:"created_at"`
	UpdatedAt *time.Time `db:"updated_at"`
	DeletedAt *time.Time `db:"deleted_at"`
}

