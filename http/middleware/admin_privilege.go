package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/lejianwen/rustdesk-api/v2/http/response"
	"github.com/lejianwen/rustdesk-api/v2/model"
	"github.com/lejianwen/rustdesk-api/v2/service"
)

// AdminPrivilege ...
func AdminPrivilege() gin.HandlerFunc {
	return func(c *gin.Context) {
		u := service.AllService.UserService.CurUser(c)

		if !service.AllService.UserService.IsAdmin(u) {
			response.Fail(c, 403, response.TranslateMsg(c, "NoAccess"))
			c.Abort()
			return
		}

		c.Next()
	}
}

// PermissionPrivilege authorizes a least-privilege role for a specific API
// capability while keeping AdminPrivilege unchanged for legacy routes.
func PermissionPrivilege(permission string) gin.HandlerFunc {
	return func(c *gin.Context) {
		u := service.AllService.UserService.CurUser(c)
		if !service.AllService.UserService.HasPermission(u, permission) {
			response.Fail(c, 403, response.TranslateMsg(c, "NoAccess"))
			c.Abort()
			return
		}
		c.Next()
	}
}

// AuditReadPrivilege is kept as a named middleware for route readability.
func AuditReadPrivilege() gin.HandlerFunc {
	return PermissionPrivilege(model.PermissionAuditRead)
}
