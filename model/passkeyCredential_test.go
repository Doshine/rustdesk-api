package model

import "testing"

func TestPasskeyCredentialViewRedactsCredentialID(t *testing.T) {
	credential := &PasskeyCredential{CredentialId: "abcdefghijklmnop", Name: "MacBook"}
	view := credential.View()
	if view == nil || view.CredentialIdHint != "abcd...mnop" {
		t.Fatalf("credential view must expose only an ID hint: %#v", view)
	}
}
