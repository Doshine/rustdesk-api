package middleware

import (
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// 审计上报接口（/api/audit/conn、/api/audit/file）是**匿名**的，且必须保持匿名：
// 被控端上报时不带任何 Authorization（rustdesk/src/server/connection.rs 的
// post_audit_async 以空 header 调用 post_request），加鉴权会直接切断全部审计采集。
//
// 但匿名 + append-only 的组合意味着：任何人都能往审计表里写伪造记录，而合法
// 管理员又删不掉。因此在保持匿名的前提下必须加上三道限制：
//   1. 请求体大小上限——避免单请求撑爆存储；
//   2. 来源 IP 限速——避免高频灌入；
//   3. 设备身份校验（在控制器内做，见 controller/api/audit.go）。

const (
	auditMaxBodyBytes   = 16 << 10 // 16 KiB，远大于正常审计事件
	auditWindow         = time.Minute
	auditMaxPerIPWindow = 120 // 单 IP 每分钟上限；正常客户端只在会话起止各报一次
)

type auditRateLimiter struct {
	mu      sync.Mutex
	buckets map[string][]time.Time
	last    time.Time
}

var auditLimiter = &auditRateLimiter{buckets: make(map[string][]time.Time)}

func (l *auditRateLimiter) allow(key string) bool {
	now := time.Now()
	cutoff := now.Add(-auditWindow)

	l.mu.Lock()
	defer l.mu.Unlock()

	// 顺带做惰性清理，避免 map 随来源 IP 数量无界增长
	if now.Sub(l.last) > auditWindow {
		for k, ts := range l.buckets {
			kept := ts[:0]
			for _, t := range ts {
				if t.After(cutoff) {
					kept = append(kept, t)
				}
			}
			if len(kept) == 0 {
				delete(l.buckets, k)
			} else {
				l.buckets[k] = kept
			}
		}
		l.last = now
	}

	kept := l.buckets[key][:0]
	for _, t := range l.buckets[key] {
		if t.After(cutoff) {
			kept = append(kept, t)
		}
	}
	if len(kept) >= auditMaxPerIPWindow {
		l.buckets[key] = kept
		return false
	}
	l.buckets[key] = append(kept, now)
	return true
}

// AuditIngestGuard 为匿名审计上报接口提供体积与速率限制。
func AuditIngestGuard() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, auditMaxBodyBytes)
		if !auditLimiter.allow(c.ClientIP()) {
			// 不返回 429 的细节，避免把限流阈值暴露成可探测的信号
			c.AbortWithStatus(http.StatusTooManyRequests)
			return
		}
		c.Next()
	}
}

// 保留 io 导入的显式引用，防止后续改动误删（MaxBytesReader 返回 io.ReadCloser）
var _ io.ReadCloser = http.NoBody
