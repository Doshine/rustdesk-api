package api

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	request "github.com/lejianwen/rustdesk-api/v2/http/request/api"
	"github.com/lejianwen/rustdesk-api/v2/http/response"
	"github.com/lejianwen/rustdesk-api/v2/model"
	"github.com/lejianwen/rustdesk-api/v2/service"
	"time"
)

type Audit struct {
}

// AuditConn
// @Tags 审计
// @Summary 审计连接
// @Description 审计连接
// @Accept  json
// @Produce  json
// @Param body body request.AuditConnForm true "审计连接"
// @Success 200 {string} string ""
// @Failure 500 {object} response.Response
// @Router /audit/conn [post]
func (a *Audit) AuditConn(c *gin.Context) {
	af := &request.AuditConnForm{}
	err := c.ShouldBindBodyWith(af, binding.JSON)
	if err != nil {
		response.Error(c, response.TranslateMsg(c, "ParamsError")+err.Error())
		return
	}
	/*ttt := &gin.H{}
	c.ShouldBindBodyWith(ttt, binding.JSON)
	fmt.Println(ttt)*/
	if !verifyReportingDevice(af.Id, af.Uuid) {
		// 不回显具体原因，避免把「哪些 uuid 存在」变成可探测的信号
		response.Error(c, response.TranslateMsg(c, "ParamsError"))
		return
	}
	ac := af.ToAuditConn()
	// IP 由服务端从连接上下文派生，不接受请求体自报。
	// 原实现直接采信 af.Ip，任何人都能把审计记录里的来源写成任意地址，
	// 使审计表在事后取证时完全不可用。
	ac.Ip = c.ClientIP()
	if af.Action == model.AuditActionNew {
		if err := service.AllService.AuditService.CreateAuditConn(ac); err != nil {
			response.Error(c, err.Error())
			return
		}
	} else if af.Action == model.AuditActionClose {
		// Closing a session is a new immutable event. Never update the original
		// connection record because audit history must remain append-only.
		ac.CloseTime = time.Now().Unix()
		if err := service.AllService.AuditService.CreateAuditConn(ac); err != nil {
			response.Error(c, err.Error())
			return
		}
	} else if af.Action == "" {
		// Legacy clients send an empty action for metadata updates. Preserve the
		// event stream by recording an explicit update event instead of mutating it.
		ac.Action = model.AuditActionUpdate
		if err := service.AllService.AuditService.CreateAuditConn(ac); err != nil {
			response.Error(c, err.Error())
			return
		}
	}
	response.Success(c, "")
}

// AuditFile
// @Tags 审计
// @Summary 审计文件
// @Description 审计文件
// @Accept  json
// @Produce  json
// @Param body body request.AuditFileForm true "审计文件"
// @Success 200 {string} string ""
// @Failure 500 {object} response.Response
// @Router /audit/file [post]
func (a *Audit) AuditFile(c *gin.Context) {
	aff := &request.AuditFileForm{}
	err := c.ShouldBindBodyWith(aff, binding.JSON)
	if err != nil {
		response.Error(c, response.TranslateMsg(c, "ParamsError")+err.Error())
		return
	}
	//ttt := &gin.H{}
	//c.ShouldBindBodyWith(ttt, binding.JSON)
	//fmt.Println(ttt)
	if !verifyReportingDevice(aff.Id, aff.Uuid) {
		response.Error(c, response.TranslateMsg(c, "ParamsError"))
		return
	}
	af := aff.ToAuditFile()
	if err := service.AllService.AuditService.CreateAuditFile(af); err != nil {
		response.Error(c, err.Error())
		return
	}
	response.Success(c, "")
}

// verifyReportingDevice 校验上报方确实是一台已注册设备。
//
// 这两个接口必须保持匿名（被控端上报不带 Authorization），所以身份只能靠
// 设备自身的注册信息来锚定：uuid 必须能在 peers 表中找到，且其 id 必须与
// 上报的 id 一致。这挡住的是「任意第三方灌入伪造审计记录」——攻击者需要先
// 拿到一台真实设备的 uuid 与 id 才能写入，而不是随手 POST 就能污染证据链。
//
// 注意这不是强身份认证：uuid 本身不是秘密。要做到不可伪造，需要设备证书或
// 由服务端签发的上报票据，属独立工作项（见审计报告 P0-2 的整改建议）。
func verifyReportingDevice(deviceId, uuid string) bool {
	uuid = strings.TrimSpace(uuid)
	deviceId = strings.TrimSpace(deviceId)
	if uuid == "" || deviceId == "" {
		return false
	}
	p := service.AllService.PeerService.FindByUuid(uuid)
	if p == nil || p.RowId == 0 {
		return false
	}
	return p.Id == deviceId
}
