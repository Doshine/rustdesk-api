package model

type UserToken struct {
	IdModel
	UserId     uint   `json:"user_id" gorm:"default:0;not null;index"`
	DeviceUuid string `json:"device_uuid" gorm:"default:'';omitempty;"`
	DeviceId   string `json:"device_id" gorm:"default:'';omitempty;"`
	// PasskeyCredentialId links a session to the credential that created it.
	// It is hidden from API responses and remains zero for other login methods.
	PasskeyCredentialId uint `json:"-" gorm:"default:0;not null;index"`
	// Token is a bearer credential and must never be serialized in list/detail APIs.
	Token     string `json:"-" gorm:"default:'';not null;index"`
	TokenHint string `json:"token_hint,omitempty" gorm:"-"`
	ExpiredAt int64  `json:"expired_at" gorm:"default:0;not null;"`
	TimeModel
}

type UserTokenList struct {
	UserTokens []UserToken `json:"list"`
	Pagination
}
