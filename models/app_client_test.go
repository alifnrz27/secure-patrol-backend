package models

import "testing"

func TestRoleCanUsePlatform(t *testing.T) {
	allRoles := []string{RoleSuperAdmin, RoleSecurityManager, RoleSecurityHead, RoleSecurityAdmin, RoleSecurityTeam, "custom_role"}

	for _, role := range allRoles {
		for _, platform := range []string{PlatformAndroid, PlatformIOS} {
			if !RoleCanUsePlatform(role, platform) {
				t.Errorf("%s must be allowed on %s", role, platform)
			}
		}
	}

	for _, platform := range []string{PlatformWeb, PlatformServer} {
		for _, role := range []string{RoleSuperAdmin, RoleSecurityManager, RoleSecurityHead, RoleSecurityAdmin} {
			if !RoleCanUsePlatform(role, platform) {
				t.Errorf("%s must be allowed on %s", role, platform)
			}
		}
		for _, role := range []string{RoleSecurityTeam, "custom_role", ""} {
			if RoleCanUsePlatform(role, platform) {
				t.Errorf("%q must be denied on %s", role, platform)
			}
		}
	}

	if RoleCanUsePlatform(RoleSuperAdmin, "desktop") || RoleCanUsePlatform(RoleSuperAdmin, "") {
		t.Error("unknown platforms must be denied")
	}
}
