package sms

import (
	log "github.com/sirupsen/logrus"
)

// MockSender 仅将验证码打印到日志, 用于开发联调
type MockSender struct {
	Logger *log.Logger
}

// SendCode 打印验证码到日志
func (s *MockSender) SendCode(phone, code string) error {
	logger := s.Logger
	if logger == nil {
		logger = log.StandardLogger()
	}
	logger.Info("[MockSms] send code ", code, " to ", phone)
	return nil
}
