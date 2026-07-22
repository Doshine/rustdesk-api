package api

type OidcAuthRequest struct {
	DeviceInfo DeviceInfoInLogin `json:"deviceInfo" label:"设备信息"`
	Id         string            `json:"id"  label:"id"`
	Op         string            `json:"op" label:"op"`
	Uuid       string            `json:"uuid"  label:"uuid"`
}

type OidcAuthQuery struct {
	Code string `json:"code" form:"code" label:"code"`
	Id   string `json:"id" form:"id" label:"id"`
	Uuid string `json:"uuid" form:"uuid" label:"uuid"`
}

type MfaChallengeRequest struct {
	Challenge string `json:"challenge" validate:"required,min=32,max=64" label:"MFA挑战"`
	Code      string `json:"code" validate:"required,min=6,max=10" label:"MFA验证码或备份码"`
}
