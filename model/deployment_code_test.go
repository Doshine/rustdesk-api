package model

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestDeploymentCodeRedactsHashAndAuditIsAppendOnly(t *testing.T) {
	code := &DeploymentCode{CodeHash: strings.Repeat("a", 64), CodeHint: "YJ-ABCD-••••-WXYZ"}
	encoded, err := json.Marshal(code)
	if err != nil {
		t.Fatalf("marshal deployment code: %v", err)
	}
	if strings.Contains(string(encoded), code.CodeHash) || strings.Contains(string(encoded), "code_hash") {
		t.Fatalf("deployment code hash must not be serialized: %s", encoded)
	}

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&DeploymentAuditEvent{}); err != nil {
		t.Fatalf("migrate deployment audit: %v", err)
	}
	event := &DeploymentAuditEvent{Action: DeploymentAuditCreated, DeploymentCodeId: 1}
	if err := db.Create(event).Error; err != nil {
		t.Fatalf("create deployment audit: %v", err)
	}
	if err := db.Model(event).Update("detail", "mutated").Error; !errors.Is(err, ErrDeploymentAuditAppendOnly) {
		t.Fatalf("update must be blocked by append-only hook, got %v", err)
	}
	if err := db.Delete(event).Error; !errors.Is(err, ErrDeploymentAuditAppendOnly) {
		t.Fatalf("delete must be blocked by append-only hook, got %v", err)
	}
}
