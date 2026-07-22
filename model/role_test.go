package model

import "testing"

func TestRolePermissionMatrix(t *testing.T) {
	checks := []struct {
		role       string
		permission string
		want       bool
	}{
		{RoleAuditor, PermissionAuditRead, true},
		{RoleAuditor, PermissionDeviceManage, false},
		{RoleOperator, PermissionSessionRevoke, true},
		{RoleUser, PermissionAuditRead, false},
		{RoleOwner, PermissionDeviceManage, true},
	}
	for _, check := range checks {
		if got := RoleHasPermission(check.role, check.permission); got != check.want {
			t.Errorf("RoleHasPermission(%q, %q) = %v, want %v", check.role, check.permission, got, check.want)
		}
	}
}

func TestRoleValidation(t *testing.T) {
	if !IsValidRole(RoleAuditor) || IsValidRole("root") {
		t.Fatal("role validation matrix is incorrect")
	}
}
