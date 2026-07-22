package sms

import (
	"errors"
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

// NewSender fails closed. Mock delivery must be selected explicitly for development.
func NewSender(cfg *Config, logger *log.Logger) (Sender, error) {
	if logger == nil {
		logger = log.StandardLogger()
	}
	if cfg == nil {
		return nil, errors.New("sms config is nil")
	}
	switch cfg.Provider {
	case ProviderAliyun:
		sender, err := NewAliyunSender(cfg)
		if err != nil {
			return nil, err
		}
		return sender, nil
	case ProviderMock:
		return &MockSender{Logger: logger}, nil
	case "":
		return nil, errors.New("sms provider is empty")
	default:
		return nil, errors.New("unknown sms provider: " + cfg.Provider)
	}
}
