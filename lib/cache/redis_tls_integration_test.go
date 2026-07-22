package cache_test

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/lejianwen/rustdesk-api/v2/config"
	appcache "github.com/lejianwen/rustdesk-api/v2/lib/cache"
)

func TestRedisTLSACLIntegration(t *testing.T) {
	if os.Getenv("YINHE_REDIS_TLS_SMOKE") != "1" {
		t.Skip("set YINHE_REDIS_TLS_SMOKE=1 to run the Redis TLS/ACL integration test")
	}
	password := readRedisSecret(t, "YINHE_REDIS_PASSWORD_FILE")
	cfg := config.Cache{
		Type: appcache.TypeRedis, RedisAddr: os.Getenv("YINHE_REDIS_ADDR"),
		RedisUsername: os.Getenv("YINHE_REDIS_USERNAME"), RedisPwd: password,
		RedisTLS: true, RedisTLSCAFile: os.Getenv("YINHE_REDIS_CA_FILE"),
		RedisTLSServerName: os.Getenv("YINHE_REDIS_TLS_SERVER_NAME"),
		RedisKeyPrefix:     os.Getenv("YINHE_REDIS_KEY_PREFIX"),
	}
	if cfg.RedisAddr == "" || cfg.RedisUsername == "" || cfg.RedisTLSCAFile == "" || cfg.RedisTLSServerName == "" || cfg.RedisKeyPrefix == "" {
		t.Fatal("Redis address, username, CA, TLS server name, and key prefix are required")
	}
	opts, err := cfg.RedisOptions()
	if err != nil {
		t.Fatalf("build Redis TLS options: %v", err)
	}

	dialer := &net.Dialer{Timeout: 5 * time.Second}
	tlsConn, err := tls.DialWithDialer(dialer, "tcp", opts.Addr, opts.TLSConfig.Clone())
	if err != nil {
		t.Fatalf("verify Redis TLS handshake: %v", err)
	}
	state := tlsConn.ConnectionState()
	_ = tlsConn.Close()
	if state.Version < tls.VersionTLS12 || len(state.VerifiedChains) == 0 {
		t.Fatal("Redis connection did not verify TLS 1.2+ certificate chain")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	rc := appcache.NewRedisWithPrefix(opts, cfg.RedisKeyPrefix)
	defer rc.Close()
	if err := rc.Ping(ctx); err != nil {
		t.Fatalf("Redis ACL PING failed: %v", err)
	}
	key := fmt.Sprintf("db03:%d", time.Now().UnixNano())
	defer rc.Delete(key)
	if err := rc.Set(key, "one-time-value", 60); err != nil {
		t.Fatalf("Redis TLS SET failed: %v", err)
	}
	var got string
	if err := rc.Get(key, &got); err != nil || got != "one-time-value" {
		t.Fatalf("Redis TLS GET failed: value=%q err=%v", got, err)
	}
	if count, err := rc.Increment(key+":counter", 60); err != nil || count != 1 {
		t.Fatalf("Redis atomic counter failed: count=%d err=%v", count, err)
	}
	defer rc.Delete(key + ":counter")
	if err := rc.GetAndDelete(key, &got); err != nil || got != "one-time-value" {
		t.Fatalf("Redis atomic get-and-delete failed: value=%q err=%v", got, err)
	}

	raw := redis.NewClient(opts)
	defer raw.Close()
	outsideKey := fmt.Sprintf("outside-db03:%d", time.Now().UnixNano())
	err = raw.Set(ctx, outsideKey, "must-be-denied", time.Minute).Err()
	if err == nil || !strings.Contains(strings.ToUpper(err.Error()), "NOPERM") {
		t.Fatalf("Redis ACL did not deny an out-of-prefix key: %v", err)
	}
}

func readRedisSecret(t *testing.T, envName string) string {
	t.Helper()
	path := strings.TrimSpace(os.Getenv(envName))
	if path == "" {
		t.Fatalf("%s must point to a mounted secret file", envName)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read mounted Redis secret for %s", envName)
	}
	value := strings.TrimRight(string(b), "\r\n")
	if value == "" {
		t.Fatalf("mounted Redis secret for %s is empty", envName)
	}
	return value
}
