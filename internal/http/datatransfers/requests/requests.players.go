package requests

type CreateAnonymousPlayerRequest struct {
	InstallationID string `json:"installation_id" validate:"required,uuid"`
}