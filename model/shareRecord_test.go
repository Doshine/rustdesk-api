package model

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestShareRecordDoesNotExposePassword(t *testing.T) {
	payload, err := json.Marshal(&ShareRecord{
		PeerId:       "123456789",
		ShareToken:   "temporary-token",
		PasswordType: "once",
		Password:     "sensitive-code",
		Expire:       1800,
	})
	if err != nil {
		t.Fatalf("marshal share record: %v", err)
	}
	if strings.Contains(string(payload), "sensitive-code") || strings.Contains(string(payload), "\"password\"") {
		t.Fatalf("share record JSON exposed the temporary password: %s", payload)
	}
}
