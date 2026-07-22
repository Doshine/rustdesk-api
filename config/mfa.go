package config

type MfaConfig struct {
	Enabled          bool   `mapstructure:"enabled"`
	Issuer           string `mapstructure:"issuer"`
	EncryptionKey    string `mapstructure:"encryption-key"`
	RequiredForAdmin bool   `mapstructure:"required-for-admin"`
}
