package admin

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/lejianwen/rustdesk-api/v2/global"
	adminRequest "github.com/lejianwen/rustdesk-api/v2/http/request/admin"
	"github.com/lejianwen/rustdesk-api/v2/http/response"
	"github.com/lejianwen/rustdesk-api/v2/model"
	"github.com/lejianwen/rustdesk-api/v2/service"
)

type Passkey struct{}

func (p *Passkey) RegisterBegin(c *gin.Context) {
	u := service.AllService.UserService.CurUser(c)
	if u == nil {
		response.Fail(c, 401, response.TranslateMsg(c, "NeedLogin"))
		return
	}
	result, err := service.AllService.PasskeyService.BeginRegistration(u, c.GetString("token"))
	if err != nil {
		response.Fail(c, 400, response.TranslateMsg(c, err.Error()))
		return
	}
	response.Success(c, result)
}

func (p *Passkey) RegisterComplete(c *gin.Context) {
	f := &adminRequest.PasskeyRegistrationCompleteRequest{}
	if err := c.ShouldBindJSON(f); err != nil {
		response.Fail(c, 400, response.TranslateMsg(c, "ParamsError"))
		return
	}
	if errList := global.Validator.ValidStruct(c, f); len(errList) > 0 {
		response.Fail(c, 400, errList[0])
		return
	}
	credential, err := service.AllService.PasskeyService.CompleteRegistration(f.Challenge, f.Credential, f.Name, c.GetString("token"))
	if err != nil {
		response.Fail(c, 400, response.TranslateMsg(c, err.Error()))
		return
	}
	response.Success(c, credential)
}

func (p *Passkey) LoginBegin(c *gin.Context) {
	result, err := service.AllService.PasskeyService.BeginLogin()
	if err != nil {
		response.Fail(c, 400, response.TranslateMsg(c, err.Error()))
		return
	}
	response.Success(c, result)
}

func (p *Passkey) LoginComplete(c *gin.Context) {
	f := &adminRequest.PasskeyLoginCompleteRequest{}
	if err := c.ShouldBindJSON(f); err != nil {
		response.Fail(c, 400, response.TranslateMsg(c, "ParamsError"))
		return
	}
	if errList := global.Validator.ValidStruct(c, f); len(errList) > 0 {
		response.Fail(c, 400, errList[0])
		return
	}
	loginResult, err := service.AllService.PasskeyService.CompleteLogin(f.Challenge, f.Credential)
	if err != nil || loginResult == nil || loginResult.User == nil || loginResult.User.Id == 0 {
		if err == nil {
			err = service.ErrPasskeyInvalidCredential
		}
		response.Fail(c, 401, response.TranslateMsg(c, err.Error()))
		return
	}
	u := loginResult.User
	source := &service.OauthCacheItem{DeviceOs: f.Platform, DeviceType: model.LoginLogClientWebAdmin, LoginType: model.LoginLogTypePasskey, PasskeyCredentialId: loginResult.CredentialId}
	if service.AllService.MfaService.RequiresMfaForLogin(u) {
		if !u.MfaEnabled {
			challenge, challengeErr := service.AllService.MfaService.CreateEnrollmentChallenge(u, &service.MfaEnrollmentChallengeItem{
				DeviceOs: f.Platform, DeviceType: model.LoginLogClientWebAdmin, LoginType: model.LoginLogTypePasskey, PasskeyCredentialId: loginResult.CredentialId,
			})
			if challengeErr != nil {
				response.Fail(c, 400, response.TranslateMsg(c, challengeErr.Error()))
				return
			}
			response.Success(c, gin.H{"mfa_enrollment_required": true, "mfa_enrollment_challenge": challenge})
			return
		}
		challenge, challengeErr := service.AllService.MfaService.CreateOauthChallenge(u, source)
		if challengeErr != nil {
			response.Fail(c, 400, response.TranslateMsg(c, challengeErr.Error()))
			return
		}
		response.Success(c, gin.H{"mfa_required": true, "mfa_challenge": challenge})
		return
	}
	ut, loginErr := service.AllService.UserService.LoginWithPasskey(u, &model.LoginLog{
		UserId: u.Id, Client: model.LoginLogClientWebAdmin, Ip: c.ClientIP(),
		Type: model.LoginLogTypePasskey, Platform: f.Platform, PasskeyCredentialId: loginResult.CredentialId,
	})
	if loginErr != nil || ut == nil {
		if loginErr == nil {
			loginErr = service.ErrPasskeyCredentialRevoked
		}
		response.Fail(c, 401, response.TranslateMsg(c, loginErr.Error()))
		return
	}
	global.LoginLimiter.RemoveAttempts(c.ClientIP())
	responseLoginSuccess(c, u, ut.Token)
}

