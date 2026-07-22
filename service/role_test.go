package service

import (
	"testing"

	"github.com/lejianwen/rustdesk-api/v2/model"
)

func TestHasPermissionKeepsAdminCompatibility(t *testing.T) {
	isAdmin := true
	admin := &model.User{IsAdmin: &isAdmin, Status: model.COMMON_STATUS_ENABLE}
	if !(&UserService{}).HasPermission(admin, model.PermissionAuditRead) {
		t.Fatal("legacy administrators must retain permissions")
	}
}

func TestHasPermissionUsesLeastPrivilegeRole(t *testing.T) {
	isAdmin := false
	auditor := &model.User{IsAdmin: &isAdmin, Role: model.RoleAuditor, Status: model.COMMON_STATUS_ENABLE}
	service := &UserService{}
	if !service.HasPermission(auditor, model.PermissionAuditRead) {
		t.Fatal("auditor should read audit records")
	}
	if service.HasPermission(auditor, model.PermissionDeviceManage) {
		t.Fatal("auditor must not manage devices")
	}
}
