package admin

import "github.com/lejianwen/rustdesk-api/v2/model"

type PeerForm struct {
	RowId    uint   `json:"row_id" `
	Id       string `json:"id"`
	Cpu      string `json:"cpu"`
	Hostname string `json:"hostname"`
	Memory   string `json:"memory"`
	Os       string `json:"os"`
	Username string `json:"username"`
	Uuid     string `json:"uuid"`
	Version  string `json:"version"`
	GroupId  uint   `json:"group_id"`
	Alias    string `json:"alias"`
}

type PeerBatchDeleteForm struct {
	RowIds []uint `json:"row_ids" validate:"required"`
}

// PeerBatchApproveForm 设备批量审批通过表单（P3-1 设备审批）
type PeerBatchApproveForm struct {
	RowIds []uint `json:"row_ids" validate:"required"`
}

// ToPeer
func (f *PeerForm) ToPeer() *model.Peer {
	return &model.Peer{
		RowId:    f.RowId,
		Id:       f.Id,
		Cpu:      f.Cpu,
		Hostname: f.Hostname,
		Memory:   f.Memory,
		Os:       f.Os,
		Username: f.Username,
		Uuid:     f.Uuid,
		Version:  f.Version,
		GroupId:  f.GroupId,
		Alias:    f.Alias,
	}
}

type PeerQuery struct {
	PageQuery
	TimeAgo  int    `json:"time_ago" form:"time_ago"`
	Id       string `json:"id" form:"id"`
	Hostname string `json:"hostname" form:"hostname"`
	Uuids    string `json:"uuids" form:"uuids"`
	Ip       string `json:"ip" form:"ip"`
	Username string `json:"username" form:"username"`
	Alias    string `json:"alias" form:"alias"`
	// Status 审批状态过滤（P3-1 设备审批）：0 待审批 / 1 已通过；不传表示不过滤。
	// 使用指针以区分"未传参"与"显式传 0"。
	Status *int `json:"status" form:"status"`
}

type SimpleDataQuery struct {
	Ids []string `json:"ids" form:"ids"`
}
