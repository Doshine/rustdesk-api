package sms

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"os"
	"strings"
	"testing"
)

// TestAliyunSandboxDelivery is an explicit, billable external-provider gate.
// It sends exactly one SMS and never prints the phone number, code, request
// payload, access key, or provider response body.
func TestAliyunSandboxDelivery(t *testing.T) {
	if os.Getenv("YINHE_SMS_SANDBOX_CONFIRM_SEND") != "1" {
		t.Skip("set YINHE_SMS_SANDBOX_CONFIRM_SEND=1 only for an approved one-message sandbox test")
	}
	accessKeyID := readSandboxSecret(t, "YINHE_SMS_SANDBOX_ACCESS_KEY_ID_FILE")
	accessKeySecret := readSandboxSecret(t, "YINHE_SMS_SANDBOX_ACCESS_KEY_SECRET_FILE")
	phone := strings.TrimSpace(os.Getenv("YINHE_SMS_SANDBOX_PHONE"))
	signName := strings.TrimSpace(os.Getenv("YINHE_SMS_SANDBOX_SIGN_NAME"))
	templateCode := strings.TrimSpace(os.Getenv("YINHE_SMS_SANDBOX_TEMPLATE_CODE"))
	if phone == "" || signName == "" || templateCode == "" {
		t.Fatal("sandbox phone, sign name, and template code must be configured")
	}

	n, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		t.Fatal("generate sandbox verification code")
	}
	sender, err := NewAliyunSender(&Config{
		Provider: ProviderAliyun, AccessKeyId: accessKeyID, AccessKeySecret: accessKeySecret,
		SignName: signName, TemplateCode: templateCode, Endpoint: strings.TrimSpace(os.Getenv("YINHE_SMS_SANDBOX_ENDPOINT")),
	})
	if err != nil {
		t.Fatal("initialize Aliyun sandbox sender")
	}
	if err := sender.SendCode(phone, fmt.Sprintf("%06d", n.Int64())); err != nil {
		t.Fatal("Aliyun sandbox delivery failed")
	}
}

func readSandboxSecret(t *testing.T, envName string) string {
	t.Helper()
	path := strings.TrimSpace(os.Getenv(envName))
	if path == "" {
		t.Fatalf("%s must point to a mounted secret file", envName)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read mounted sandbox secret for %s", envName)
	}
	value := strings.TrimRight(string(b), "\r\n")
	if value == "" {
		t.Fatalf("mounted sandbox secret for %s is empty", envName)
	}
	return value
}
