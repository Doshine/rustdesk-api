package model

import (
	"encoding/json"
	"errors"

	"gorm.io/gorm"
)

const (
	DeploymentCodeStatusActive  = 1
	DeploymentCodeStatusRevoked = 2
)

const (
	DeploymentAuditCreated = "created"
	DeploymentAuditClaimed = "claimed"
	DeploymentAuditRevoked = "revoked"
	DeploymentAuditRotated = "rotated"
)

var ErrDeploymentAuditAppendOnly = errors.New("deployment audit events are append-only")

// DeploymentCode is a scoped, short-lived device enrollment credential.
// Only the SHA-256 digest is persisted; the plaintext is returned once when
// the code is created or rotated.
type DeploymentCode struct {
	IdModel
	Name             string `json:"name" gorm:"default:'';not null;"`
	CodeHash         string `json:"-" gorm:"size:64;uniqueIndex;not null;"`
	CodeHint         string `json:"code_hint" gorm:"default:'';not null;"`
	DeviceGroupId    uint   `json:"device_group_id" gorm:"default:0;not null;index"`
	AllowedPlatforms string `json:"allowed_platforms" gorm:"type:text;not null;"`
	ExpiresAt        int64  `json:"expires_at" gorm:"default:0;not null;index"`
	MaxUses          int    `json:"max_uses" gorm:"default:1;not null;"`
	UsedCount        int    `json:"used_count" gorm:"default:0;not null;"`
	Status           int    `json:"status" gorm:"default:1;not null;index"`
	CreatedBy        uint   `json:"created_by" gorm:"default:0;not null;index"`
	RevokedBy        uint   `json:"revoked_by" gorm:"default:0;not null;"`
	RevokedAt        int64  `json:"revoked_at" gorm:"default:0;not null;"`
	RotatedFromId    uint   `json:"rotated_from_id" gorm:"default:0;not null;index"`
	TimeModel
}

func (d *DeploymentCode) PlatformList() []string {
	var platforms []string
	if err := json.Unmarshal([]byte(d.AllowedPlatforms), &platforms); err != nil {
		return []string{}
	}
	return platforms
}

type DeploymentCodeList struct {
	DeploymentCodes []*DeploymentCode `json:"list"`
	Pagination
}

// DeploymentAuditEvent is separate from connection/file audit so enrollment
// evidence stays explicit and can have stricter database privileges.
type DeploymentAuditEvent struct {
	IdModel
	Action           string `json:"action" gorm:"default:'';not null;index"`
	DeploymentCodeId uint   `json:"deployment_code_id" gorm:"default:0;not null;index"`
	ActorUserId      uint   `json:"actor_user_id" gorm:"default:0;not null;index"`
	PeerRowId        uint   `json:"peer_row_id" gorm:"default:0;not null;index"`
	DeviceId         string `json:"device_id" gorm:"default:'';not null;index"`
	Uuid             string `json:"uuid" gorm:"default:'';not null;index"`
	Ip               string `json:"ip" gorm:"default:'';not null;"`
	Detail           string `json:"detail" gorm:"default:'';not null;"`
	TimeModel
}

func (*DeploymentAuditEvent) BeforeUpdate(*gorm.DB) error { return ErrDeploymentAuditAppendOnly }
func (*DeploymentAuditEvent) BeforeDelete(*gorm.DB) error { return ErrDeploymentAuditAppendOnly }

type DeploymentAuditEventList struct {
	Events []*DeploymentAuditEvent `json:"list"`
	Pagination
}
