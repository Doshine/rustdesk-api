package model

// PasskeyCredential stores the WebAuthn credential record and its lifecycle
// state. CredentialData is a serialized go-webauthn credential and contains
// only public credential material; no authenticator private key is ever sent to
// or stored by the server.
type PasskeyCredential struct {
	IdModel
	UserId           uint   `json:"user_id" gorm:"default:0;not null;index:idx_passkey_user_rp,priority:1"`
	RPID             string `json:"rp_id" gorm:"default:'';not null;index:idx_passkey_user_rp,priority:2;uniqueIndex:idx_passkey_credential_rp,priority:1"`
	CredentialId     string `json:"credential_id" gorm:"type:varchar(512);default:'';not null;uniqueIndex:idx_passkey_credential_rp,priority:2"`
	UserHandle       string `json:"-" gorm:"type:varchar(128);default:'';not null;index"`
	Name             string `json:"name" gorm:"type:varchar(128);default:'';not null"`
	CredentialData   string `json:"-" gorm:"type:text;not null"`
	FlagsRaw         uint8  `json:"-" gorm:"default:0;not null"`
	SignCount        uint32 `json:"sign_count" gorm:"default:0;not null"`
	CloneWarning     bool   `json:"clone_warning" gorm:"default:0;not null"`
	BackupEligible   bool   `json:"backup_eligible" gorm:"default:0;not null"`
	BackupState      bool   `json:"backup_state" gorm:"default:0;not null"`
	LastUsedAt       int64  `json:"last_used_at" gorm:"default:0;not null"`
	RevokedAt        *int64 `json:"revoked_at,omitempty"`
	RevokedBy        uint   `json:"revoked_by,omitempty" gorm:"default:0;not null"`
	RevocationReason string `json:"revocation_reason,omitempty" gorm:"type:varchar(255);default:'';not null"`
	TimeModel
}

type PasskeyCredentialView struct {
	Id               uint   `json:"id"`
	UserId           uint   `json:"user_id"`
	RPID             string `json:"rp_id"`
	CredentialIdHint string `json:"credential_id_hint"`
	Name             string `json:"name"`
	SignCount        uint32 `json:"sign_count"`
	CloneWarning     bool   `json:"clone_warning"`
	BackupEligible   bool   `json:"backup_eligible"`
	BackupState      bool   `json:"backup_state"`
	LastUsedAt       int64  `json:"last_used_at"`
	RevokedAt        *int64 `json:"revoked_at,omitempty"`
	RevocationReason string `json:"revocation_reason,omitempty"`
}

func (p *PasskeyCredential) View() *PasskeyCredentialView {
	if p == nil {
		return nil
	}
	hint := p.CredentialId
	if len(hint) > 8 {
		hint = hint[:4] + "..." + hint[len(hint)-4:]
	}
	return &PasskeyCredentialView{
		Id: p.Id, UserId: p.UserId, RPID: p.RPID, CredentialIdHint: hint,
		Name: p.Name, SignCount: p.SignCount, CloneWarning: p.CloneWarning,
		BackupEligible: p.BackupEligible, BackupState: p.BackupState,
		LastUsedAt: p.LastUsedAt, RevokedAt: p.RevokedAt,
		RevocationReason: p.RevocationReason,
	}
}
