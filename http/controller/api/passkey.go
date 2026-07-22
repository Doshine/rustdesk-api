package api

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/lejianwen/rustdesk-api/v2/global"
	apiRequest "github.com/lejianwen/rustdesk-api/v2/http/request/api"
	"github.com/lejianwen/rustdesk-api/v2/http/response"
	apiResp "github.com/lejianwen/rustdesk-api/v2/http/response/api"
	"github.com/lejianwen/rustdesk-api/v2/model"
	"github.com/lejianwen/rustdesk-api/v2/service"
)

type Passkey struct{}

func (p *Passkey) RegisterBegin(c *gin.Context) {
	u := service.AllService.UserService.CurUser(c)
	if u == nil {
		response.Error(c, response.TranslateMsg(c, "NeedLogin"))
		return
	}
	result, err := service.AllService.PasskeyService.BeginRegistration(u, c.GetString("token"))
	if err != nil {
		response.Error(c, response.TranslateMsg(c, err.Error()))
		return
	}
	response.Success(c, result)
}

func (p *Passkey) RegisterComplete(c *gin.Context) {
	f := &apiRequest.PasskeyRegistrationCompleteRequest{}
	if err := c.ShouldBindJSON(f); err != nil {
		response.Error(c, response.TranslateMsg(c, "ParamsError"))
		return
	}
	if errList := global.Validator.ValidStruct(c, f); len(errList) > 0 {
		response.Error(c, errList[0])
		return
	}
	credential, err := service.AllService.PasskeyService.CompleteRegistration(f.Challenge, f.Credential, f.Name, c.GetString("token"))
	if err != nil {
		response.Error(c, response.TranslateMsg(c, err.Error()))
		return
	}
	response.Success(c, credential)
}

func (p *Passkey) LoginBegin(c *gin.Context) {
	result, err := service.AllService.PasskeyService.BeginLogin()
	if err != nil {
		response.Error(c, response.TranslateMsg(c, err.Error()))
		return
	}
	response.Success(c, result)
}

func (p *Passkey) LoginComplete(c *gin.Context) {
	f := &apiRequest.PasskeyLoginCompleteRequest{}
	if err := c.ShouldBindJSON(f); err != nil {
		response.Error(c, response.TranslateMsg(c, "ParamsError"))
		return
	}
	if errList := global.Validator.ValidStruct(c, f); len(errList) > 0 {
		response.Error(c, errList[0])
		return
	}
	loginResult, err := service.AllService.PasskeyService.CompleteLogin(f.Challenge, f.Credential)
	if err != nil || loginResult == nil || loginResult.User == nil || loginResult.User.Id == 0 {
		if err == nil {
			err = service.ErrPasskeyInvalidCredential
		}
		response.Error(c, response.TranslateMsg(c, err.Error()))
		return
	}
	u := loginResult.User
	client := f.DeviceInfo.Type
	if client == "" {
		client = model.LoginLogClientWeb
	}
	if c.GetHeader("referer") != "" {
		client = model.LoginLogClientWeb
	}
	source := &service.OauthCacheItem{Id: f.Id, Uuid: f.Uuid, DeviceOs: f.DeviceInfo.Os, DeviceType: client, LoginType: model.LoginLogTypePasskey, PasskeyCredentialId: loginResult.CredentialId}
	if service.AllService.MfaService.RequiresMfaForLogin(u) {
		if !u.MfaEnabled {
			challenge, challengeErr := service.AllService.MfaService.CreateEnrollmentChallenge(u, &service.MfaEnrollmentChallengeItem{
				Id: f.Id, Uuid: f.Uuid, DeviceOs: f.DeviceInfo.Os, DeviceType: client, LoginType: model.LoginLogTypePasskey, PasskeyCredentialId: loginResult.CredentialId,
			})
			if challengeErr != nil {
				response.Error(c, response.TranslateMsg(c, challengeErr.Error()))
				return
			}
			c.JSON(http.StatusOK, gin.H{"mfa_enrollment_required": true, "mfa_enrollment_challenge": challenge})
			return
		}
		challenge, challengeErr := service.AllService.MfaService.CreateOauthChallenge(u, source)
		if challengeErr != nil {
			response.Error(c, response.TranslateMsg(c, challengeErr.Error()))
			return
		}
		c.JSON(http.StatusOK, gin.H{"mfa_required": true, "mfa_challenge": challenge})
		return
	}
	ut, loginErr := service.AllService.UserService.LoginWithPasskey(u, &model.LoginLog{
		UserId: u.Id, Client: client, DeviceId: f.Id, Uuid: f.Uuid,
		Ip: c.ClientIP(), Type: model.LoginLogTypePasskey, Platform: f.DeviceInfo.Os,
		PasskeyCredentialId: loginResult.CredentialId,
	})
	if loginErr != nil || ut == nil {
		if loginErr == nil {
			loginErr = service.ErrPasskeyCredentialRevoked
		}
		response.Error(c, response.TranslateMsg(c, loginErr.Error()))
		return
	}
	global.LoginLimiter.RemoveAttempts(c.ClientIP())
	c.JSON(http.StatusOK, apiResp.LoginRes{AccessToken: ut.Token, Type: "access_token", User: *(&apiResp.UserPayload{}).FromUser(u)})
}

func (p *Passkey) List(c *gin.Context) {
	u := service.AllService.UserService.CurUser(c)
	if u == nil {
		response.Error(c, response.TranslateMsg(c, "NeedLogin"))
		return
	}
	result, err := service.AllService.PasskeyService.List(u, false)
	if err != nil {
		response.Error(c, response.TranslateMsg(c, err.Error()))
		return
	}
	response.Success(c, result)
}

func (p *Passkey) Revoke(c *gin.Context) {
	u := service.AllService.UserService.CurUser(c)
	if u == nil {
		response.Error(c, response.TranslateMsg(c, "NeedLogin"))
		return
	}
	f := &apiRequest.PasskeyRevokeRequest{}
	if c.Request.ContentLength != 0 {
		if err := c.ShouldBindJSON(f); err != nil {
			response.Error(c, response.TranslateMsg(c, "ParamsError"))
			return
		}
		if errList := global.Validator.ValidStruct(c, f); len(errList) > 0 {
			response.Error(c, errList[0])
			return
		}
	}
	var id uint
	if _, err := fmt.Sscan(c.Param("id"), &id); err != nil || id == 0 {
		response.Error(c, response.TranslateMsg(c, "ParamsError"))
		return
	}
	sessionsRevoked, err := service.AllService.PasskeyService.Revoke(u, id, f.Reason)
	if err != nil {
		response.Error(c, response.TranslateMsg(c, err.Error()))
		return
	}
	response.Success(c, gin.H{"revoked": true, "id": id, "sessions_revoked": sessionsRevoked})
}
