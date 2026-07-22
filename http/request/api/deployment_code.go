package api

type DeploymentClaimForm struct {
	Code     string `json:"code" validate:"required"`
	DeviceId string `json:"device_id" validate:"required"`
	Uuid     string `json:"uuid" validate:"required"`
	Hostname string `json:"hostname"`
	Os       string `json:"os"`
	Username string `json:"username"`
	Version  string `json:"version"`
	Platform string `json:"platform" validate:"required"`
}
