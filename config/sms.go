package config

type SmsConfig struct {
	Provider          string `mapstructure:"provider"`            // mock / aliyun, 为空或未识别时降级为mock
	AccessKeyId       string `mapstructure:"access-key-id"`       // 阿里云 AccessKeyId
	AccessKeySecret   string `mapstructure:"access-key-secret"`   // 阿里云 AccessKeySecret
	SignName          string `mapstructure:"sign-name"`           // 阿里云短信签名
	TemplateCode      string `mapstructure:"template-code"`       // 阿里云短信模板code, 模板参数为 {"code":"123456"}
	Endpoint          string `mapstructure:"endpoint"`            // 阿里云短信endpoint, 默认 dysmsapi.aliyuncs.com
	CodeExpireSeconds int    `mapstructure:"code-expire-seconds"` // 验证码有效期(秒), 默认300
	DailyLimit        int    `mapstructure:"daily-limit"`         // 单手机号每日发送上限, <=0不限制
	PerIpLimit        int    `mapstructure:"per-ip-limit"`        // 单IP每分钟发送上限, <=0不限制
}
