package service

import "testing"

func TestTokenHintDoesNotExposeBearerCredential(t *testing.T) {
	token := "eyJhbGciOiJIUzI1NiJ9.secret.payload"
	hint := tokenHint(token)
	if hint != "••••load" {
		t.Fatalf("unexpected token hint: %s", hint)
	}
	if hint == token {
		t.Fatal("token hint must not equal the bearer token")
	}
}

func TestTokenHintForEmptyToken(t *testing.T) {
	if got := tokenHint(""); got != "" {
		t.Fatalf("empty token should have empty hint, got %q", got)
	}
}
