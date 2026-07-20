package sms

import (
	"errors"

	"github.com/aliyun/alibaba-cloud-sdk-go/services/dysmsapi"
)

// AliyunSender 阿里云短信发送
type AliyunSender struct {
	client       *dysmsapi.Client
	signName     string
	templateCode string
	endpoint     string
}

// NewAliyunSender 创建阿里云短信Sender
func NewAliyunSender(cfg *Config) (*AliyunSender, error) {
	if cfg.AccessKeyId == "" || cfg.AccessKeySecret == "" {
		return nil, errors.New("aliyun sms access-key-id/access-key-secret is empty")
	}
	if cfg.SignName == "" || cfg.TemplateCode == "" {
		return nil, errors.New("aliyun sms sign-name/template-code is empty")
	}
	endpoint := cfg.Endpoint
	if endpoint == "" {
		endpoint = DefaultAliyunEndpoint
	}
	client, err := dysmsapi.NewClientWithAccessKey("cn-hangzhou", cfg.AccessKeyId, cfg.AccessKeySecret)
	if err != nil {
		return nil, err
	}
	return &AliyunSender{
		client:       client,
		signName:     cfg.SignName,
		templateCode: cfg.TemplateCode,
		endpoint:     endpoint,
	}, nil
}

// SendCode 发送短信验证码, 模板参数为 {"code":"123456"}
func (s *AliyunSender) SendCode(phone, code string) error {
	request := dysmsapi.CreateSendSmsRequest()
	request.Scheme = "https"
	request.Domain = s.endpoint
	request.PhoneNumbers = phone
	request.SignName = s.signName
	request.TemplateCode = s.templateCode
	request.TemplateParam = `{"code":"` + code + `"}`
	response, err := s.client.SendSms(request)
	if err != nil {
		return err
	}
	if response.Code != "OK" {
		return errors.New("aliyun sms send failed: " + response.Code + " " + response.Message)
	}
	return nil
}
