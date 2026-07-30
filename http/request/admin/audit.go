package admin

type AuditQuery struct {
	PeerId   string `form:"peer_id"`
	FromPeer string `form:"from_peer"`
	// Uuid 按会话精确过滤。审计时间线（设计稿 §5.5）要把同一次会话的连接与
	// 文件事件排成一条经过，只能按 uuid 取；用 peer_id 拉再在前端筛，会话文件
	// 跨页时就会漏事件，而审计里漏事件比慢更严重。
	Uuid string `form:"uuid"`
	PageQuery
}

type AuditConnLogIds struct {
	Ids []uint `json:"ids" validate:"required"`
}
type AuditFileLogIds struct {
	Ids []uint `json:"ids" validate:"required"`
}
