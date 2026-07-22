package admin

import (
	"github.com/gin-gonic/gin"
	"github.com/lejianwen/rustdesk-api/v2/global"
	"github.com/lejianwen/rustdesk-api/v2/http/request/admin"
	"github.com/lejianwen/rustdesk-api/v2/http/response"
	"github.com/lejianwen/rustdesk-api/v2/service"
)

type Mfa struct{}

// Enroll creates a pending TOTP secret and one-time backup codes. The secret
// is encrypted at rest and MFA remains disabled until Enable verifies a code.
func (m *Mfa) Enroll(c *gin.Context) {
	u := service.AllService.UserService.CurUser(c)
	if u == nil {
		response.Fail(c, 403, response.TranslateMsg(c, "NeedLogin"))
		return
	}
	if u.MfaEnabled {
		response.Fail(c, 101, response.TranslateMsg(c, "MfaAlreadyEnabled"))
		return
	}
	enrollment, err := service.AllService.MfaService.Enroll(u)
	if err != nil {
		response.Fail(c, 101, response.TranslateMsg(c, err.Error()))
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
	f := &admin.MfaEnrollmentChallengeRequest{}
	if err := c.ShouldBindJSON(f); err != nil {
		response.Fail(c, 101, response.TranslateMsg(c, "ParamsError"))
		return
	}
	if errList := global.Validator.ValidStruct(c, f); len(errList) > 0 {
		response.Fail(c, 101, errList[0])
		return
	}
	_, enrollment, err := service.AllService.MfaService.BeginEnrollmentChallenge(f.Challenge)
	if err != nil {
		response.Fail(c, 101, response.TranslateMsg(c, err.Error()))
		return
	}
	response.Success(c, gin.H{"challenge": f.Challenge, "enrollment": enrollment})
}

func (m *Mfa) BootstrapComplete(c *gin.Context) {
	f := &admin.MfaEnrollmentCompleteRequest{}
	if err := c.ShouldBindJSON(f); err != nil {
		response.Fail(c, 101, response.TranslateMsg(c, "ParamsError"))
		return
	}
	if errList := global.Validator.ValidStruct(c, f); len(errList) > 0 {
		response.Fail(c, 101, errList[0])
		return
	}
	u, ut, err := service.AllService.MfaService.CompleteEnrollmentChallenge(f.Challenge, f.Code, c.ClientIP())
	if err != nil {
		response.Fail(c, 101, response.TranslateMsg(c, err.Error()))
		return
	}
	global.LoginLimiter.RemoveAttempts(c.ClientIP())
	responseLoginSuccess(c, u, ut.Token)
}

func (m *Mfa) complete(c *gin.Context, enable bool) {
	f := &admin.MfaCodeForm{}
	if err := c.ShouldBindJSON(f); err != nil {
		response.Fail(c, 101, response.TranslateMsg(c, "ParamsError")+err.Error())
		return
	}
	if errList := global.Validator.ValidStruct(c, f); len(errList) > 0 {
		response.Fail(c, 101, errList[0])
		return
	}
	u := service.AllService.UserService.CurUser(c)
	if u == nil {
		response.Fail(c, 403, response.TranslateMsg(c, "NeedLogin"))
		return
	}
	var err error
	if enable {
		err = service.AllService.MfaService.Enable(u, f.Code)
	} else {
		err = service.AllService.MfaService.Disable(u, f.Code, f.Password)
	}
	if err != nil {
		response.Fail(c, 101, response.TranslateMsg(c, err.Error()))
		return
	}
	response.Success(c, gin.H{"mfa_enabled": u.MfaEnabled})
}
