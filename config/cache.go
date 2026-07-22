package config

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
)

type Cache struct {
	Type               string
	RedisAddr          string `mapstructure:"redis-addr"`
	RedisUsername      string `mapstructure:"redis-username"`
	RedisPwd           string `mapstructure:"redis-pwd"`
	RedisDb            int    `mapstructure:"redis-db"`
	RedisTLS           bool   `mapstructure:"redis-tls"`
	RedisTLSCAFile     string `mapstructure:"redis-tls-ca-file"`
	RedisTLSServerName string `mapstructure:"redis-tls-server-name"`
	RedisKeyPrefix     string `mapstructure:"redis-key-prefix"`
	FileDir            string `mapstructure:"file-dir"`
}

// RedisOptions creates a verification-enabled Redis client configuration.
// TLS certificates are always validated; there is intentionally no
// insecure-skip-verify setting.
func (c Cache) RedisOptions() (*redis.Options, error) {
	addr := strings.TrimSpace(c.RedisAddr)
	if addr == "" {
		return nil, errors.New("cache.redis-addr is empty")
	}
	opts := &redis.Options{
		Addr:         addr,
		Username:     c.RedisUsername,
		Password:     c.RedisPwd,
		DB:           c.RedisDb,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		PoolTimeout:  4 * time.Second,
	}
	if !c.RedisTLS {
		return opts, nil
	}

	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, fmt.Errorf("cache.redis-addr must include host and port: %w", err)
	}
	serverName := strings.TrimSpace(c.RedisTLSServerName)
	if serverName == "" {
		serverName = host
	}
	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS12,
		ServerName: serverName,
	}
	if caFile := strings.TrimSpace(c.RedisTLSCAFile); caFile != "" {
		pem, err := os.ReadFile(caFile)
		if err != nil {
			return nil, fmt.Errorf("read cache.redis-tls-ca-file: %w", err)
		}
		roots, err := x509.SystemCertPool()
		if err != nil || roots == nil {
			roots = x509.NewCertPool()
		}
		if !roots.AppendCertsFromPEM(pem) {
			return nil, errors.New("cache.redis-tls-ca-file contains no valid certificate")
		}
		tlsConfig.RootCAs = roots
	}
	opts.TLSConfig = tlsConfig
	return opts, nil
}
