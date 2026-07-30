package controller

import (
	"crypto/subtle"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// 有意不引入 prometheus/client_golang：
// Prometheus 文本暴露格式是稳定的公开规范，这里需要的只是若干 gauge 与
// counter，自己格式化即可，避免为一个最小可观测性起步引入一整条依赖链
// （以及随之而来的 go.sum 变更与供应链面）。
// 若将来需要 histogram、exemplar 等能力，再换成官方库不迟。

// HTTPStats 由中间件累加，进程内计数。
//
// 注意这是**进程内**状态：多副本部署时每个副本各报各的，Prometheus 按
// instance 标签区分即可；但进程重启会清零，因此这些是 counter 语义，
// 查询时应使用 rate()/increase() 而不是绝对值。
type HTTPStats struct {
	total    atomic.Uint64
	byClass  [6]atomic.Uint64 // index 0 未用；1xx..5xx 对应 1..5
	inflight atomic.Int64
}

var httpStats = &HTTPStats{}

// HTTPStatsMiddleware 统计请求量与状态分布。开销是几个原子加法。
func HTTPStatsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		httpStats.inflight.Add(1)
		c.Next()
		httpStats.inflight.Add(-1)
		httpStats.total.Add(1)
		class := c.Writer.Status() / 100
		if class >= 1 && class <= 5 {
			httpStats.byClass[class].Add(1)
		}
	}
}

type Metrics struct {
	db        *gorm.DB
	schema    string
	token     string
	startedAt time.Time

	mu        sync.Mutex
	lastQuery time.Time
	cached    map[string]int64
}

// metricsQueryTTL 限制业务计数的查库频率。Prometheus 默认 15s 抓取一次，
// 而 count(*) 在大表上不便宜；这里做 30 秒缓存，抓取端不会因此拿到过期数据
// （抓取间隔本身就大于等于这个量级）。同时避免 /metrics 被高频请求时
// 变成又一个数据库压力放大器（与 /health/ready 同类问题）。
const metricsQueryTTL = 30 * time.Second

func NewMetrics(db *gorm.DB, schema, token string) *Metrics {
	return &Metrics{db: db, schema: schema, token: token, startedAt: time.Now()}
}

func (m *Metrics) authorized(c *gin.Context) bool {
	if m.token == "" {
		return false
	}
	h := c.GetHeader("Authorization")
	const p = "Bearer "
	if !strings.HasPrefix(h, p) {
		return false
	}
	// 常量时间比较，避免按前缀逐字节试探
	return subtle.ConstantTimeCompare([]byte(h[len(p):]), []byte(m.token)) == 1
}

func (m *Metrics) Handler(c *gin.Context) {
	if !m.authorized(c) {
		// 不返回 401 的细节，避免把「端点存在」变成可探测信号
		c.AbortWithStatus(http.StatusNotFound)
		return
	}

	var b strings.Builder
	write := func(name, typ, help string, val interface{}, labels string) {
		fmt.Fprintf(&b, "# HELP %s %s\n# TYPE %s %s\n", name, help, name, typ)
		if labels != "" {
			fmt.Fprintf(&b, "%s{%s} %v\n", name, labels, val)
		} else {
			fmt.Fprintf(&b, "%s %v\n", name, val)
		}
	}

	write("yinhe_up", "gauge", "API 进程存活（恒为 1，用于确认抓取链路本身通畅）", 1, "")
	write("yinhe_uptime_seconds", "gauge", "进程启动至今的秒数", int64(time.Since(m.startedAt).Seconds()), "")

	// 依赖健康
	dbUp := 0
	if m.db != nil {
		if sqlDB, err := m.db.DB(); err == nil && sqlDB.Ping() == nil {
			dbUp = 1
		}
	}
	write("yinhe_database_up", "gauge", "数据库可达性（1 可达 / 0 不可达）", dbUp, "")

	// HTTP 计数
	write("yinhe_http_requests_total", "counter", "HTTP 请求总数（不含 /health/* 与 /metrics，它们注册在统计中间件之前；进程内计数，重启清零，请用 rate()）", httpStats.total.Load(), "")
	for class := 1; class <= 5; class++ {
		fmt.Fprintf(&b, "yinhe_http_responses_total{class=\"%dxx\"} %d\n", class, httpStats.byClass[class].Load())
	}
	write("yinhe_http_requests_inflight", "gauge", "当前正在处理的请求数", httpStats.inflight.Load(), "")

	// 业务计数（带缓存）
	for name, help := range map[string]string{
		"yinhe_users_total":       "用户总数",
		"yinhe_peers_total":       "已注册设备总数",
		"yinhe_peers_online":      "最近 5 分钟有心跳的设备数",
		"yinhe_audit_conns_total": "连接审计事件总数",
		"yinhe_login_logs_total":  "登录日志总数",
	} {
		if v, ok := m.counts()[name]; ok {
			write(name, "gauge", help, v, "")
		}
	}

	c.Header("Cache-Control", "no-store")
	c.Data(http.StatusOK, "text/plain; version=0.0.4; charset=utf-8", []byte(b.String()))
}

func (m *Metrics) counts() map[string]int64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.cached != nil && time.Since(m.lastQuery) < metricsQueryTTL {
		return m.cached
	}
	res := make(map[string]int64, 5)
	if m.db == nil {
		m.cached, m.lastQuery = res, time.Now()
		return res
	}
	q := func(metric, table, where string) {
		var n int64
		tx := m.db.Table(m.qualified(table))
		if where != "" {
			tx = tx.Where(where)
		}
		if err := tx.Count(&n).Error; err == nil {
			res[metric] = n
		}
		// 查询失败就不输出该指标——宁可缺一条，也不要报一个会被当成真值的 0
	}
	q("yinhe_users_total", "users", "")
	q("yinhe_peers_total", "peers", "")
	q("yinhe_peers_online", "peers", fmt.Sprintf("last_online_time > %d", time.Now().Add(-5*time.Minute).Unix()))
	q("yinhe_audit_conns_total", "audit_conns", "")
	q("yinhe_login_logs_total", "login_logs", "")

	m.cached, m.lastQuery = res, time.Now()
	return res
}

// qualified 在配置了私有 schema 时补上前缀（PostgreSQL 部署走 yinhe_app）。
func (m *Metrics) qualified(table string) string {
	s := strings.TrimSpace(m.schema)
	if s == "" || s == "public" {
		return table
	}
	return s + "." + table
}
