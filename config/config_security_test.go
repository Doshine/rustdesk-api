package config

import (
	"crypto/tls"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestProductionSecurityValidationRejectsInsecureDefaults(t *testing.T) {
	cfg := &Config{}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected insecure production defaults to be rejected")
	}
}

func TestDevelopmentOverrideIsExplicit(t *testing.T) {
	cfg := &Config{App: App{AllowInsecureDevelopment: true, CaptchaThreshold: -1}}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("explicit development override should pass: %v", err)
	}
}

func TestProductionSecurityValidationAcceptsHardenedConfig(t *testing.T) {
	cfg := &Config{
		App: App{CaptchaThreshold: 3, BanThreshold: 5, TokenExpire: 24 * time.Hour},
		Jwt: Jwt{Key: "0123456789abcdef0123456789abcdef", ExpireDuration: 24 * time.Hour},
		Cache: Cache{
			Type: "redis", RedisAddr: "redis:6379", RedisUsername: "yinhe", RedisPwd: "redis-secret",
			RedisTLS: true, RedisTLSServerName: "redis", RedisKeyPrefix: "yinhe:test:",
		},
		Sms: SmsConfig{
			Provider: "aliyun", AccessKeyId: "id", AccessKeySecret: "secret",
			SignName: "sign", TemplateCode: "template",
		},
		Gorm:       Gorm{Type: TypePostgresql},
		Postgresql: Postgresql{Sslmode: "verify-full"},
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("hardened production config should pass: %v", err)
	}
}

func TestInvalidCaptchaAndBanCombinationIsRejected(t *testing.T) {
	cfg := &Config{App: App{AllowInsecureDevelopment: true, CaptchaThreshold: -1, BanThreshold: 5}}
	if err := cfg.Validate(); err == nil {
		t.Fatal("negative captcha threshold with banning enabled must be rejected")
	}
}

func TestMfaRequiresEncryptionMaterialWhenEnabled(t *testing.T) {
	cfg := &Config{
		App: App{AllowInsecureDevelopment: true},
		Mfa: MfaConfig{Enabled: true},
	}
	// Development override intentionally bypasses production checks; MFA
	// encryption is still validated by the service before enrollment.
	if err := cfg.Validate(); err != nil {
		t.Fatalf("development override should not reject MFA config: %v", err)
	}

	cfg.App.AllowInsecureDevelopment = false
	if err := cfg.Validate(); err == nil {
		t.Fatal("production MFA must require encryption material")
	}
	cfg.Jwt.Key = "0123456789abcdef0123456789abcdef"
	cfg.Mfa.EncryptionKey = "too-short"
	if err := cfg.Validate(); err == nil {
		t.Fatal("configured MFA encryption key must be strong enough")
	}
}

func TestPasskeyRequiresExplicitRelyingPartyBoundary(t *testing.T) {
	cfg := &Config{
		App: App{CaptchaThreshold: 3, TokenExpire: 24 * time.Hour},
		Jwt: Jwt{Key: "0123456789abcdef0123456789abcdef", ExpireDuration: 24 * time.Hour},
		Cache: Cache{
			Type: "redis", RedisAddr: "redis:6379", RedisUsername: "yinhe", RedisPwd: "redis-secret",
			RedisTLS: true, RedisTLSServerName: "redis", RedisKeyPrefix: "yinhe:test:",
		},
		Sms:  SmsConfig{Provider: "aliyun", AccessKeyId: "id", AccessKeySecret: "secret", SignName: "sign", TemplateCode: "template"},
		Gorm: Gorm{Type: TypePostgresql}, Postgresql: Postgresql{Sslmode: "verify-full"},
		Passkey: PasskeyConfig{Enabled: true, RPDisplayName: "Yinhe", Origins: []string{"https://console.example.com"}},
	}
	if err := cfg.Validate(); err == nil {
		t.Fatal("passkey must require an explicit rp-id")
	}
	cfg.Passkey.RPID = "console.example.com"
	if err := cfg.Validate(); err != nil {
		t.Fatalf("explicit passkey relying-party boundary should pass: %v", err)
	}
}

