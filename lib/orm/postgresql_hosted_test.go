package orm_test

import (
	"os"
	"testing"

	"github.com/lejianwen/rustdesk-api/v2/config"
	apporm "github.com/lejianwen/rustdesk-api/v2/lib/orm"
	"gorm.io/gorm/logger"
)

type discardLogWriter struct{}

func (discardLogWriter) Printf(string, ...interface{}) {}

// TestHostedSupabaseSessionConnection is opt-in because it requires a
// short-lived database secret or Supabase platform token. The secret must be
// injected through the environment and must never be written to a config file
// or test log.
func TestHostedSupabaseSessionConnection(t *testing.T) {
	if os.Getenv("YINHE_HOSTED_DB_SMOKE") != "1" {
		t.Skip("set YINHE_HOSTED_DB_SMOKE=1 to run the hosted Supabase smoke test")
	}

	host := os.Getenv("YINHE_SUPABASE_DB_HOST")
	user := os.Getenv("YINHE_SUPABASE_DB_USER")
	secret := os.Getenv("YINHE_SUPABASE_DB_SECRET")
	rootCert := os.Getenv("YINHE_SUPABASE_CA_FILE")
	expectedRole := os.Getenv("YINHE_SUPABASE_EXPECTED_ROLE")
	if host == "" || user == "" || secret == "" || rootCert == "" || expectedRole == "" {
		t.Fatal("hosted smoke test requires host, user, expected role, database secret, and CA file environment variables")
	}

	dsn, err := (config.Postgresql{
		Host: host, Port: "5432", User: user, Password: secret, Dbname: "postgres",
		Sslmode: "verify-full", SslRootCert: rootCert, TimeZone: "Asia/Shanghai",
		Schema: "yinhe_app", PoolMode: config.PostgresqlPoolModeSession,
		JITAccess: os.Getenv("YINHE_SUPABASE_JIT_ACCESS") == "1",
	}).DSN()
	if err != nil {
		t.Fatalf("build hosted Supabase DSN: %v", err)
	}

	db, err := apporm.NewPostgresql(&apporm.PostgresqlConfig{
		Dsn: dsn, TimeZone: "Asia/Shanghai", MaxIdleConns: 1, MaxOpenConns: 1,
	}, discardLogWriter{})
	if err != nil {
		t.Fatalf("connect through application GORM/pgx path: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get hosted connection pool: %v", err)
	}
	defer sqlDB.Close()

	type evidence struct {
		CurrentUser     string
		CurrentSchema   string
		PostgresVersion string
		SSL             bool
		TLSVersion      string
		Cipher          string
		ApplicationName string
		AppVersion      uint
		VersionsRows    int64
		SchemaUsage     bool
		VersionsSelect  bool
		VersionsInsert  bool
		RuntimeMember   bool
	}
	var got evidence
	if err := db.Raw(`
		select current_user,
		       current_schema() as current_schema,
		       current_setting('server_version') as postgres_version,
		       s.ssl,
		       s.version as tls_version,
		       s.cipher,
		       current_setting('application_name') as application_name,
		       (select max(version) from yinhe_app.versions) as app_version,
		       (select count(*) from yinhe_app.versions) as versions_rows,
		       has_schema_privilege(current_user, 'yinhe_app', 'usage') as schema_usage,
		       has_table_privilege(current_user, 'yinhe_app.versions', 'select') as versions_select,
		       has_table_privilege(current_user, 'yinhe_app.versions', 'insert') as versions_insert,
		       pg_has_role(current_user, 'yinhe_app_runtime', 'member') as runtime_member
		from pg_stat_ssl s
		where s.pid = pg_backend_pid()
	`).Scan(&got).Error; err != nil {
		t.Fatalf("query hosted connection evidence: %v", err)
	}

	if got.CurrentUser != expectedRole {
		t.Fatalf("unexpected database role %q", got.CurrentUser)
	}
	if got.CurrentSchema != "yinhe_app" || got.AppVersion != 272 || got.VersionsRows != 1 {
		t.Fatalf("unexpected deployed schema evidence: schema=%q version=%d rows=%d", got.CurrentSchema, got.AppVersion, got.VersionsRows)
	}
	if !got.SSL || got.TLSVersion == "" || got.Cipher == "" {
		t.Fatalf("hosted database connection is not protected by TLS: ssl=%t version=%q cipher=%q", got.SSL, got.TLSVersion, got.Cipher)
	}
	// The shared Session Pooler identifies the server-side connection as
	// Supavisor. The client application name remains asserted in the DSN unit
	// test, but it is not preserved in pg_stat_activity across this pooler.
	if got.ApplicationName != "Supavisor" {
		t.Fatalf("unexpected Session Pooler application_name %q", got.ApplicationName)
	}
	if !got.RuntimeMember || !got.SchemaUsage || !got.VersionsSelect || got.VersionsInsert {
		t.Fatalf("unexpected runtime privileges: member=%t schema_usage=%t versions_select=%t versions_insert=%t", got.RuntimeMember, got.SchemaUsage, got.VersionsSelect, got.VersionsInsert)
	}

	t.Logf("hosted Supabase verified: role=%s schema=%s PostgreSQL=%s TLS=%s app_version=%d",
		got.CurrentUser, got.CurrentSchema, got.PostgresVersion, got.TLSVersion, got.AppVersion)
}

var _ logger.Writer = discardLogWriter{}
