package api

import (
	"github.com/gin-gonic/gin"
	"github.com/lejianwen/rustdesk-api/v2/global"
	apiRequest "github.com/lejianwen/rustdesk-api/v2/http/request/api"
	"github.com/lejianwen/rustdesk-api/v2/http/response"
	apiResp "github.com/lejianwen/rustdesk-api/v2/http/response/api"
	"github.com/lejianwen/rustdesk-api/v2/service"
)

type Mfa struct{}

// Enroll creates a pending TOTP secret for the authenticated user. MFA remains
// disabled until the user confirms the first authenticator code via Enable.
func (m *Mfa) Enroll(c *gin.Context) {
	u := service.AllService.UserService.CurUser(c)
	if u == nil {
		response.Error(c, response.TranslateMsg(c, "NeedLogin"))
		return
	}
	enrollment, err := service.AllService.MfaService.Enroll(u)
	if err != nil {
		response.Error(c, response.TranslateMsg(c, err.Error()))
		return
	}
	response.Success(c, enrollment)
}

func (m *Mfa) Enable(c *gin.Context) {
	m.complete(c, true)
}

func (m *Mfa) Disable(c *gin.Context) {
	m.complete(c, false)
}

func (m *Mfa) BootstrapBegin(c *gin.Context) {
	f := &apiRequest.MfaEnrollmentChallengeRequest{}
	if err := c.ShouldBindJSON(f); err != nil {
		response.Error(c, response.TranslateMsg(c, "ParamsError"))
		return
	}
	if errList := global.Validator.ValidStruct(c, f); len(errList) > 0 {
		response.Error(c, errList[0])
		return
	}
	_, enrollment, err := service.AllService.MfaService.BeginEnrollmentChallenge(f.Challenge)
	if err != nil {
		response.Error(c, response.TranslateMsg(c, err.Error()))
		return
	}
	response.Success(c, gin.H{"challenge": f.Challenge, "enrollment": enrollment})
}

func (m *Mfa) BootstrapComplete(c *gin.Context) {
	f := &apiRequest.MfaEnrollmentCompleteRequest{}
	if err := c.ShouldBindJSON(f); err != nil {
		response.Error(c, response.TranslateMsg(c, "ParamsError"))
		return
	}
	if errList := global.Validator.ValidStruct(c, f); len(errList) > 0 {
		response.Error(c, errList[0])
		return
	}
	u, ut, err := service.AllService.MfaService.CompleteEnrollmentChallenge(f.Challenge, f.Code, c.ClientIP())
	if err != nil {
		response.Error(c, response.TranslateMsg(c, err.Error()))
		return
	}
	global.LoginLimiter.RemoveAttempts(c.ClientIP())
	c.JSON(200, apiResp.LoginRes{
		AccessToken: ut.Token,
		Type:        "access_token",
		User:        *(&apiResp.UserPayload{}).FromUser(u),
	})
}

func (m *Mfa) complete(c *gin.Context, enable bool) {
	f := &apiRequest.MfaCodeForm{}
	if err := c.ShouldBindJSON(f); err != nil {
		response.Error(c, response.TranslateMsg(c, "ParamsError"))
		return
	}
	if errList := global.Validator.ValidStruct(c, f); len(errList) > 0 {
		response.Error(c, errList[0])
		return
	}
	u := service.AllService.UserService.CurUser(c)
	if u == nil {
		response.Error(c, response.TranslateMsg(c, "NeedLogin"))
		return
	}
	var err error
	if enable {
		err = service.AllService.MfaService.Enable(u, f.Code)
	} else {
		err = service.AllService.MfaService.Disable(u, f.Code, f.Password)
	}
	if err != nil {
		response.Error(c, response.TranslateMsg(c, err.Error()))
		return
	}
	response.Success(c, gin.H{"mfa_enabled": u.MfaEnabled})
}
