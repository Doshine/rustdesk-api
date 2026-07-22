package api

import (
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/lejianwen/rustdesk-api/v2/global"
	"github.com/lejianwen/rustdesk-api/v2/http/request/api"
	"github.com/lejianwen/rustdesk-api/v2/http/response"
	apiResp "github.com/lejianwen/rustdesk-api/v2/http/response/api"
	"github.com/lejianwen/rustdesk-api/v2/model"
	"github.com/lejianwen/rustdesk-api/v2/service"
	"net/http"
)

type Login struct {
}

// Login 登录
// @Tags 登录
// @Summary 登录
// @Description 登录
// @Accept  json
// @Produce  json
// @Param body body api.LoginForm true "登录表单"
// @Success 200 {object} apiResp.LoginRes
// @Failure 500 {object} response.ErrorResponse
// @Router /login [post]
func (l *Login) Login(c *gin.Context) {
	if global.Config.App.DisablePwdLogin {
		response.Error(c, response.TranslateMsg(c, "PwdLoginDisabled"))
		return
	}

	// 检查登录限制
	loginLimiter := global.LoginLimiter
	clientIp := c.ClientIP()

	f := &api.LoginForm{}
	err := c.ShouldBindJSON(f)
	//fmt.Println(f)
	if err != nil {
		loginLimiter.RecordFailedAttempt(clientIp)
		global.Logger.Warn(fmt.Sprintf("Login Fail: %s %s %s", "ParamsError", c.RemoteIP(), c.ClientIP()))
		response.Error(c, response.TranslateMsg(c, "ParamsError")+err.Error())
		return
	}

	errList := global.Validator.ValidStruct(c, f)
	if len(errList) > 0 {
		loginLimiter.RecordFailedAttempt(clientIp)
		global.Logger.Warn(fmt.Sprintf("Login Fail: %s %s %s", "ParamsError", c.RemoteIP(), c.ClientIP()))
		response.Error(c, errList[0])
		return
	}

	u := service.AllService.UserService.InfoByUsernamePassword(f.Username, f.Password)

	if u.Id == 0 {
		loginLimiter.RecordFailedAttempt(clientIp)
		global.Logger.Warn(fmt.Sprintf("Login Fail: %s %s %s", "UsernameOrPasswordError", c.RemoteIP(), c.ClientIP()))
		response.Error(c, response.TranslateMsg(c, "UsernameOrPasswordError"))
		return
	}

	if !service.AllService.UserService.CheckUserEnable(u) {
		response.Error(c, response.TranslateMsg(c, "UserDisabled"))
		return
	}
	// 根据 referer 判断是 WebClient 还是 App，登录门禁和最终会话必须使用同一设备类型。
	if c.GetHeader("referer") != "" {
		f.DeviceInfo.Type = model.LoginLogClientWeb
	}
	if err := service.AllService.MfaService.VerifyLogin(u, f.MfaCode); err != nil {
		if err == service.ErrMfaRequired {
			challenge, challengeErr := service.AllService.MfaService.CreateEnrollmentChallenge(u, &service.MfaEnrollmentChallengeItem{
				Id:         f.Id,
				Uuid:       f.Uuid,
				DeviceOs:   f.DeviceInfo.Os,
				DeviceType: f.DeviceInfo.Type,
				LoginType:  model.LoginLogTypeAccount,
			})
			if challengeErr != nil {
				response.Error(c, response.TranslateMsg(c, challengeErr.Error()))
				return
			}
			c.JSON(http.StatusOK, gin.H{"mfa_enrollment_required": true, "mfa_enrollment_challenge": challenge})
			return
		}
		loginLimiter.RecordFailedAttempt(clientIp)
		response.Error(c, response.TranslateMsg(c, err.Error()))
		return
	}

	ut := service.AllService.UserService.Login(u, &model.LoginLog{
		UserId:   u.Id,
		Client:   f.DeviceInfo.Type,
		DeviceId: f.Id,
		Uuid:     f.Uuid,
		Ip:       c.ClientIP(),
		Type:     model.LoginLogTypeAccount,
		Platform: f.DeviceInfo.Os,
	})

	c.JSON(http.StatusOK, apiResp.LoginRes{
		AccessToken: ut.Token,
		Type:        "access_token",
		User:        *(&apiResp.UserPayload{}).FromUser(u),
	})
}

// SmsCode 发送短信验证码
// @Tags 登录
// @Summary 发送短信验证码
// @Description 发送短信验证码
// @Accept  json
// @Produce  json
// @Param body body api.SmsCodeForm true "手机号"
// @Success 200 {object} response.Response
// @Failure 500 {object} response.ErrorResponse
// @Router /sms-code [post]
func (l *Login) SmsCode(c *gin.Context) {
	loginLimiter := global.LoginLimiter
	clientIp := c.ClientIP()
	banned, _ := loginLimiter.CheckSecurityStatus(clientIp)
	if banned {
		response.Error(c, response.TranslateMsg(c, "LoginBanned"))
		return
	}

	f := &api.SmsCodeForm{}
	err := c.ShouldBindJSON(f)
	if err != nil {
		global.Logger.Warn(fmt.Sprintf("SmsCode Fail: %s %s %s", "ParamsError", c.RemoteIP(), clientIp))
		response.Error(c, response.TranslateMsg(c, "ParamsError")+err.Error())
		return
	}

	errList := global.Validator.ValidStruct(c, f)
	if len(errList) > 0 {
		global.Logger.Warn(fmt.Sprintf("SmsCode Fail: %s %s %s", "ParamsError", c.RemoteIP(), clientIp))
		response.Error(c, errList[0])
		return
	}

	err = service.AllService.SmsService.SendLoginCode(f.Phone, clientIp)
	if err != nil {
		global.Logger.Warn(fmt.Sprintf("SmsCode Fail: %s %s %s", err.Error(), c.RemoteIP(), clientIp))
		response.Error(c, response.TranslateMsg(c, err.Error()))
		return
	}
	response.Success(c, nil)
}

