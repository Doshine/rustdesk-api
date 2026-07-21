package admin

import "github.com/lejianwen/rustdesk-api/v2/model"

// RelayNodeForm 中继节点表单
type RelayNodeForm struct {
	RowId     uint   `json:"row_id"`
	Name      string `json:"name" validate:"required"`
	Host      string `json:"host" validate:"required"`
	Port      int    `json:"port" validate:"gte=0,lte=65535"` // 0/不传 = 默认 21117
	PublicKey string `json:"public_key"`                      // 可选，节点公钥
	Region    string `json:"region"`
	Isp       string `json:"isp"`
	Priority  int    `json:"priority"`
	Enabled   *bool  `json:"enabled"` // 指针：区分"未传（默认启用）"与"显式关闭"
	Remark    string `json:"remark"`
}

// ToRelayNode 转换为模型。Port 缺省回落 21117；Enabled 缺省为 true。
func (f *RelayNodeForm) ToRelayNode() *model.RelayNode {
	port := f.Port
	if port <= 0 {
		port = model.RelayNodeDefaultPort
	}
	enabled := true
	if f.Enabled != nil {
		enabled = *f.Enabled
	}
	return &model.RelayNode{
		RowId:     f.RowId,
		Name:      f.Name,
		Host:      f.Host,
		Port:      port,
		PublicKey: f.PublicKey,
		Region:    f.Region,
		Isp:       f.Isp,
		Priority:  f.Priority,
		Enabled:   enabled,
		Remark:    f.Remark,
	}
}

// RelayNodeQuery 列表查询
type RelayNodeQuery struct {
	PageQuery
	Name string `json:"name" form:"name"` // 名称模糊匹配
}

// RelayNodeBatchDeleteForm 批量删除
type RelayNodeBatchDeleteForm struct {
	RowIds []uint `json:"row_ids" validate:"required"`
}

// RelayNodeTestForm 连通性测试。两种入参二选一：
// 1) {"row_id": N}        —— 用库内节点配置测试（结果回写 status/last_latency_ms/last_checked_at）
// 2) {"host","port"}      —— 直接测试（新增前预检，不写库）
type RelayNodeTestForm struct {
	RowId uint   `json:"row_id"`
	Host  string `json:"host"`
	Port  int    `json:"port"`
}
