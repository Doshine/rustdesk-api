package api

import (
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
	ac := af.ToAuditConn()
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
	af := aff.ToAuditFile()
	if err := service.AllService.AuditService.CreateAuditFile(af); err != nil {
		response.Error(c, err.Error())
		return
	}
	response.Success(c, "")
}