func (p *Passkey) List(c *gin.Context) {
	u := service.AllService.UserService.CurUser(c)
	if u == nil {
		response.Fail(c, 401, response.TranslateMsg(c, "NeedLogin"))
		return
	}
	result, err := service.AllService.PasskeyService.List(u, false)
	if err != nil {
		response.Fail(c, 400, response.TranslateMsg(c, err.Error()))
		return
	}
	response.Success(c, result)
}

func (p *Passkey) Revoke(c *gin.Context) {
	u := service.AllService.UserService.CurUser(c)
	if u == nil {
		response.Fail(c, 401, response.TranslateMsg(c, "NeedLogin"))
		return
	}
	id, err := parsePasskeyID(c.Param("id"))
	if err != nil {
		response.Fail(c, 400, response.TranslateMsg(c, "ParamsError"))
		return
	}
	f := &adminRequest.PasskeyRevokeRequest{}
	if c.Request.ContentLength != 0 && c.ShouldBindJSON(f) != nil {
		response.Fail(c, 400, response.TranslateMsg(c, "ParamsError"))
		return
	}
	sessionsRevoked, err := service.AllService.PasskeyService.Revoke(u, id, f.Reason)
	if err != nil {
		response.Fail(c, 400, response.TranslateMsg(c, err.Error()))
		return
	}
	response.Success(c, gin.H{"revoked": true, "id": id, "sessions_revoked": sessionsRevoked})
}

func (p *Passkey) ListUser(c *gin.Context) {
	userID, err := parsePasskeyID(c.Param("userId"))
	if err != nil {
		response.Fail(c, 400, response.TranslateMsg(c, "ParamsError"))
		return
	}
	u := service.AllService.UserService.InfoById(userID)
	if u == nil || u.Id == 0 {
		response.Fail(c, 404, response.TranslateMsg(c, "UserNotFound"))
		return
	}
	result, err := service.AllService.PasskeyService.List(u, true)
	if err != nil {
		response.Fail(c, 400, response.TranslateMsg(c, err.Error()))
		return
	}
	response.Success(c, result)
}

func (p *Passkey) RevokeUser(c *gin.Context) {
	actor := service.AllService.UserService.CurUser(c)
	userID, err := parsePasskeyID(c.Param("userId"))
	if err != nil {
		response.Fail(c, 400, response.TranslateMsg(c, "ParamsError"))
		return
	}
	credentialID, err := parsePasskeyID(c.Param("id"))
	if err != nil {
		response.Fail(c, 400, response.TranslateMsg(c, "ParamsError"))
		return
	}
	f := &adminRequest.PasskeyRevokeRequest{}
	if c.Request.ContentLength != 0 && c.ShouldBindJSON(f) != nil {
		response.Fail(c, 400, response.TranslateMsg(c, "ParamsError"))
		return
	}
	sessionsRevoked, err := service.AllService.PasskeyService.RevokeForAdmin(actor, userID, credentialID, f.Reason)
	if err != nil {
		response.Fail(c, 403, response.TranslateMsg(c, err.Error()))
		return
	}
	response.Success(c, gin.H{"revoked": true, "id": credentialID, "user_id": userID, "sessions_revoked": sessionsRevoked})
}

func parsePasskeyID(raw string) (uint, error) {
	var id uint
	if _, err := fmt.Sscan(raw, &id); err != nil || id == 0 {
		return 0, fmt.Errorf("invalid passkey id")
	}
	return id, nil
}
