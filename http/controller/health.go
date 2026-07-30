package controller

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lejianwen/rustdesk-api/v2/lib/cache"
	"gorm.io/gorm"
)

const defaultReadinessTimeout = 3 * time.Second

// readinessCacheTTL 是 /health/ready 探测结果的复用窗口。
//
// 该端点无鉴权、且注册在限流中间件之前（供编排器在认证流量高峰时仍能可靠
// 探测），如果每个请求都真实 Ping 数据库与 Redis，就成了一个低成本的连接池
// 耗尽面：外部只需高频请求即可占满连接。
//
// 2 秒窗口既远小于常见的 10-30 秒探测间隔（编排器拿到的仍是新鲜结果），
// 又能把突发请求折叠成一次真实探测。
const readinessCacheTTL = 2 * time.Second

// Health exposes non-authenticated orchestration probes without returning
// dependency names, addresses, credentials, or raw error messages.
type Health struct {
	databasePing func(context.Context) error
	cachePing    func(context.Context) error
	timeout      time.Duration

	mu          sync.Mutex
	lastChecked time.Time
	lastHealthy bool
	// inflight 保证并发请求只触发一次真实探测，其余等待同一结果
	inflight *sync.WaitGroup
}

func NewHealth(db *gorm.DB, ca cache.Handler, requireExternalCache bool) *Health {
	databasePing := func(ctx context.Context) error {
		if db == nil {
			return errors.New("database is not initialized")
		}
		sqlDB, err := db.DB()
		if err != nil {
			return err
		}
		return sqlDB.PingContext(ctx)
	}
	cachePing := func(ctx context.Context) error {
		checker, ok := ca.(cache.HealthChecker)
		if !ok {
			if requireExternalCache {
				return errors.New("external cache health check is unavailable")
			}
			return nil
		}
		return checker.Ping(ctx)
	}
	return &Health{databasePing: databasePing, cachePing: cachePing, timeout: defaultReadinessTimeout}
}

func (h *Health) Live(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (h *Health) Ready(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	if !h.readiness() {
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "unavailable"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// readiness 返回依赖健康状态，并对真实探测做短窗口缓存与并发合并，
// 使无鉴权的 /health/ready 不能被用来放大数据库与缓存的连接压力。
func (h *Health) readiness() bool {
	h.mu.Lock()
	if !h.lastChecked.IsZero() && time.Since(h.lastChecked) < readinessCacheTTL {
		healthy := h.lastHealthy
		h.mu.Unlock()
		return healthy
	}
	if wg := h.inflight; wg != nil {
		// 已有一次真实探测在进行，等它出结果，不再新开连接
		h.mu.Unlock()
		wg.Wait()
		h.mu.Lock()
		healthy := h.lastHealthy
		h.mu.Unlock()
		return healthy
	}
	wg := &sync.WaitGroup{}
	wg.Add(1)
	h.inflight = wg
	h.mu.Unlock()

	healthy := h.probe()

	h.mu.Lock()
	h.lastChecked = time.Now()
	h.lastHealthy = healthy
	h.inflight = nil
	h.mu.Unlock()
	wg.Done()
	return healthy
}

func (h *Health) probe() bool {
	timeout := h.timeout
	if timeout <= 0 {
		timeout = defaultReadinessTimeout
	}
	// 用独立 context 而非请求 context：探测结果会被其他等待者复用，
	// 不能让某一个客户端断开就把共享结果污染成不健康。
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	if h.databasePing == nil || h.cachePing == nil {
		return false
	}
	if h.databasePing(ctx) != nil {
		return false
	}
	return h.cachePing(ctx) == nil
}
