package controller

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lejianwen/rustdesk-api/v2/lib/cache"
	"gorm.io/gorm"
)

const defaultReadinessTimeout = 3 * time.Second

// Health exposes non-authenticated orchestration probes without returning
// dependency names, addresses, credentials, or raw error messages.
type Health struct {
	databasePing func(context.Context) error
	cachePing    func(context.Context) error
	timeout      time.Duration
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
	timeout := h.timeout
	if timeout <= 0 {
		timeout = defaultReadinessTimeout
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
	defer cancel()
	if h.databasePing == nil || h.cachePing == nil || h.databasePing(ctx) != nil || h.cachePing(ctx) != nil {
		c.Header("Cache-Control", "no-store")
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "unavailable"})
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
