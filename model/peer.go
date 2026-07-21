package model

// Peer 设备审批状态
const (
	PeerStatusPending  = 0 // 待审批
	PeerStatusApproved = 1 // 已通过
)

type Peer struct {
	RowId          uint   `json:"row_id" gorm:"primaryKey;"`
	Id             string `json:"id"  gorm:"default:'';not null;index"`
	Cpu            string `json:"cpu"  gorm:"default:'';not null;"`
	Hostname       string `json:"hostname"  gorm:"default:'';not null;"`
	Memory         string `json:"memory"  gorm:"default:'';not null;"`
	Os             string `json:"os"  gorm:"default:'';not null;"`
	Username       string `json:"username"  gorm:"default:'';not null;"`
	Uuid           string `json:"uuid"  gorm:"default:'';not null;index"`
	Version        string `json:"version"  gorm:"default:'';not null;"`
	UserId         uint   `json:"user_id"  gorm:"default:0;not null;index"`
	User           *User  `json:"user,omitempty"`
	LastOnlineTime int64  `json:"last_online_time"  gorm:"default:0;not null;"`
	LastOnlineIp   string `json:"last_online_ip"  gorm:"default:'';not null;"`
	GroupId        uint   `json:"group_id"  gorm:"default:0;not null;index"`
	Alias          string `json:"alias" gorm:"default:'';not null;index"`
	// Status 设备审批状态：0 待审批 / 1 已通过。
	// gorm default:1 用于兼容存量数据（AutoMigrate 加列后存量记录即为已通过，
	// 另有 apimain 的版本迁移兜底）。注意：由于带 default 标签，gorm 结构体
	// Create/Updates 会忽略该字段的零值(0)，需要显式置 0 时（如新设备待审批）
	// 必须使用 PeerService.UpdateStatus 单字段更新。
	Status int `json:"status" gorm:"default:1;not null;index"`
	TimeModel
}

type PeerList struct {
	Peers []*Peer `json:"list"`
	Pagination
}
