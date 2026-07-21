package service

import (
	"net"
	"strconv"
	"time"

	"github.com/lejianwen/rustdesk-api/v2/model"
	"gorm.io/gorm"
)

type RelayNodeService struct {
}

// InfoById 根据row_id取中继节点
func (s *RelayNodeService) InfoById(id uint) *model.RelayNode {
	n := &model.RelayNode{}
	DB.Where("row_id = ?", id).First(n)
	return n
}

// List 分页列表
func (s *RelayNodeService) List(page, pageSize uint, where func(tx *gorm.DB)) (res *model.RelayNodeList) {
	res = &model.RelayNodeList{}
	res.Page = int64(page)
	res.PageSize = int64(pageSize)
	tx := DB.Model(&model.RelayNode{})
	if where != nil {
		where(tx)
	}
	tx.Count(&res.Total)
	tx.Scopes(Paginate(page, pageSize))
	tx.Order("priority desc, row_id asc")
	tx.Find(&res.RelayNodes)
	return
}

// Create 创建
func (s *RelayNodeService) Create(u *model.RelayNode) error {
	return DB.Create(u).Error
}

// Update 更新。Select("*") 全字段更新（参考 AddressBookService.UpdateAll），
// 确保 enabled=false 等零值也能落库；created_at 保持不变。
func (s *RelayNodeService) Update(u *model.RelayNode) error {
	return DB.Model(u).Select("*").Omit("created_at").Updates(u).Error
}

// BatchDelete 按 row_id 列表批量删除
func (s *RelayNodeService) BatchDelete(ids []uint) error {
	return DB.Where("row_id in (?)", ids).Delete(&model.RelayNode{}).Error
}

// UpdateCheckResult 回写探测结果：状态 / 最近延迟 / 最近探测时间
func (s *RelayNodeService) UpdateCheckResult(rowId uint, status int, latencyMs int64) error {
	return DB.Model(&model.RelayNode{}).Where("row_id = ?", rowId).Updates(map[string]interface{}{
		"status":          status,
		"last_latency_ms": latencyMs,
		"last_checked_at": time.Now().Unix(),
	}).Error
}

// TestConnection TCP 连通性探测：连接指定 host:port 并测量 RTT（timeout 3s）。
// 返回延迟毫秒数；err 非空表示不可达。
//
// 关于 hbbr 特征读取：hbbr（RustDesk 中继）使用私有 protobuf 协议，
// 无标准 banner/握手，服务端通常不会在建连后主动下发数据，强行按协议
// 构造请求成本过高。这里仅在 TCP 连通后设短读超时尽力读取一次（读不到
// 属正常），ok 判定以 TCP 连通为准。
func (s *RelayNodeService) TestConnection(host string, port int) (int64, error) {
	addr := net.JoinHostPort(host, strconv.Itoa(port))
	start := time.Now()
	conn, err := net.DialTimeout("tcp", addr, 3*time.Second)
	if err != nil {
		return 0, err
	}
	defer conn.Close()
	latency := time.Since(start).Milliseconds()
	// 尽力读取 hbbr 特征（非必需，见函数注释）：短读超时 1.5s，读不到不影响结果
	_ = conn.SetReadDeadline(time.Now().Add(1500 * time.Millisecond))
	buf := make([]byte, 64)
	_, _ = conn.Read(buf)
	return latency, nil
}
