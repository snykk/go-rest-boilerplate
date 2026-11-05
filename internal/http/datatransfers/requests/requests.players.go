package requests

type CreateAnonymousPlayerRequest struct {
	InstallationId string `json:"installation_id" validate:"required,uuid"`
}