// LoginSms 短信验证码登录
// @Tags 登录
// @Summary 短信验证码登录
// @Description 短信验证码登录
// @Accept  json
// @Produce  json
// @Param body body api.LoginSmsForm true "登录表单"
// @Success 200 {object} apiResp.LoginRes
// @Failure 500 {object} response.ErrorResponse
// @Router /login-sms [post]
func (l *Login) LoginSms(c *gin.Context) {
	loginLimiter := global.LoginLimiter
	clientIp := c.ClientIP()
	banned, _ := loginLimiter.CheckSecurityStatus(clientIp)
	if banned {
		response.Error(c, response.TranslateMsg(c, "LoginBanned"))
		return
	}

	f := &api.LoginSmsForm{}
	err := c.ShouldBindJSON(f)
	if err != nil {
		global.Logger.Warn(fmt.Sprintf("LoginSms Fail: %s %s %s", "ParamsError", c.RemoteIP(), clientIp))
		response.Error(c, response.TranslateMsg(c, "ParamsError")+err.Error())
		return
	}

	errList := global.Validator.ValidStruct(c, f)
	if len(errList) > 0 {
		global.Logger.Warn(fmt.Sprintf("LoginSms Fail: %s %s %s", "ParamsError", c.RemoteIP(), clientIp))
		response.Error(c, errList[0])
		return
	}

	if !service.AllService.SmsService.VerifyLoginCode(f.Phone, f.Code) {
		loginLimiter.RecordFailedAttempt(clientIp)
		global.Logger.Warn(fmt.Sprintf("LoginSms Fail: %s %s %s", "SmsCodeError", c.RemoteIP(), clientIp))
		response.Error(c, response.TranslateMsg(c, "SmsCodeError"))
		return
	}

	u := service.AllService.UserService.InfoByPhone(f.Phone)
	if u.Id == 0 {
		// 手机号未注册, 按注册开关自动注册
		if !global.Config.App.Register {
			response.Error(c, response.TranslateMsg(c, "RegisterClosed"))
			return
		}
		u = service.AllService.UserService.RegisterByPhone(f.Phone)
		if u == nil || u.Id == 0 {
			response.Error(c, response.TranslateMsg(c, "OperationFailed"))
			return
		}
	}

	if !service.AllService.UserService.CheckUserEnable(u) {
		response.Error(c, response.TranslateMsg(c, "UserDisabled"))
		return
	}
	if err := service.AllService.MfaService.VerifyLogin(u, f.MfaCode); err != nil {
		if err == service.ErrMfaRequired {
			challenge, challengeErr := service.AllService.MfaService.CreateEnrollmentChallenge(u, &service.MfaEnrollmentChallengeItem{
				Id:         f.Id,
				Uuid:       f.Uuid,
				DeviceOs:   f.DeviceInfo.Os,
				DeviceType: model.LoginLogClientApp,
				LoginType:  model.LoginLogTypeSms,
			})
			if challengeErr != nil {
				response.Error(c, response.TranslateMsg(c, challengeErr.Error()))
				return
			}
			c.JSON(http.StatusOK, gin.H{"mfa_enrollment_required": true, "mfa_enrollment_challenge": challenge})
			return
		}
		loginLimiter.RecordFailedAttempt(clientIp)
		response.Error(c, response.TranslateMsg(c, err.Error()))
		return
	}

	ut := service.AllService.UserService.Login(u, &model.LoginLog{
		UserId:   u.Id,
		Client:   model.LoginLogClientApp,
		DeviceId: f.Id,
		Uuid:     f.Uuid,
		Ip:       clientIp,
		Type:     model.LoginLogTypeSms,
		Platform: f.DeviceInfo.Os,
	})

	// 登录成功，清除登录限制
	loginLimiter.RemoveAttempts(clientIp)
	c.JSON(http.StatusOK, apiResp.LoginRes{
		AccessToken: ut.Token,
		Type:        "access_token",
		User:        *(&apiResp.UserPayload{}).FromUser(u),
	})
}

// LoginOptions
// @Tags 登录
// @Summary 登录选项
// @Description 登录选项
// @Accept  json
// @Produce  json
// @Success 200 {object} []string
// @Failure 500 {object} response.ErrorResponse
// @Router /login-options [get]
func (l *Login) LoginOptions(c *gin.Context) {
	ops := service.AllService.OauthService.GetOauthProviders()
	if global.Config.App.WebSso {
		ops = append(ops, model.OauthTypeWebauth)
	}
	var oidcItems []map[string]string
	for _, v := range ops {
		oidcItems = append(oidcItems, map[string]string{"name": v})
	}
	common, err := json.Marshal(oidcItems)
	if err != nil {
		response.Error(c, response.TranslateMsg(c, "SystemError")+err.Error())
		return
	}
	var res []string
	res = append(res, "common-oidc/"+string(common))
	for _, v := range ops {
		res = append(res, "oidc/"+v)
	}
	c.JSON(http.StatusOK, res)
}

// Logout
// @Tags 登录
// @Summary 登出
// @Description 登出
// @Accept  json
// @Produce  json
// @Success 200 {string} string
// @Failure 500 {object} response.ErrorResponse
// @Router /logout [post]
func (l *Login) Logout(c *gin.Context) {
	u := service.AllService.UserService.CurUser(c)
	token, ok := c.Get("token")
	if ok {
		service.AllService.UserService.Logout(u, token.(string))
	}
	c.JSON(http.StatusOK, nil)

}
