package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/lejianwen/rustdesk-api/v2/global"
	"github.com/lejianwen/rustdesk-api/v2/http/response"
	"github.com/lejianwen/rustdesk-api/v2/service"
)

// BackendUserAuth 后台权限验证中间件
func BackendUserAuth() gin.HandlerFunc {
	return func(c *gin.Context) {

		//测试先关闭
		token := c.GetHeader("api-token")
		if token == "" {
			response.Fail(c, 403, response.TranslateMsg(c, "NeedLogin"))
			c.Abort()
			return
		}
		// api-token is also a JWT. Checking it here prevents the database-side
		// sliding refresh from extending an already expired bearer credential.
		if global.Jwt != nil && len(global.Jwt.Key) > 0 {
			uid, err := service.AllService.UserService.VerifyJWT(token)
			if err != nil || uid == 0 {
				response.Fail(c, 403, response.TranslateMsg(c, "NeedLogin"))
				c.Abort()
				return
			}
		}
		user, ut := service.AllService.UserService.InfoByAccessToken(token)
		if user.Id == 0 {
			response.Fail(c, 403, response.TranslateMsg(c, "NeedLogin"))
			c.Abort()
			return
		}
		if global.Jwt != nil && len(global.Jwt.Key) > 0 {
			uid, _ := service.AllService.UserService.VerifyJWT(token)
			if uid != user.Id {
				response.Fail(c, 403, response.TranslateMsg(c, "NeedLogin"))
				c.Abort()
				return
			}
		}

		if !service.AllService.UserService.CheckUserEnable(user) {
			c.JSON(401, gin.H{
				"error": "Unauthorized",
			})
			c.Abort()
			return
		}

		c.Set("curUser", user)
		c.Set("token", token)
		//如果时间小于1天,token自动续期
		service.AllService.UserService.AutoRefreshAccessToken(ut)

		c.Next()
	}
}
