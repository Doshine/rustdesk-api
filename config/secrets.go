package config

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/viper"
)

const maxSecretFileBytes = 64 * 1024

type secretFileBinding struct {
	key       string
	valueEnv  string
	secretEnv string
}

var secretFileBindings = []secretFileBinding{
	{key: "jwt.key", valueEnv: "RUSTDESK_API_JWT_KEY", secretEnv: "RUSTDESK_API_JWT_KEY_FILE"},
	{key: "mfa.encryption-key", valueEnv: "RUSTDESK_API_MFA_ENCRYPTION_KEY", secretEnv: "RUSTDESK_API_MFA_ENCRYPTION_KEY_FILE"},
	{key: "postgresql.password", valueEnv: "RUSTDESK_API_POSTGRESQL_PASSWORD", secretEnv: "RUSTDESK_API_POSTGRESQL_PASSWORD_FILE"},
	{key: "mysql.password", valueEnv: "RUSTDESK_API_MYSQL_PASSWORD", secretEnv: "RUSTDESK_API_MYSQL_PASSWORD_FILE"},
	{key: "cache.redis-pwd", valueEnv: "RUSTDESK_API_CACHE_REDIS_PWD", secretEnv: "RUSTDESK_API_CACHE_REDIS_PWD_FILE"},
	{key: "sms.access-key-id", valueEnv: "RUSTDESK_API_SMS_ACCESS_KEY_ID", secretEnv: "RUSTDESK_API_SMS_ACCESS_KEY_ID_FILE"},
	{key: "sms.access-key-secret", valueEnv: "RUSTDESK_API_SMS_ACCESS_KEY_SECRET", secretEnv: "RUSTDESK_API_SMS_ACCESS_KEY_SECRET_FILE"},
	{key: "ldap.bind-password", valueEnv: "RUSTDESK_API_LDAP_BIND_PASSWORD", secretEnv: "RUSTDESK_API_LDAP_BIND_PASSWORD_FILE"},
	{key: "oss.access-key-id", valueEnv: "RUSTDESK_API_OSS_ACCESS_KEY_ID", secretEnv: "RUSTDESK_API_OSS_ACCESS_KEY_ID_FILE"},
	{key: "oss.access-key-secret", valueEnv: "RUSTDESK_API_OSS_ACCESS_KEY_SECRET", secretEnv: "RUSTDESK_API_OSS_ACCESS_KEY_SECRET_FILE"},
	{key: "metrics.token", valueEnv: "RUSTDESK_API_METRICS_TOKEN", secretEnv: "RUSTDESK_API_METRICS_TOKEN_FILE"},
}

// applySecretFileOverrides supports Docker/Kubernetes/Rainbond mounted secrets
// without placing secret values in YAML or process arguments. A direct value
// environment variable and its matching _FILE variable are mutually exclusive
// so deployment precedence cannot become ambiguous.
func applySecretFileOverrides(v *viper.Viper) error {
	for _, binding := range secretFileBindings {
		path, ok := os.LookupEnv(binding.secretEnv)
		if !ok {
			continue
		}
		if _, directSet := os.LookupEnv(binding.valueEnv); directSet {
			return fmt.Errorf("%s and %s cannot both be set", binding.valueEnv, binding.secretEnv)
		}
		value, err := readSecretFile(path)
		if err != nil {
			return fmt.Errorf("%s: %w", binding.secretEnv, err)
		}
		v.Set(binding.key, value)
	}
	return nil
}

func readSecretFile(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", errors.New("secret file path is empty")
	}
	info, err := os.Stat(path)
	if err != nil {
		return "", fmt.Errorf("stat secret file: %w", err)
	}
	if !info.Mode().IsRegular() {
		return "", errors.New("secret path is not a regular file")
	}
	if info.Size() > maxSecretFileBytes {
		return "", errors.New("secret file exceeds 64 KiB")
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read secret file: %w", err)
	}
	value := strings.TrimRight(string(b), "\r\n")
	if value == "" {
		return "", errors.New("secret file is empty")
	}
	if strings.IndexByte(value, 0) >= 0 {
		return "", errors.New("secret file contains a NUL byte")
	}
	return value, nil
}
