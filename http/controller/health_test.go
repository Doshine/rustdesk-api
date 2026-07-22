package controller

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestHealthLiveDoesNotDependOnExternalServices(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := &Health{}
	r := gin.New()
	r.GET("/health/live", h.Live)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/health/live", nil))
	if w.Code != http.StatusOK || w.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("unexpected liveness response: status=%d cache-control=%q", w.Code, w.Header().Get("Cache-Control"))
	}
}

func TestHealthReadyRequiresDatabaseAndCache(t *testing.T) {
	gin.SetMode(gin.TestMode)
	passes := &Health{
		databasePing: func(context.Context) error { return nil },
		cachePing:    func(context.Context) error { return nil },
		timeout:      time.Second,
	}
	fails := &Health{
		databasePing: func(context.Context) error { return nil },
		cachePing:    func(context.Context) error { return errors.New("redis unavailable") },
		timeout:      time.Second,
	}

	for name, tc := range map[string]struct {
		health *Health
		want   int
	}{
		"ready":              {health: passes, want: http.StatusOK},
		"dependency failure": {health: fails, want: http.StatusServiceUnavailable},
	} {
		t.Run(name, func(t *testing.T) {
			r := gin.New()
			r.GET("/health/ready", tc.health.Ready)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/health/ready", nil))
			if w.Code != tc.want {
				t.Fatalf("unexpected readiness status: got=%d want=%d body=%s", w.Code, tc.want, w.Body.String())
			}
			if strings.Contains(w.Body.String(), "redis unavailable") {
				t.Fatal("readiness response leaked a dependency error")
			}
		})
	}
}
