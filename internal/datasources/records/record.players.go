package records

import "time"

type Player struct {
	Id                string     `db:"id"`
	InstallationId    string     `db:"installation_id"`
	RoleId            int        `db:"role_id"`
	Email             string     `db:"email"`
	Password          string     `db:"password"`
	CreatedAt         time.Time  `db:"created_at"`
	UpdatedAt         *time.Time `db:"updated_at"`
	DeletedAt         *time.Time `db:"deleted_at"`
	PasswordChangedAt *time.Time `db:"password_changed_at"`

}