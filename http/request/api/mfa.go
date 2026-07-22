package api

import "encoding/json"

type MfaCodeForm struct {
	Code     string `json:"code" validate:"required,min=6,max=10" label:"MFA验证码或备份码"`
	Password string `json:"password,omitempty" validate:"omitempty,gte=4,lte=32" label:"当前密码"`
}

type MfaEnrollmentChallengeRequest struct {
	Challenge string `json:"challenge" validate:"required,min=32,max=64" label:"MFA注册挑战"`
}

type MfaEnrollmentCompleteRequest struct {
	Challenge string `json:"challenge" validate:"required,min=32,max=64" label:"MFA注册挑战"`
	Code      string `json:"code" validate:"required,len=6" label:"MFA验证码"`
}

type PasskeyRegistrationCompleteRequest struct {
	Challenge  string          `json:"challenge" validate:"required,min=32,max=64" label:"Passkey注册挑战"`
	Credential json.RawMessage `json:"credential" validate:"required" label:"WebAuthn注册凭据"`
	Name       string          `json:"name,omitempty" validate:"omitempty,max=128" label:"设备名称"`
}

type PasskeyLoginCompleteRequest struct {
	Challenge  string            `json:"challenge" validate:"required,min=32,max=64" label:"Passkey登录挑战"`
	Credential json.RawMessage   `json:"credential" validate:"required" label:"WebAuthn登录凭据"`
	DeviceInfo DeviceInfoInLogin `json:"deviceInfo" label:"设备信息"`
	Id         string            `json:"id" label:"设备ID"`
	Uuid       string            `json:"uuid" label:"设备UUID"`
}

type PasskeyRevokeRequest struct {
	Reason string `json:"reason,omitempty" validate:"omitempty,max=255" label:"撤销原因"`
}
