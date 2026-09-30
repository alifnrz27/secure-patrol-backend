package models

import "testing"

func TestRoleCanUsePlatform(t *testing.T) {
	for _, platform := range []string{PlatformAndroid, PlatformIOS} {
		for _, role := range []string{RoleSecurityHead, RoleSecurityAdmin, RoleSecurityTeam, "custom_role"} {
			if !RoleCanUsePlatform(role, platform) {
				t.Errorf("%s must be allowed on %s", role, platform)
			}
		}
		for _, role := range []string{RoleSuperAdmin, RoleSecurityManager} {
			if RoleCanUsePlatform(role, platform) {
				t.Errorf("head office role %s must be denied on %s", role, platform)
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
