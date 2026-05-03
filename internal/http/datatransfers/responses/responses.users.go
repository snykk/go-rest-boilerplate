package responses

import (
	"time"

	"github.com/snykk/go-rest-boilerplate/internal/business/domain"
	"github.com/snykk/go-rest-boilerplate/internal/business/usecases/auth"
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

func (u *UserResponse) ToV1Domain() domain.User {
	return domain.User{
		ID:        u.Id,
		Username:  u.Username,
		Email:     u.Email,
		RoleID:    u.RoleId,
		CreatedAt: u.CreatedAt,
		UpdatedAt: u.UpdatedAt,
	}
}

// FromV1Domain projects the user entity into the response DTO. Token
// fields stay zero — the entity carries no auth artifacts. Use
// FromLoginResult on the /login and /refresh paths.
func FromV1Domain(u domain.User) UserResponse {
	return UserResponse{
		Id:        u.ID,
		Username:  u.Username,
		Email:     u.Email,
		RoleId:    u.RoleID,
		CreatedAt: u.CreatedAt,
		UpdatedAt: u.UpdatedAt,
	}
}

// FromLoginResponse is the /login + /refresh response shape: the user
// fields are the same as FromV1Domain, plus the freshly-minted token
// pair from the auth flow.
func FromLoginResponse(r auth.LoginResponse) UserResponse {
	resp := FromV1Domain(r.User)
	resp.Token = r.AccessToken
	resp.RefreshToken = r.RefreshToken
	return resp
}

func ToResponseList(domains []domain.User) []UserResponse {
	var result []UserResponse

	for _, val := range domains {
		result = append(result, FromV1Domain(val))
	}

	return result
}
