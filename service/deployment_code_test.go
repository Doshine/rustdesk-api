package service

import (
	"errors"
	"strings"
	"testing"
	"time"

	appLock "github.com/lejianwen/rustdesk-api/v2/lib/lock"
	"github.com/lejianwen/rustdesk-api/v2/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupDeploymentCodeTest(t *testing.T) *DeploymentCodeService {
	t.Helper()
	previousDB, previousLock := DB, Lock
	db, err := gorm.Open(sqlite.Open("file:deployment-code-test?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.Migrator().DropTable(&model.DeploymentAuditEvent{}, &model.DeploymentCode{}, &model.Peer{}, &model.DeviceGroup{}); err != nil {
		t.Fatalf("reset deployment test tables: %v", err)
	}
	if err := db.AutoMigrate(&model.DeviceGroup{}, &model.Peer{}, &model.DeploymentCode{}, &model.DeploymentAuditEvent{}); err != nil {
		t.Fatalf("migrate deployment test tables: %v", err)
	}
	DB = db
	Lock = appLock.NewLocal()
	t.Cleanup(func() {
		if sqlDB, closeErr := db.DB(); closeErr == nil {
			_ = sqlDB.Close()
		}
		DB, Lock = previousDB, previousLock
	})
	return &DeploymentCodeService{}
}

func TestDeploymentCodeCreateAndClaim(t *testing.T) {
	service := setupDeploymentCodeTest(t)
	group := &model.DeviceGroup{Name: "Finance workstations"}
	if err := DB.Create(group).Error; err != nil {
		t.Fatalf("create device group: %v", err)
	}
	secret, err := service.Create(DeploymentCodeCreateInput{
		Name: "Quarterly onboarding", DeviceGroupId: group.Id,
		AllowedPlatforms: []string{"Windows", "macOS"}, ExpiresAt: time.Now().Add(time.Hour).Unix(), MaxUses: 2,
	}, 7, "192.0.2.4")
	if err != nil {
		t.Fatalf("create deployment code: %v", err)
	}
	if !strings.HasPrefix(secret.Code, "YJ-") || strings.Contains(secret.DeploymentCode.CodeHint, secret.Code) {
		t.Fatalf("unexpected secret or hint: %#v", secret)
	}
	var stored model.DeploymentCode
	if err := DB.First(&stored, secret.DeploymentCode.Id).Error; err != nil {
		t.Fatalf("load deployment code: %v", err)
	}
	if stored.CodeHash == "" || stored.CodeHash == secret.Code || strings.Contains(stored.AllowedPlatforms, secret.Code) {
		t.Fatal("plaintext deployment code must never be persisted")
	}

	claim := DeploymentClaimInput{
		Code: secret.Code, DeviceId: "901234567", Uuid: "device-uuid-1", Hostname: "FIN-MBP-04",
		Os: "macOS 15", Username: "operator", Version: "1.3.9", Platform: "Darwin", Ip: "192.0.2.10",
	}
	peer, err := service.Claim(claim)
	if err != nil {
		t.Fatalf("claim deployment code: %v", err)
	}
	if peer.Status != model.PeerStatusPending || peer.GroupId != group.Id {
		t.Fatalf("claimed device must be pending in scoped group: %#v", peer)
	}
	if _, err := service.Claim(claim); err != nil {
		t.Fatalf("same pending device claim must be idempotent: %v", err)
	}
	if err := DB.First(&stored, secret.DeploymentCode.Id).Error; err != nil {
		t.Fatalf("reload deployment code: %v", err)
	}
	if stored.UsedCount != 1 {
		t.Fatalf("idempotent retry must not consume capacity, got %d", stored.UsedCount)
	}
	var auditCount int64
	if err := DB.Model(&model.DeploymentAuditEvent{}).Where("deployment_code_id = ?", stored.Id).Count(&auditCount).Error; err != nil {
		t.Fatalf("count deployment audit: %v", err)
	}
	if auditCount != 2 {
		t.Fatalf("expected create and claim audit events, got %d", auditCount)
	}
}

func TestDeploymentCodeRejectsGuardrailViolations(t *testing.T) {
	service := setupDeploymentCodeTest(t)
	if _, err := service.Create(DeploymentCodeCreateInput{
		Name: "Too broad", AllowedPlatforms: []string{"windows"}, ExpiresAt: time.Now().Add(91 * 24 * time.Hour).Unix(), MaxUses: 1,
	}, 1, ""); !errors.Is(err, ErrDeploymentCodeInvalid) {
		t.Fatalf("expiry beyond 90 days must fail, got %v", err)
	}
	secret, err := service.Create(DeploymentCodeCreateInput{
		Name: "Linux only", AllowedPlatforms: []string{"linux"}, ExpiresAt: time.Now().Add(time.Hour).Unix(), MaxUses: 1,
	}, 1, "")
	if err != nil {
		t.Fatalf("create limited deployment code: %v", err)
	}
	_, err = service.Claim(DeploymentClaimInput{
		Code: secret.Code, DeviceId: "1", Uuid: "uuid-1", Platform: "windows",
	})
	if !errors.Is(err, ErrDeploymentPlatform) {
		t.Fatalf("disallowed platform must fail, got %v", err)
	}
	if err := service.Revoke(secret.DeploymentCode.Id, 1, ""); err != nil {
		t.Fatalf("revoke deployment code: %v", err)
	}
	_, err = service.Claim(DeploymentClaimInput{
		Code: secret.Code, DeviceId: "2", Uuid: "uuid-2", Platform: "linux",
	})
	if !errors.Is(err, ErrDeploymentCodeRevoked) {
		t.Fatalf("revoked deployment code must fail, got %v", err)
	}
}
