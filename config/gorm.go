package config

import (
	"fmt"
	"net"
	"net/url"
	"regexp"
	"strings"
)

const (
	TypeSqlite     = "sqlite"
	TypeMysql      = "mysql"
	TypePostgresql = "postgresql"
)

const (
	PostgresqlPoolModeDirect  = "direct"
	PostgresqlPoolModeSession = "session"
)

var postgresqlSchemaPattern = regexp.MustCompile(`^[a-z_][a-z0-9_]{0,62}$`)

type Gorm struct {
	Type         string `mapstructure:"type"`
	MaxIdleConns int    `mapstructure:"max-idle-conns"`
	MaxOpenConns int    `mapstructure:"max-open-conns"`
	AutoMigrate  bool   `mapstructure:"auto-migrate"`
}

type Mysql struct {
	Addr     string `mapstructure:"addr"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
	Dbname   string `mapstructure:"dbname"`
	Tls      string `mapstructure:"tls"` // true / false / skip-verify / custom
}

type Postgresql struct {
	Host        string `mapstructure:"host"`
	Port        string `mapstructure:"port"`
	User        string `mapstructure:"user"`
	Password    string `mapstructure:"password"`
	Dbname      string `mapstructure:"dbname"`
	Sslmode     string `mapstructure:"sslmode"`       // "disable", "require", "verify-ca", "verify-full"
	SslRootCert string `mapstructure:"ssl-root-cert"` // optional PEM path for verify-ca / verify-full
	TimeZone    string `mapstructure:"time-zone"`     // e.g., "Asia/Shanghai"
	Schema      string `mapstructure:"schema"`        // unquoted lowercase PostgreSQL identifier
	PoolMode    string `mapstructure:"pool-mode"`     // direct or session; transaction mode is unsupported
	JITAccess   bool   `mapstructure:"jit-access"`    // Supabase temporary access; token is injected as the password
}

func (p Postgresql) SchemaName() string {
	schema := strings.TrimSpace(p.Schema)
	if schema == "" {
		return "public"
	}
	return schema
}

func (p Postgresql) EffectivePoolMode() string {
	mode := strings.ToLower(strings.TrimSpace(p.PoolMode))
	if mode == "" {
		return PostgresqlPoolModeDirect
	}
	return mode
}

func (p Postgresql) ValidateConnectionBoundary() error {
	if !postgresqlSchemaPattern.MatchString(p.SchemaName()) {
		return fmt.Errorf("postgresql.schema must be a lowercase unquoted identifier of at most 63 characters")
	}
	switch p.EffectivePoolMode() {
	case PostgresqlPoolModeDirect, PostgresqlPoolModeSession:
		return nil
	default:
		return fmt.Errorf("postgresql.pool-mode must be direct or session; transaction pooling is unsupported")
	}
}

// DSN returns a URI rather than a keyword/value string so passwords and other
// connection fields are escaped correctly. The schema is validated before it
// is used as search_path and must never be supplied as raw SQL.
func (p Postgresql) DSN() (string, error) {
	if err := p.ValidateConnectionBoundary(); err != nil {
		return "", err
	}

	host := strings.TrimSpace(p.Host)
	if p.Port != "" {
		host = net.JoinHostPort(host, p.Port)
	}
	u := &url.URL{
		Scheme: "postgresql",
		User:   url.UserPassword(p.User, p.Password),
		Host:   host,
		Path:   "/" + p.Dbname,
	}
	query := u.Query()
	query.Set("sslmode", p.Sslmode)
	query.Set("search_path", p.SchemaName())
	query.Set("application_name", "yinhe-rustdesk-api")
	query.Set("connect_timeout", "10")
	if p.JITAccess && p.EffectivePoolMode() == PostgresqlPoolModeSession {
		// Supavisor requires this startup parameter when a short-lived
		// Supabase platform token is used instead of a database password.
		query.Set("options", "-c jit=true")
	}
	if p.SslRootCert != "" {
		query.Set("sslrootcert", p.SslRootCert)
	}
	u.RawQuery = query.Encode()
	return u.String(), nil
}
