package admin

import (
	"errors"
	"net"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/lejianwen/rustdesk-api/v2/global"
	"github.com/lejianwen/rustdesk-api/v2/http/request/admin"
	"github.com/lejianwen/rustdesk-api/v2/http/response"
	"github.com/lejianwen/rustdesk-api/v2/model"
	"github.com/lejianwen/rustdesk-api/v2/service"
	"gorm.io/gorm"
)

type RelayNode struct {
}

// List 列表
// @Tags 中继节点
// @Summary 中继节点列表
// @Description 中继节点列表（name 模糊匹配 + 分页，按 priority 降序、row_id 升序）
// @Accept  json
// @Produce  json
// @Param page query int false "页码"
// @Param page_size query int false "页大小"
// @Param name query string false "名称（模糊）"
// @Success 200 {object} response.Response{data=model.RelayNodeList}
// @Failure 500 {object} response.Response
// @Router /admin/relay-nodes/list [get]
// @Security token
func (ct *RelayNode) List(c *gin.Context) {
	query := &admin.RelayNodeQuery{}
	if err := c.ShouldBindQuery(query); err != nil {
		response.Fail(c, 101, response.TranslateMsg(c, "ParamsError")+err.Error())
		return
	}
	res := service.AllService.RelayNodeService.List(query.Page, query.PageSize, func(tx *gorm.DB) {
		if query.Name != "" {
			tx.Where("name like ?", "%"+query.Name+"%")
		}
	})
	response.Success(c, res)
}

// Create 创建
// @Tags 中继节点
// @Summary 创建中继节点
// @Description 创建中继节点（port 不传默认 21117，enabled 不传默认启用）
// @Accept  json
// @Produce  json
// @Param body body admin.RelayNodeForm true "节点信息"
// @Success 200 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /admin/relay-nodes/create [post]
// @Security token
func (ct *RelayNode) Create(c *gin.Context) {
	f := &admin.RelayNodeForm{}
	if err := c.ShouldBindJSON(f); err != nil {
		response.Fail(c, 101, response.TranslateMsg(c, "ParamsError")+err.Error())
		return
	}
	errList := global.Validator.ValidStruct(c, f)
	if len(errList) > 0 {
		response.Fail(c, 101, errList[0])
		return
	}
	n := f.ToRelayNode()
	err := service.AllService.RelayNodeService.Create(n)
	if err != nil {
		response.Fail(c, 101, response.TranslateMsg(c, "OperationFailed")+err.Error())
		return
	}
	response.Success(c, &gin.H{"row_id": n.RowId})
}

// Update 编辑
// @Tags 中继节点
// @Summary 编辑中继节点
// @Description 编辑中继节点（全字段更新，status/last_latency_ms/last_checked_at 由探测回写不受表单影响）
// @Accept  json
// @Produce  json
// @Param body body admin.RelayNodeForm true "节点信息"
// @Success 200 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /admin/relay-nodes/update [post]
// @Security token
func (ct *RelayNode) Update(c *gin.Context) {
	f := &admin.RelayNodeForm{}
	if err := c.ShouldBindJSON(f); err != nil {
		response.Fail(c, 101, response.TranslateMsg(c, "ParamsError")+err.Error())
		return
	}
	if f.RowId == 0 {
		response.Fail(c, 101, response.TranslateMsg(c, "ParamsError"))
		return
	}
	errList := global.Validator.ValidStruct(c, f)
	if len(errList) > 0 {
		response.Fail(c, 101, errList[0])
		return
	}
	ex := service.AllService.RelayNodeService.InfoById(f.RowId)
	if ex.RowId == 0 {
		response.Fail(c, 101, response.TranslateMsg(c, "ItemNotFound"))
		return
	}
	n := f.ToRelayNode()
	// 探测结果字段不由表单维护，保留库内值（Select("*") 全字段更新会覆盖）
	n.Status = ex.Status
	n.LastLatencyMs = ex.LastLatencyMs
	n.LastCheckedAt = ex.LastCheckedAt
	err := service.AllService.RelayNodeService.Update(n)
	if err != nil {
		response.Fail(c, 101, response.TranslateMsg(c, "OperationFailed")+err.Error())
		return
	}
	response.Success(c, nil)
}

