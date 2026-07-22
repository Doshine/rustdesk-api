package service

import (
	"testing"
	"time"

	"github.com/lejianwen/rustdesk-api/v2/config"
)

func TestVerifyTotpRfc6238Vector(t *testing.T) {
	// RFC 6238 Appendix B uses the base32 secret "12345678901234567890".
	secret := "GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ"
	if !VerifyTotp(secret, "94287082"[2:], time.Unix(59, 0)) {
		t.Fatal("RFC 6238 TOTP vector should verify")
	}
	if VerifyTotp(secret, "000000", time.Unix(59, 0)) {
		t.Fatal("invalid TOTP code must be rejected")
	}
}

func TestMfaValueEncryptionRoundTrip(t *testing.T) {
	original := Config
	if Config == nil {
		Config = &config.Config{}
	}
	Config.Jwt.Key = "0123456789abcdef0123456789abcdef"
	Config.Mfa.EncryptionKey = ""
	defer func() { Config = original }()

	ciphertext, err := encryptMfaValue("do-not-log-this")
	if err != nil {
		t.Fatalf("encrypt MFA value: %v", err)
	}
	plaintext, err := decryptMfaValue(ciphertext)
	if err != nil || plaintext != "do-not-log-this" {
		t.Fatalf("decrypt MFA value: plaintext=%q err=%v", plaintext, err)
	}
	if ciphertext == plaintext {
		t.Fatal("MFA secret must not be stored as plaintext")
	}
}
