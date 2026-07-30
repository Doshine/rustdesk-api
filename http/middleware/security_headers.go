package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
)

// SecurityHeaders 为所有响应补齐基础安全响应头。
//
// 此前全体系没有任何安全响应头：管理后台与 Web Client 同源、由 Gin StaticFS
// 裸服务，既无 CSP 也无 frame-ancestors/X-Frame-Options，任何同源 XSS 或
// 点击劫持都没有第二道防线。
//
// CSP 分两档：
//   - 静态页面（管理后台 / Web Client）：允许自身脚本与内联样式。
//     不能直接上 script-src 'self' 严格模式——Web Client 的 index.html 含
//     内联引导脚本、Flutter 产物依赖 wasm 与 blob worker，一刀切会直接白屏。
//     因此这里的目标是"挡住外部脚本源与被嵌套"，而不是完整的 nonce 化 CSP；
//     后者需要改造前端构建，属独立工作项。
//   - API 响应：给最严格的策略，API 不应加载任何资源。
func SecurityHeaders() gin.HandlerFunc {
	const (
		staticCSP = "default-src 'self'; " +
			"script-src 'self' 'unsafe-inline' 'unsafe-eval' blob:; " +
			"style-src 'self' 'unsafe-inline'; " +
			"img-src 'self' data: blob:; " +
			"font-src 'self' data:; " +
			"connect-src 'self' ws: wss:; " +
			"worker-src 'self' blob:; " +
			"object-src 'none'; " +
			"base-uri 'self'; " +
			"form-action 'self'; " +
			"frame-ancestors 'none'"
		apiCSP = "default-src 'none'; frame-ancestors 'none'; base-uri 'none'; form-action 'none'"
	)

	return func(c *gin.Context) {
		h := c.Writer.Header()
		// 与 CSP frame-ancestors 并存：老浏览器只认 X-Frame-Options
		h.Set("X-Frame-Options", "DENY")
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		h.Set("Cross-Origin-Opener-Policy", "same-origin")
		h.Set("Permissions-Policy", "geolocation=(), payment=(), usb=()")

		if strings.HasPrefix(c.Request.URL.Path, "/api/") {
			h.Set("Content-Security-Policy", apiCSP)
		} else {
			h.Set("Content-Security-Policy", staticCSP)
		}

		// HSTS 只在确实经 TLS 到达时下发。生产由 Rainbond 网关终止 TLS，
		// 此处依据 X-Forwarded-Proto 判断；该头只有在 gin.trust-proxy 正确
		// 配置时才可信，而生产配置校验已要求填写它。
		if c.Request.TLS != nil || strings.EqualFold(c.GetHeader("X-Forwarded-Proto"), "https") {
			h.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}

		c.Next()
	}
}
