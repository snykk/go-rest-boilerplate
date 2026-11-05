package records

import "github.com/go-rest-boilerplate/internal/business/domain"

func (p *Player) ToV1Domain() domain.Player {
	return domain.Player{
		Id:                p.Id,
		InstallationId:    p.InstallationId,
		RoleId:            p.RoleId,
		Email:             p.Email,
		Password:          p.Password,
		CreatedAt:         p.CreatedAt,
		UpdatedAt:         p.UpdatedAt,
		DeletedAt:         p.DeletedAt,
		PasswordChangedAt: p.PasswordChangedAt,
	}
}

func FromPlayersV1Domain(p *domain.Player) Player {
	return Player{
		Id:                p.Id,
		InstallationId:    p.InstallationId,
		RoleId:            2,
		Email:             p.Email,
		Password:          p.Password,
		CreatedAt:         p.CreatedAt,
		UpdatedAt:         p.UpdatedAt,
		DeletedAt:         p.DeletedAt,
		PasswordChangedAt: p.PasswordChangedAt,
	}
}