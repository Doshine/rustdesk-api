package model

// RelayNode 中继节点探测状态
const (
	RelayNodeStatusUnknown = 0 // 未知（未探测）
	RelayNodeStatusOnline  = 1 // 在线
	RelayNodeStatusOffline = 2 // 离线
)

// RelayNodeDefaultPort 中继（hbbr）默认端口
const RelayNodeDefaultPort = 21117

// RelayNode 中继节点（hbbr）。企业可持续增配自建中继，后台可视化管理。
// Status/LastLatencyMs/LastCheckedAt 由探测接口回写，不由表单维护。
type RelayNode struct {
	RowId         uint   `json:"row_id" gorm:"primaryKey;"`
	Name          string `json:"name" gorm:"default:'';not null;index"`
	Host          string `json:"host" gorm:"default:'';not null;"`
	Port          int    `json:"port" gorm:"default:21117;not null;"`
	PublicKey     string `json:"public_key" gorm:"default:'';not null;"`     // 可选，节点公钥（-k key）
	Region        string `json:"region" gorm:"default:'';not null;"`         // 区域/城市
	Isp           string `json:"isp" gorm:"default:'';not null;"`            // 运营商
	Priority      int    `json:"priority" gorm:"default:0;not null;"`        // 优先级权重，越大越优先
	Enabled       bool   `json:"enabled" gorm:"not null;"`                   // 开关：false 不参与分配。不带 default 标签，避免 gorm 忽略零值 false
	Status        int    `json:"status" gorm:"default:0;not null;index"`     // 探测状态：0未知/1在线/2离线
	LastLatencyMs int64  `json:"last_latency_ms" gorm:"default:0;not null;"` // 最近一次探测延迟（毫秒）
	LastCheckedAt int64  `json:"last_checked_at" gorm:"default:0;not null;"` // 最近一次探测时间（Unix 秒）
	Remark        string `json:"remark" gorm:"default:'';not null;"`
	TimeModel
}

type RelayNodeList struct {
	RelayNodes []*RelayNode `json:"list"`
	Pagination
}
