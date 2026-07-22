package model

import (
	"errors"

	"gorm.io/gorm"
)

// ErrAuditAppendOnly is returned when an audit event is mutated or deleted.
// Audit records are evidence and must be corrected by appending a new event.
var ErrAuditAppendOnly = errors.New("audit records are append-only")

const (
	AuditActionNew    = "new"
	AuditActionClose  = "close"
	AuditActionUpdate = "update"
)

type AuditConn struct {
	IdModel
	Action    string `json:"action" gorm:"default:'';not null;"`
	ConnId    int64  `json:"conn_id" gorm:"default:0;not null;index"`
	PeerId    string `json:"peer_id" gorm:"default:'';not null;index"`
	FromPeer  string `json:"from_peer" gorm:"default:'';not null;"`
	FromName  string `json:"from_name" gorm:"default:'';not null;"`
	Ip        string `json:"ip" gorm:"default:'';not null;"`
	SessionId string `json:"session_id" gorm:"default:'';not null;"`
	Type      int    `json:"type" gorm:"default:0;not null;"`
	Uuid      string `json:"uuid" gorm:"default:'';not null;"`
	CloseTime int64  `json:"close_time" gorm:"default:0;not null;"`
	TimeModel
}

func (*AuditConn) BeforeUpdate(*gorm.DB) error { return ErrAuditAppendOnly }
func (*AuditConn) BeforeDelete(*gorm.DB) error { return ErrAuditAppendOnly }

type AuditConnList struct {
	AuditConns []*AuditConn `json:"list"`
	Pagination
}

type AuditFile struct {
	IdModel
	FromPeer string `json:"from_peer" gorm:"default:'';not null;index"`
	Info     string `json:"info" gorm:"default:'';not null;"`
	IsFile   bool   `json:"is_file" gorm:"default:0;not null;"`
	Path     string `json:"path" gorm:"default:'';not null;"`
	PeerId   string `json:"peer_id" gorm:"default:'';not null;index"`
	Type     int    `json:"type" gorm:"default:0;not null;"`
	Uuid     string `json:"uuid" gorm:"default:'';not null;"`
	Ip       string `json:"ip" gorm:"default:'';not null;"`
	Num      int    `json:"num" gorm:"default:0;not null;"`
	FromName string `json:"from_name" gorm:"default:'';not null;"`
	TimeModel
}

func (*AuditFile) BeforeUpdate(*gorm.DB) error { return ErrAuditAppendOnly }
func (*AuditFile) BeforeDelete(*gorm.DB) error { return ErrAuditAppendOnly }

type AuditFileList struct {
	AuditFiles []*AuditFile `json:"list"`
	Pagination
}
