package config

import (
	"errors"
	"fmt"
	"github.com/spf13/viper"
	"strings"
	"time"
)

const (
	DebugMode     = "debug"
	ReleaseMode   = "release"
	DefaultConfig = "conf/config.yaml"
)

type App struct {
	WebClient                int           `mapstructure:"web-client"`
	Register                 bool          `mapstructure:"register"`
	RegisterStatus           int           `mapstructure:"register-status"`
	ShowSwagger              int           `mapstructure:"show-swagger"`
	TokenExpire              time.Duration `mapstructure:"token-expire"`
	WebSso                   bool          `mapstructure:"web-sso"`
	DisablePwdLogin          bool          `mapstructure:"disable-pwd-login"`
	CaptchaThreshold         int           `mapstructure:"captcha-threshold"`
	BanThreshold             int           `mapstructure:"ban-threshold"`
	AllowInsecureDevelopment bool          `mapstructure:"allow-insecure-development"`
}
type Admin struct {
	Title           string `mapstructure:"title"`
	Hello           string `mapstructure:"hello"`
	HelloFile       string `mapstructure:"hello-file"`
	IdServerPort    int    `mapstructure:"id-server-port"`
	RelayServerPort int    `mapstructure:"relay-server-port"`
}
type Config struct {
	Lang       string `mapstructure:"lang"`
	App        App
	Admin      Admin
	Gorm       Gorm
	Mysql      Mysql
	Postgresql Postgresql
	Gin        Gin
	Logger     Logger
	Cache      Cache
	Oss        Oss
	Jwt        Jwt
	Rustdesk   Rustdesk
	Proxy      Proxy
	Ldap       Ldap
	Sms        SmsConfig
	Mfa        MfaConfig
	Passkey    PasskeyConfig
}

func (a *Admin) Init() {
	if a.IdServerPort == 0 {
		a.IdServerPort = DefaultIdServerPort
	}
	if a.RelayServerPort == 0 {
		a.RelayServerPort = DefaultRelayServerPort
	}
}

// Init 初始化配置
func Init(rowVal *Config, path string) *viper.Viper {
	if path == "" {
		path = DefaultConfig
	}
	v := viper.GetViper()
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
	v.SetEnvPrefix("RUSTDESK_API")
	// Viper's AutomaticEnv is only consulted by Get/IsSet and is not
	// consistently applied during Unmarshal when a value also exists in the
	// YAML file. Bind security-sensitive keys explicitly so operators can
	// override the checked-in development template with environment variables.
	for _, key := range []string{
		"app.allow-insecure-development",
		"jwt.key",
		"oss.access-key-id", "oss.access-key-secret",
		"gorm.type", "gorm.max-idle-conns", "gorm.max-open-conns", "gorm.auto-migrate",
		"cache.type", "cache.redis-addr", "cache.redis-username", "cache.redis-pwd", "cache.redis-db",
		"cache.redis-tls", "cache.redis-tls-ca-file", "cache.redis-tls-server-name", "cache.redis-key-prefix",
		"sms.provider", "sms.access-key-id", "sms.access-key-secret", "sms.sign-name", "sms.template-code", "sms.endpoint",
		"ldap.enable", "ldap.url", "ldap.tls-ca-file", "ldap.tls-verify", "ldap.bind-dn", "ldap.bind-password", "ldap.allow-local-fallback",
		"mysql.addr", "mysql.username", "mysql.password", "mysql.dbname", "mysql.tls",
		"postgresql.host", "postgresql.port", "postgresql.user", "postgresql.password", "postgresql.dbname",
		"postgresql.sslmode", "postgresql.ssl-root-cert", "postgresql.time-zone", "postgresql.schema", "postgresql.pool-mode", "postgresql.jit-access",
		"mfa.enabled", "mfa.issuer", "mfa.encryption-key", "mfa.required-for-admin",
		"passkey.enabled", "passkey.rp-id", "passkey.rp-display-name", "passkey.origins", "passkey.require-user-verification",
	} {
		if err := v.BindEnv(key); err != nil {
			panic(fmt.Errorf("fatal environment binding for %s: %w", key, err))
		}
	}
	v.SetConfigFile(path)
	v.SetConfigType("yaml")
	err := v.ReadInConfig()
	if err != nil {
		panic(fmt.Errorf("Fatal error config file: %s \n", err))
	}
	if err := applySecretFileOverrides(v); err != nil {
		panic(fmt.Errorf("fatal secret file configuration: %w", err))
	}
	/*
		v.WatchConfig()


			//监听配置修改没什么必要
			v.OnConfigChange(func(e fsnotify.Event) {
				//配置文件修改监听
				fmt.Println("config file changed:", e.Name)
				if err2 := v.Unmarshal(rowVal); err2 != nil {
					fmt.Println(err2)
				}
				rowVal.Rustdesk.LoadKeyFile()
				rowVal.Rustdesk.ParsePort()
			})
	*/
	if err := v.Unmarshal(rowVal); err != nil {
		panic(fmt.Errorf("Fatal error config: %s \n", err))
	}
	rowVal.Rustdesk.LoadKeyFile()
	rowVal.Admin.Init()
	if err := rowVal.Validate(); err != nil {
		panic(fmt.Errorf("fatal security configuration: %w", err))
	}
	return v
}

