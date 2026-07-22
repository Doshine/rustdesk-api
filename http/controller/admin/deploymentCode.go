package admin

import (
	"github.com/gin-gonic/gin"
	"github.com/lejianwen/rustdesk-api/v2/global"
	request "github.com/lejianwen/rustdesk-api/v2/http/request/admin"
	"github.com/lejianwen/rustdesk-api/v2/http/response"
	"github.com/lejianwen/rustdesk-api/v2/service"
)

type DeploymentCode struct{}

func (ct *DeploymentCode) Create(c *gin.Context) {
	form := &request.DeploymentCodeCreateForm{}
	if err := c.ShouldBindJSON(form); err != nil {
		response.Fail(c, 101, response.TranslateMsg(c, "ParamsError"))
		return
	}
	if errs := global.Validator.ValidStruct(c, form); len(errs) > 0 {
		response.Fail(c, 101, errs[0])
		return
	}
	user := service.AllService.UserService.CurUser(c)
	result, err := service.AllService.DeploymentCodeService.Create(service.DeploymentCodeCreateInput{
		Name: form.Name, DeviceGroupId: form.DeviceGroupId, AllowedPlatforms: form.AllowedPlatforms,
		ExpiresAt: form.ExpiresAt, MaxUses: form.MaxUses,
	}, user.Id, c.ClientIP())
	if err != nil {
		response.Fail(c, 101, response.TranslateMsg(c, "OperationFailed")+err.Error())
		return
	}
	response.Success(c, result)
}

func (ct *DeploymentCode) List(c *gin.Context) {
	query := &request.DeploymentCodeQuery{}
	if err := c.ShouldBindQuery(query); err != nil {
		response.Fail(c, 101, response.TranslateMsg(c, "ParamsError"))
		return
	}
	result, err := service.AllService.DeploymentCodeService.List(query.Page, query.PageSize, query.Status)
	if err != nil {
		response.Fail(c, 101, response.TranslateMsg(c, "OperationFailed"))
		return
	}
	response.Success(c, result)
}

func (ct *DeploymentCode) Revoke(c *gin.Context) {
	ct.mutate(c, false)
}

func (ct *DeploymentCode) Rotate(c *gin.Context) {
	ct.mutate(c, true)
}

func (ct *DeploymentCode) mutate(c *gin.Context, rotate bool) {
	form := &request.DeploymentCodeIdForm{}
	if err := c.ShouldBindJSON(form); err != nil {
		response.Fail(c, 101, response.TranslateMsg(c, "ParamsError"))
		return
	}
	if errs := global.Validator.ValidStruct(c, form); len(errs) > 0 {
		response.Fail(c, 101, errs[0])
		return
	}
	user := service.AllService.UserService.CurUser(c)
	if rotate {
		result, err := service.AllService.DeploymentCodeService.Rotate(form.Id, user.Id, c.ClientIP())
		if err != nil {
			response.Fail(c, 101, response.TranslateMsg(c, "OperationFailed")+err.Error())
			return
		}
		response.Success(c, result)
		return
	}
	if err := service.AllService.DeploymentCodeService.Revoke(form.Id, user.Id, c.ClientIP()); err != nil {
		response.Fail(c, 101, response.TranslateMsg(c, "OperationFailed")+err.Error())
		return
	}
	response.Success(c, nil)
}

func (ct *DeploymentCode) Audit(c *gin.Context) {
	query := &request.DeploymentAuditQuery{}
	if err := c.ShouldBindQuery(query); err != nil {
		response.Fail(c, 101, response.TranslateMsg(c, "ParamsError"))
		return
	}
	if errs := global.Validator.ValidStruct(c, query); len(errs) > 0 {
		response.Fail(c, 101, errs[0])
		return
	}
	result, err := service.AllService.DeploymentCodeService.AuditList(query.DeploymentCodeId, query.Page, query.PageSize)
	if err != nil {
		response.Fail(c, 101, response.TranslateMsg(c, "OperationFailed"))
		return
	}
	response.Success(c, result)
}
