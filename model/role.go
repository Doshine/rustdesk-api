package model

const (
	RoleOwner    = "owner"
	RoleAdmin    = "admin"
	RoleAuditor  = "auditor"
	RoleOperator = "operator"
	RoleUser     = "user"
)

const (
	PermissionAuditRead    = "audit:read"
	PermissionSessionRevoke = "session:revoke"
	PermissionDeviceManage = "device:manage"
)

var validRoles = map[string]struct{}{
	RoleOwner:    {},
	RoleAdmin:    {},
	RoleAuditor:  {},
	RoleOperator: {},
	RoleUser:     {},
}

var rolePermissions = map[string]map[string]struct{}{
	RoleOwner: {
		PermissionAuditRead: {}, PermissionSessionRevoke: {}, PermissionDeviceManage: {},
	},
	RoleAdmin: {
		PermissionAuditRead: {}, PermissionSessionRevoke: {}, PermissionDeviceManage: {},
	},
	RoleAuditor: {
		PermissionAuditRead: {},
	},
	RoleOperator: {
		PermissionAuditRead: {}, PermissionSessionRevoke: {}, PermissionDeviceManage: {},
	},
	RoleUser: {},
}

func IsValidRole(role string) bool {
	_, ok := validRoles[role]
	return ok
}

func RoleHasPermission(role, permission string) bool {
	permissions, ok := rolePermissions[role]
	if !ok {
		return false
	}
	_, ok = permissions[permission]
	return ok
}
