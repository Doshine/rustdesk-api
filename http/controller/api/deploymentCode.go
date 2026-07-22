package api

import (
	"github.com/gin-gonic/gin"
	"github.com/lejianwen/rustdesk-api/v2/global"
	request "github.com/lejianwen/rustdesk-api/v2/http/request/api"
	"github.com/lejianwen/rustdesk-api/v2/http/response"
	"github.com/lejianwen/rustdesk-api/v2/service"
)

type DeploymentCode struct{}

// Claim exchanges a high-entropy deployment code for a pending device record.
// All domain failures intentionally share one public response to avoid exposing
// code validity, expiry, or remaining capacity to unauthenticated callers.
func (ct *DeploymentCode) Claim(c *gin.Context) {
	form := &request.DeploymentClaimForm{}
	if err := c.ShouldBindJSON(form); err != nil {
		response.Fail(c, 101, "Deployment code unavailable")
		return
	}
	if errs := global.Validator.ValidStruct(c, form); len(errs) > 0 {
		response.Fail(c, 101, "Deployment code unavailable")
		return
	}
	peer, err := service.AllService.DeploymentCodeService.Claim(service.DeploymentClaimInput{
		Code: form.Code, DeviceId: form.DeviceId, Uuid: form.Uuid, Hostname: form.Hostname,
		Os: form.Os, Username: form.Username, Version: form.Version, Platform: form.Platform,
		Ip: c.ClientIP(),
	})
	if err != nil {
		response.Fail(c, 101, "Deployment code unavailable")
		return
	}
	response.Success(c, gin.H{
		"device_id": peer.Id, "status": "pending_approval", "approval_status": peer.Status,
	})
}
