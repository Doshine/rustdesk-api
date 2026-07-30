package config

// MetricsConfig 控制 /metrics 端点。
//
// 默认关闭：指标里包含用户数、设备数、在线数等经营信息，不应无条件暴露。
// 启用时必须同时配置 token——与项目其余部分的 fail-closed 风格一致，
// 宁可运维多配一项，也不要出现一个「以为没开、实际裸奔」的端点。
type MetricsConfig struct {
	Enabled bool   `mapstructure:"enabled"`
	// Token 走 Bearer 校验。建议经 RUSTDESK_API_METRICS_TOKEN_FILE 注入。
	Token string `mapstructure:"token"`
}
