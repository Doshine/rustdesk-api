package sms

import (
	log "github.com/sirupsen/logrus"
)

// 短信验证码发送渠道
const (
	ProviderMock   = "mock"
	ProviderAliyun = "aliyun"
)

// DefaultAliyunEndpoint 阿里云短信默认endpoint
const DefaultAliyunEndpoint = "dysmsapi.aliyuncs.com"

// Sender 短信验证码发送接口
type Sender interface {
	SendCode(phone, code string) error
}

// Config 短信配置
type Config struct {
	Provider        string // mock / aliyun, 为空或未识别时降级为mock
	AccessKeyId     string
	AccessKeySecret string
	SignName        string
	TemplateCode    string
	Endpoint        string // 阿里云短信endpoint, 为空时使用默认endpoint
}

// NewSender 根据配置创建Sender, 配置缺失或初始化失败时降级为mock并打warn日志
func NewSender(cfg *Config, logger *log.Logger) Sender {
	if logger == nil {
		logger = log.StandardLogger()
	}
	if cfg == nil {
		logger.Warn("sms config is nil, fallback to mock sender")
		return &MockSender{Logger: logger}
	}
	switch cfg.Provider {
	case ProviderAliyun:
		sender, err := NewAliyunSender(cfg)
		if err != nil {
			logger.Warn("init aliyun sms sender failed, fallback to mock sender: ", err)
			return &MockSender{Logger: logger}
		}
		return sender
	case ProviderMock, "":
		return &MockSender{Logger: logger}
	default:
		logger.Warn("unknown sms provider ", cfg.Provider, ", fallback to mock sender")
		return &MockSender{Logger: logger}
	}
}
