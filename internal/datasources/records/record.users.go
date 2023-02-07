package records

import (
	"time"

	"github.com/snykk/go-rest-boilerplate/internal/business/domains"
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

func (u *Users) ToDomain() domains.UserDomain {
	return domains.UserDomain{
		ID:        u.Id,
		Username:  u.Username,
		Email:     u.Email,
		Password:  u.Password,
		Active:    u.Active,
		RoleID:    u.RoleId,
		CreatedAt: u.CreatedAt,
		UpdatedAt: u.UpdatedAt,
	}
}

func FromUsersDomain(u *domains.UserDomain) Users {
	return Users{
		Id:        u.ID,
		Username:  u.Username,
		Email:     u.Email,
		Password:  u.Password,
		Active:    u.Active,
		RoleId:    u.RoleID,
		CreatedAt: u.CreatedAt,
		UpdatedAt: u.UpdatedAt,
	}
}

func ToArrayOfUsersDomain(u *[]Users) []domains.UserDomain {
	var result []domains.UserDomain

	for _, val := range *u {
		result = append(result, val.ToDomain())
	}

	return result
}
