package responses

import (
	"time"

	"github.com/snykk/go-rest-boilerplate/internal/business/entities"
)

type UserResponse struct {
	Id           string     `json:"id"`
	Username     string     `json:"username"`
	Email        string     `json:"email"`
	RoleId       int        `json:"role_id"`
	Token        string     `json:"token,omitempty"`
	RefreshToken string     `json:"refresh_token,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    *time.Time `json:"updated_at"`
}

func (u *UserResponse) ToV1Domain() entities.UserDomain {
	return entities.UserDomain{
		ID:        u.Id,
		Username:  u.Username,
		Email:     u.Email,
		RoleID:    u.RoleId,
		CreatedAt: u.CreatedAt,
		UpdatedAt: u.UpdatedAt,
	}
}

func FromV1Domain(u entities.UserDomain) UserResponse {
	return UserResponse{
		Id:           u.ID,
		Username:     u.Username,
		Email:        u.Email,
		Token:        u.Token,
		RefreshToken: u.RefreshToken,
		RoleId:       u.RoleID,
		CreatedAt:    u.CreatedAt,
		UpdatedAt:    u.UpdatedAt,
	}
}

func ToResponseList(domains []entities.UserDomain) []UserResponse {
	var result []UserResponse

	for _, val := range domains {
		result = append(result, FromV1Domain(val))
	}

	return result
}
