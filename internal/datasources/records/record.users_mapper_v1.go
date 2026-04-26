package records

import (
	"github.com/snykk/go-rest-boilerplate/internal/business/entities"
)

func (u *Users) ToV1Domain() entities.UserDomain {
	return entities.UserDomain{
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

func FromUsersV1Domain(u *entities.UserDomain) Users {
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

func ToArrayOfUsersV1Domain(u *[]Users) []entities.UserDomain {
	var result []entities.UserDomain

	for _, val := range *u {
		result = append(result, val.ToV1Domain())
	}

	return result
}
