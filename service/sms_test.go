package service

import (
	"errors"
	"regexp"
	"testing"

	"github.com/lejianwen/rustdesk-api/v2/config"
	"github.com/lejianwen/rustdesk-api/v2/lib/cache"
	log "github.com/sirupsen/logrus"
)

// fakeSender 记录发送的验证码, 用于测试
type fakeSender struct {
	codes map[string]string
	err   error
}

func (f *fakeSender) SendCode(phone, code string) error {
	if f.err != nil {
		return f.err
	}
	f.codes[phone] = code
	return nil
}

func setupSmsTest(cfg config.SmsConfig) (*SmsService, *fakeSender) {
	Config = &config.Config{}
	Config.Sms = cfg
	Logger = log.New()
	fs := &fakeSender{codes: make(map[string]string)}
	return NewSmsService(fs, cache.NewSimpleCache()), fs
}

func TestGenerateSmsCode(t *testing.T) {
	re := regexp.MustCompile(`^\d{6}$`)
	for i := 0; i < 100; i++ {
		code := generateSmsCode()
		if !re.MatchString(code) {
			t.Fatalf("code %s is not 6 digits", code)
		}
	}
}

func TestIsValidCnPhone(t *testing.T) {
	valid := []string{"13812345678", "19998765432", "15000000000", "18611112222"}
	for _, p := range valid {
		if !IsValidCnPhone(p) {
			t.Fatalf("phone %s should be valid", p)
		}
	}
	invalid := []string{"", "12345678901", "1381234567", "138123456789", "1381234567a", "23812345678"}
	for _, p := range invalid {
		if IsValidCnPhone(p) {
			t.Fatalf("phone %s should be invalid", p)
		}
	}
}

func TestMaskCnPhone(t *testing.T) {
	if got := MaskCnPhone("13812345678"); got != "138****5678" {
		t.Fatalf("mask got %s", got)
	}
	if got := MaskCnPhone("123"); got != "123" {
		t.Fatalf("short phone should be unchanged, got %s", got)
	}
}

// TestSendAndVerifyLoginCode 验证码生成/发送/校验生命周期
func TestSendAndVerifyLoginCode(t *testing.T) {
	ss, fs := setupSmsTest(config.SmsConfig{})
	phone, ip := "13812345678", "127.0.0.1"

	if err := ss.SendLoginCode(phone, ip); err != nil {
		t.Fatalf("SendLoginCode err: %v", err)
	}
	code := fs.codes[phone]
	if code == "" {
		t.Fatal("sender did not receive code")
	}
	// 错误验证码
	if ss.VerifyLoginCode(phone, "000000") && code != "000000" {
		t.Fatal("verify wrong code should fail")
	}
	// 上面的错误校验已触发一次即焚, 重新发送
	if err := ss.SendLoginCode(phone, ip); err != nil {
		t.Fatalf("SendLoginCode err: %v", err)
	}
	code = fs.codes[phone]
	if !ss.VerifyLoginCode(phone, code) {
		t.Fatal("verify correct code should pass")
	}
	// 已焚毁, 再次校验失败
	if ss.VerifyLoginCode(phone, code) {
		t.Fatal("code should be burned after verify")
	}
}

// TestVerifyLoginCodeBurnOnWrong 校验错误也焚毁验证码
func TestVerifyLoginCodeBurnOnWrong(t *testing.T) {
	ss, fs := setupSmsTest(config.SmsConfig{})
	phone, ip := "13812345678", "127.0.0.1"

	if err := ss.SendLoginCode(phone, ip); err != nil {
		t.Fatalf("SendLoginCode err: %v", err)
	}
	code := fs.codes[phone]
	wrong := "000000"
	for wrong == code {
		wrong = "111111"
	}
	if ss.VerifyLoginCode(phone, wrong) {
		t.Fatal("verify wrong code should fail")
	}
	if ss.VerifyLoginCode(phone, code) {
		t.Fatal("code should be burned even after wrong verify")
	}
}

// TestSendLoginCodeDailyLimit 单手机号日限
func TestSendLoginCodeDailyLimit(t *testing.T) {
	ss, _ := setupSmsTest(config.SmsConfig{DailyLimit: 2})
	phone, ip := "13812345678", "127.0.0.1"

	for i := 0; i < 2; i++ {
		if err := ss.SendLoginCode(phone, ip); err != nil {
			t.Fatalf("send %d err: %v", i, err)
		}
	}
	if err := ss.SendLoginCode(phone, ip); err == nil || err.Error() != "SmsDailyLimitExceeded" {
		t.Fatalf("expect SmsDailyLimitExceeded, got %v", err)
	}
	// 其他手机号不受限
	if err := ss.SendLoginCode("13987654321", ip); err != nil {
		t.Fatalf("other phone should not be limited, got %v", err)
	}
}

// TestSendLoginCodeIpLimit 单IP分钟限
func TestSendLoginCodeIpLimit(t *testing.T) {
	ss, _ := setupSmsTest(config.SmsConfig{PerIpLimit: 1})
	phone, ip := "13812345678", "127.0.0.1"

	if err := ss.SendLoginCode(phone, ip); err != nil {
		t.Fatalf("send err: %v", err)
	}
	if err := ss.SendLoginCode(phone, ip); err == nil || err.Error() != "SmsIpLimitExceeded" {
		t.Fatalf("expect SmsIpLimitExceeded, got %v", err)
	}
	// 其他IP不受限
	if err := ss.SendLoginCode(phone, "127.0.0.2"); err != nil {
		t.Fatalf("other ip should not be limited, got %v", err)
	}
}

// TestSendLoginCodeInvalidPhone 手机号格式校验
func TestSendLoginCodeInvalidPhone(t *testing.T) {
	ss, _ := setupSmsTest(config.SmsConfig{})
	if err := ss.SendLoginCode("123", "127.0.0.1"); err == nil || err.Error() != "PhoneFormatError" {
		t.Fatalf("expect PhoneFormatError, got %v", err)
	}
}

// TestSendLoginCodeSendFailed 发送失败不消耗限流配额且验证码作废
func TestSendLoginCodeSendFailed(t *testing.T) {
	ss, fs := setupSmsTest(config.SmsConfig{DailyLimit: 1})
	phone, ip := "13812345678", "127.0.0.1"

	fs.err = errors.New("mock send error")
	if err := ss.SendLoginCode(phone, ip); err == nil || err.Error() != "SmsSendFailed" {
		t.Fatalf("expect SmsSendFailed, got %v", err)
	}
	// 验证码已作废
	if ss.VerifyLoginCode(phone, "123456") {
		t.Fatal("code should be burned after send failed")
	}
	// 未消耗日限配额, 恢复后可正常发送
	fs.err = nil
	if err := ss.SendLoginCode(phone, ip); err != nil {
		t.Fatalf("send after failure should pass, got %v", err)
	}
	if !ss.VerifyLoginCode(phone, fs.codes[phone]) {
		t.Fatal("verify correct code should pass")
	}
}
