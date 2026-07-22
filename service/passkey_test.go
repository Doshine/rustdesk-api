package service

import "testing"

func TestPasskeyBindingIsOpaqueAndStable(t *testing.T) {
	const bearer = "api-token-used-only-for-binding"
	first := passkeyBinding(bearer)
	second := passkeyBinding(bearer)
	if first == "" || first != second {
		t.Fatalf("passkey binding must be stable and non-empty: first=%q second=%q", first, second)
	}
	if first == bearer {
		t.Fatal("passkey challenge binding must not store the bearer token")
	}
	if first == passkeyBinding("different-token") {
		t.Fatal("different bearer tokens must not share a passkey binding")
	}
	if passkeyBinding("") != "" {
		t.Fatal("empty binding should remain empty for public login challenges")
	}
}

func TestRequestFromCredentialRejectsInvalidJSON(t *testing.T) {
	if _, err := requestFromCredential([]byte("not-json")); err != ErrPasskeyInvalidCredential {
		t.Fatalf("invalid credential JSON should be rejected: %v", err)
	}
	req, err := requestFromCredential([]byte(`{"id":"credential"}`))
	if err != nil || req == nil {
		t.Fatalf("valid credential JSON should produce an HTTP request: %v", err)
	}
}