// Validate prevents production from silently falling back to insecure defaults.
func (c *Config) Validate() error {
	var errs []error
	if c.App.CaptchaThreshold < 0 && c.App.BanThreshold > 0 {
		errs = append(errs, errors.New("captcha-threshold cannot be negative while IP banning is enabled"))
	}
	// 自锁组合：RequiresMfaForLogin() 只看 required-for-admin，不看 enabled。
	// 二者取 true/false 时，管理员登录会被要求提供 MFA，而 MFA 功能整体关闭导致
	// 无法绑定验证器——全体管理员被永久锁在登录页。这属于配置错误，
	// 在开发模式下同样致命，因此放在 AllowInsecureDevelopment 提前返回之前。
	if c.Mfa.RequiredForAdmin && !c.Mfa.Enabled {
		errs = append(errs, errors.New("mfa.required-for-admin requires mfa.enabled=true, otherwise administrators cannot enrol and will be locked out"))
	}
	if c.Gorm.Type == TypePostgresql {
		if err := c.Postgresql.ValidateConnectionBoundary(); err != nil {
			errs = append(errs, err)
		}
	}
	if c.App.AllowInsecureDevelopment {
		return errors.Join(errs...)
	}
	if len(c.Jwt.Key) < 32 {
		errs = append(errs, errors.New("jwt.key must contain at least 32 characters"))
	}
	if c.App.TokenExpire <= 0 {
		errs = append(errs, errors.New("app.token-expire must be greater than zero"))
	}
	if c.Jwt.ExpireDuration <= 0 {
		errs = append(errs, errors.New("jwt.expire-duration must be greater than zero"))
	}
	if c.Mfa.Enabled {
		if c.Mfa.EncryptionKey == "" && len(c.Jwt.Key) < 32 {
			errs = append(errs, errors.New("mfa.encryption-key or a strong jwt.key is required when MFA is enabled"))
		}
		if c.Mfa.EncryptionKey != "" && len(c.Mfa.EncryptionKey) < 32 {
			errs = append(errs, errors.New("mfa.encryption-key must contain at least 32 characters when configured"))
		}
	}
	if c.Passkey.Enabled {
		if strings.TrimSpace(c.Passkey.RPID) == "" {
			errs = append(errs, errors.New("passkey.rp-id is required when passkey is enabled"))
		}
		if strings.TrimSpace(c.Passkey.RPDisplayName) == "" {
			errs = append(errs, errors.New("passkey.rp-display-name is required when passkey is enabled"))
		}
		if len(c.Passkey.Origins) == 0 {
			errs = append(errs, errors.New("passkey.origins must contain at least one origin when passkey is enabled"))
		} else {
			for _, origin := range c.Passkey.Origins {
				trimmed := strings.TrimSpace(origin)
				if !strings.HasPrefix(strings.ToLower(trimmed), "https://") && !(c.App.AllowInsecureDevelopment && strings.HasPrefix(strings.ToLower(trimmed), "http://")) {
					errs = append(errs, errors.New("passkey.origins must use https:// outside explicit development mode"))
					break
				}
			}
		}
	}
	if c.Cache.Type != "redis" || strings.TrimSpace(c.Cache.RedisAddr) == "" {
		errs = append(errs, errors.New("cache.type must be redis and cache.redis-addr must be configured"))
	} else {
		if strings.TrimSpace(c.Cache.RedisUsername) == "" || c.Cache.RedisPwd == "" {
			errs = append(errs, errors.New("cache.redis-username and cache.redis-pwd are required for Redis ACL authentication"))
		}
		if !c.Cache.RedisTLS {
			errs = append(errs, errors.New("cache.redis-tls must be true outside explicit development mode"))
		}
		if c.Cache.RedisDb != 0 {
			errs = append(errs, errors.New("cache.redis-db must be 0; use the key prefix and ACL pattern for isolation"))
		}
		prefix := strings.TrimSpace(c.Cache.RedisKeyPrefix)
		if prefix == "" || !strings.HasSuffix(prefix, ":") {
			errs = append(errs, errors.New("cache.redis-key-prefix must be non-empty and end with ':'"))
		}
		if _, err := c.Cache.RedisOptions(); err != nil {
			errs = append(errs, err)
		}
	}
	if c.Sms.Provider != "aliyun" {
		errs = append(errs, errors.New("sms.provider must be aliyun outside explicit development mode"))
	} else if c.Sms.AccessKeyId == "" || c.Sms.AccessKeySecret == "" || c.Sms.SignName == "" || c.Sms.TemplateCode == "" {
		errs = append(errs, errors.New("aliyun SMS credentials, sign name and template code are required"))
	}
	if c.Ldap.Enable {
		if !c.Ldap.TlsVerify {
			errs = append(errs, errors.New("ldap.tls-verify must be true when LDAP is enabled"))
		}
		if !strings.HasPrefix(strings.ToLower(c.Ldap.Url), "ldaps://") {
			errs = append(errs, errors.New("ldap.url must use ldaps:// when LDAP is enabled"))
		}
	}
	if c.Gorm.Type == TypeMysql && (c.Mysql.Tls == "" || c.Mysql.Tls == "false" || c.Mysql.Tls == "skip-verify") {
		errs = append(errs, errors.New("mysql.tls must verify the database server certificate"))
	}
	if c.Gorm.Type == TypePostgresql && c.Postgresql.Sslmode != "verify-full" {
		errs = append(errs, errors.New("postgresql.sslmode must be verify-full"))
	}
	return errors.Join(errs...)
}

// ReadEnv 读取环境变量
func ReadEnv(rowVal interface{}) *viper.Viper {
	v := viper.New()
	v.AutomaticEnv()
	if err := v.Unmarshal(rowVal); err != nil {
		fmt.Println(err)
	}
	return v
}
