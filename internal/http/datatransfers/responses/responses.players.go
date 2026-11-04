package responses

import (
	"time"

	"github.com/go-rest-boilerplate/internal/business/domain"
)

type CreateAnonymousPlayerResponse struct {
	Id             string     `json:"id"`
	RoleId         int        `json:"role_id"`
	InstallationID string     `json:"installation_id"`
	Token          string     `json:"token,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      *time.Time `json:"updated_at,omitempty"`
}

func PlayerFromV1Domain(p domain.Player) CreateAnonymousPlayerResponse {
	return CreateAnonymousPlayerResponse{
		Id:             p.ID,
		RoleId:         p.RoleID,
		InstallationID: p.InstallationID,
		CreatedAt:      p.CreatedAt,
	}
}