func TestProductionRedisRequiresTLSACLAndKeyPrefix(t *testing.T) {
	cfg := &Config{
		App:   App{CaptchaThreshold: 3, TokenExpire: 24 * time.Hour},
		Jwt:   Jwt{Key: "0123456789abcdef0123456789abcdef", ExpireDuration: 24 * time.Hour},
		Cache: Cache{Type: "redis", RedisAddr: "redis:6379"},
		Sms:   SmsConfig{Provider: "aliyun", AccessKeyId: "id", AccessKeySecret: "secret", SignName: "sign", TemplateCode: "template"},
		Gorm:  Gorm{Type: TypePostgresql}, Postgresql: Postgresql{Sslmode: "verify-full"},
	}
	if err := cfg.Validate(); err == nil {
		t.Fatal("production Redis without TLS, ACL credentials, and key prefix must be rejected")
	}
	cfg.Cache.RedisUsername = "yinhe"
	cfg.Cache.RedisPwd = "redis-secret"
	cfg.Cache.RedisTLS = true
	cfg.Cache.RedisKeyPrefix = "yinhe:prod:"
	if err := cfg.Validate(); err != nil {
		t.Fatalf("hardened Redis boundary should pass: %v", err)
	}
	opts, err := cfg.Cache.RedisOptions()
	if err != nil {
		t.Fatalf("build Redis TLS options: %v", err)
	}
	if opts.TLSConfig == nil || opts.TLSConfig.MinVersion != tls.VersionTLS12 || opts.Username != "yinhe" {
		t.Fatal("Redis TLS/ACL options were not applied")
	}
	cfg.Cache.RedisDb = 1
	if err := cfg.Validate(); err == nil {
		t.Fatal("production Redis must use DB 0 and key-prefix isolation")
	}
}

func TestPostgresqlDSNEscapesCredentialsAndSetsPrivateSchema(t *testing.T) {
	cfg := Postgresql{
		Host: "db.example.com", Port: "5432", User: "app.user",
		Password: "space + slash/colon:", Dbname: "postgres",
		Sslmode: "verify-full", TimeZone: "Asia/Shanghai",
		Schema: "yinhe_app", PoolMode: PostgresqlPoolModeSession,
	}
	dsn, err := cfg.DSN()
	if err != nil {
		t.Fatalf("build PostgreSQL DSN: %v", err)
	}
	parsed, err := url.Parse(dsn)
	if err != nil {
		t.Fatalf("parse PostgreSQL DSN: %v", err)
	}
	password, ok := parsed.User.Password()
	if !ok || password != cfg.Password {
		t.Fatal("PostgreSQL password was not URI-escaped losslessly")
	}
	if parsed.Query().Get("search_path") != "yinhe_app" {
		t.Fatalf("expected private search_path, got %q", parsed.Query().Get("search_path"))
	}
	if strings.Contains(dsn, cfg.Password) {
		t.Fatal("raw PostgreSQL password must not appear unescaped in the DSN")
	}
}

func TestPostgresqlDSNEnablesSupabaseJITOnlyForSessionPooler(t *testing.T) {
	session := Postgresql{
		Host: "aws-1-us-west-2.pooler.supabase.com", Port: "5432",
		User: "yinhe_app_runtime.project-ref", Password: "short-lived-token", Dbname: "postgres",
		Sslmode: "verify-full", Schema: "yinhe_app", PoolMode: PostgresqlPoolModeSession,
		JITAccess: true,
	}
	dsn, err := session.DSN()
	if err != nil {
		t.Fatalf("build Supabase session DSN: %v", err)
	}
	parsed, err := url.Parse(dsn)
	if err != nil {
		t.Fatalf("parse Supabase session DSN: %v", err)
	}
	if got := parsed.Query().Get("options"); got != "-c jit=true" {
		t.Fatalf("expected Supavisor JIT startup option, got %q", got)
	}

	direct := session
	direct.PoolMode = PostgresqlPoolModeDirect
	directDSN, err := direct.DSN()
	if err != nil {
		t.Fatalf("build Supabase direct DSN: %v", err)
	}
	directParsed, err := url.Parse(directDSN)
	if err != nil {
		t.Fatalf("parse Supabase direct DSN: %v", err)
	}
	if got := directParsed.Query().Get("options"); got != "" {
		t.Fatalf("direct JIT access must not send the Supavisor option, got %q", got)
	}
}

func TestPostgresqlConnectionBoundaryRejectsUnsafeSchemaAndTransactionPooler(t *testing.T) {
	unsafeSchema := Postgresql{Schema: "public, malicious", PoolMode: PostgresqlPoolModeDirect}
	if err := unsafeSchema.ValidateConnectionBoundary(); err == nil {
		t.Fatal("unsafe search_path schema must be rejected")
	}

	transactionPooler := Postgresql{Schema: "yinhe_app", PoolMode: "transaction"}
	if err := transactionPooler.ValidateConnectionBoundary(); err == nil {
		t.Fatal("transaction pooler must be rejected for the current GORM/pgx path")
	}
}
