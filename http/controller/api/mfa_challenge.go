package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/lejianwen/rustdesk-api/v2/global"
	apiRequest "github.com/lejianwen/rustdesk-api/v2/http/request/api"
	"github.com/lejianwen/rustdesk-api/v2/http/response"
	apiResp "github.com/lejianwen/rustdesk-api/v2/http/response/api"
	"github.com/lejianwen/rustdesk-api/v2/service"
)

func (o *Oauth) MfaVerify(c *gin.Context) {
	f := &apiRequest.MfaChallengeRequest{}
	if err := c.ShouldBindJSON(f); err != nil {
		response.Error(c, response.TranslateMsg(c, "ParamsError"))
		return
	}
	if errList := global.Validator.ValidStruct(c, f); len(errList) > 0 {
		response.Error(c, errList[0])
		return
	}
	u, ut, err := service.AllService.MfaService.CompleteOauthChallenge(f.Challenge, f.Code, c.ClientIP())
	if err != nil {
		response.Error(c, response.TranslateMsg(c, err.Error()))
		return
	}
	c.JSON(http.StatusOK, apiResp.LoginRes{
		AccessToken: ut.Token,
		Type:        "access_token",
		User:        *(&apiResp.UserPayload{}).FromUser(u),
	})
}
