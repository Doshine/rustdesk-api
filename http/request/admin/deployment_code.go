package admin

type DeploymentCodeCreateForm struct {
	Name             string   `json:"name" validate:"required"`
	DeviceGroupId    uint     `json:"device_group_id"`
	AllowedPlatforms []string `json:"allowed_platforms" validate:"required"`
	ExpiresAt        int64    `json:"expires_at" validate:"required,gt=0"`
	MaxUses          int      `json:"max_uses" validate:"required,gt=0,lte=10000"`
}

type DeploymentCodeIdForm struct {
	Id uint `json:"id" validate:"required,gt=0"`
}

type DeploymentCodeQuery struct {
	PageQuery
	Status *int `json:"status" form:"status"`
}

type DeploymentAuditQuery struct {
	PageQuery
	DeploymentCodeId uint `json:"deployment_code_id" form:"deployment_code_id" validate:"required,gt=0"`
}