// Delete 删除
// @Tags 中继节点
// @Summary 删除中继节点（批量）
// @Description 按 row_id 列表批量删除中继节点
// @Accept  json
// @Produce  json
// @Param body body admin.RelayNodeBatchDeleteForm true "row_id 列表"
// @Success 200 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /admin/relay-nodes/delete [post]
// @Security token
func (ct *RelayNode) Delete(c *gin.Context) {
	f := &admin.RelayNodeBatchDeleteForm{}
	if err := c.ShouldBindJSON(f); err != nil {
		response.Fail(c, 101, response.TranslateMsg(c, "ParamsError")+err.Error())
		return
	}
	if len(f.RowIds) == 0 {
		response.Fail(c, 101, response.TranslateMsg(c, "ParamsError"))
		return
	}
	err := service.AllService.RelayNodeService.BatchDelete(f.RowIds)
	if err != nil {
		response.Fail(c, 101, response.TranslateMsg(c, "OperationFailed")+err.Error())
		return
	}
	response.Success(c, nil)
}

// Test 连通性测试
// @Tags 中继节点
// @Summary 中继节点连通性测试
// @Description 两种入参：{"row_id":N} 用库内配置测试并回写 status/last_latency_ms/last_checked_at；
// @Description {"host":"...","port":...} 直接测试（新增前预检，不写库）。
// @Description 测试方式：TCP connect 测 RTT（timeout 3s）；端口连通后尽力读取 hbbr 特征
// @Description （hbbr 私有协议无标准 banner，通常读不到，ok 以 TCP 连通为准）。
// @Accept  json
// @Produce  json
// @Param body body admin.RelayNodeTestForm true "row_id 或 host+port"
// @Success 200 {object} response.Response{data=gin.H}
// @Failure 500 {object} response.Response
// @Router /admin/relay-nodes/test [post]
// @Security token
func (ct *RelayNode) Test(c *gin.Context) {
	f := &admin.RelayNodeTestForm{}
	if err := c.ShouldBindJSON(f); err != nil {
		response.Fail(c, 101, response.TranslateMsg(c, "ParamsError")+err.Error())
		return
	}
	host := strings.TrimSpace(f.Host)
	port := f.Port
	if f.RowId > 0 {
		// 用库内节点配置测试
		n := service.AllService.RelayNodeService.InfoById(f.RowId)
		if n.RowId == 0 {
			response.Fail(c, 101, response.TranslateMsg(c, "ItemNotFound"))
			return
		}
		host = n.Host
		port = n.Port
	} else if host == "" {
		// 既未给 row_id 也未给 host
		response.Fail(c, 101, response.TranslateMsg(c, "ParamsError"))
		return
	}
	if port <= 0 {
		port = model.RelayNodeDefaultPort
	}
	latency, err := service.AllService.RelayNodeService.TestConnection(host, port)
	ok := err == nil
	// 不回显原始错误串。该接口接受任意 host:port，原样返回 net 包的错误文本
	// （connection refused / no route to host / i/o timeout 三者可区分）会把它
	// 变成一个精确的内网探测预言机——管理员账号一旦被盗即可用来摸清内网拓扑。
	// 归一成粗粒度分类：功能上足够运维判断，信息量不足以做端口扫描。
	errMsg := classifyDialError(err)
	// 库内节点测试：回写探测结果（在线/离线、延迟、探测时间）
	if f.RowId > 0 {
		status := model.RelayNodeStatusOnline
		if !ok {
			status = model.RelayNodeStatusOffline
		}
		if rerr := service.AllService.RelayNodeService.UpdateCheckResult(f.RowId, status, latency); rerr != nil {
			// 回写失败不影响测试结果返回，仅记录
			service.Logger.Warn("relay node UpdateCheckResult failed: ", rerr)
		}
	}
	response.Success(c, &gin.H{
		"ok":         ok,
		"latency_ms": latency,
		"error":      errMsg,
	})
}

// classifyDialError 把探测错误归一为粗粒度分类，避免把精确的网络错误
// 回显给调用方（见 Test 中的说明）。
func classifyDialError(err error) string {
	if err == nil {
		return ""
	}
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return "timeout"
	}
	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		return "unresolved"
	}
	return "unreachable"
}
