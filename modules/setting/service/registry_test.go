package service

import (
	"sort"
	"testing"
)

func TestDefaultsAreValid(t *testing.T) {
	seen := map[string]bool{}
	for _, def := range Definitions {
		if seen[def.Key] {
			t.Errorf("duplicate setting %s", def.Key)
		}
		seen[def.Key] = true
		if _, err := def.normalize(def.Default); err != nil {
			t.Errorf("%s: default %q is invalid: %v", def.Key, def.Default, err)
		}
	}
}

func TestDefaultValues(t *testing.T) {
	v := DefaultValues()
	if v.PatrolLocationRadiusMeters != 100 || v.PatrolMaxOfflineHours != 24 || v.FaceMobileAccuracy != 0.75 ||
		v.FaceMatchMinScore != 0 || !v.FacePhotoValidation || v.FaceMinSizeRatio != 0.2 || v.FaceMaxTiltDegrees != 20 ||
		v.FaceMaxTurnRatio != 0.12 || v.LoginMaxFailedAttempts != 3 || v.LoginLockMinutes != 5 ||
		v.AccessTokenTTLMinutes != 60 || v.RefreshTokenTTLDays != 30 {
		t.Fatalf("unexpected defaults: %+v", v)
	}
}

func TestInvalidStoredValueFallsBackToDefault(t *testing.T) {
	var warned []string
	v := buildValues([]map[string]string{{
		"patrol_location_radius_meters": "0",     // below minimum
		"login_max_failed_attempts":     "abc",   // not a number
		"face_photo_validation":         "maybe", // not a boolean
		"access_token_ttl_minutes":      "15",    // valid
	}}, func(key, value string, err error) { warned = append(warned, key) })

	if v.PatrolLocationRadiusMeters != 100 || v.LoginMaxFailedAttempts != 3 || !v.FacePhotoValidation {
		t.Errorf("invalid values must fall back to defaults: %+v", v)
	}
	if v.AccessTokenTTLMinutes != 15 {
		t.Errorf("valid stored value must be used, got %d", v.AccessTokenTTLMinutes)
	}
	sort.Strings(warned)
	if len(warned) != 3 {
		t.Errorf("expected 3 warnings, got %v", warned)
	}
}

func TestNormalize(t *testing.T) {
	radius, _ := definition("patrol_location_radius_meters")
	attempts, _ := definition("login_max_failed_attempts")
	validation, _ := definition("face_photo_validation")

	valid := []struct {
		def  Definition
		in   string
		want string
	}{
		{radius, "150", "150"}, {radius, " 12.50 ", "12.5"}, {radius, "10000", "10000"},
		{attempts, "5", "5"}, {validation, "false", "false"}, {validation, "1", "true"},
	}
	for _, c := range valid {
		if got, err := c.def.normalize(c.in); err != nil || got != c.want {
			t.Errorf("%s %q: got %q, %v; want %q", c.def.Key, c.in, got, err, c.want)
		}
	}

	invalid := []struct {
		def Definition
		in  string
	}{
		{radius, "0"}, {radius, "-5"}, {radius, "10001"}, {radius, "abc"},
		{attempts, "2.5"}, {attempts, "0"}, {attempts, "21"}, {validation, "yes please"},
	}
	for _, c := range invalid {
		if _, err := c.def.normalize(c.in); err == nil {
			t.Errorf("%s %q must be rejected", c.def.Key, c.in)
		}
	}
}

func TestPublicMapHasOnlyAppSettings(t *testing.T) {
	var keys []string
	for key := range DefaultValues().PublicMap() {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	want := []string{
		"access_token_ttl_minutes", "face_mobile_accuracy", "login_lock_minutes", "login_max_failed_attempts",
		"patrol_location_radius_meters", "patrol_max_offline_hours", "refresh_token_ttl_days",
	}
	if len(keys) != len(want) {
		t.Fatalf("got %v, want %v", keys, want)
	}
	for i := range want {
		if keys[i] != want[i] {
			t.Fatalf("got %v, want %v", keys, want)
		}
	}
}

func TestUnitOverridesGlobal(t *testing.T) {
	global := map[string]string{"patrol_location_radius_meters": "150", "login_lock_minutes": "10"}
	unit := map[string]string{"patrol_location_radius_meters": "50", "login_lock_minutes": "oops"}

	v := buildValues([]map[string]string{global, unit}, nil)
	if v.PatrolLocationRadiusMeters != 50 {
		t.Errorf("unit override must win, got %v", v.PatrolLocationRadiusMeters)
	}
	if v.LoginLockMinutes != 10 {
		t.Errorf("invalid unit value must fall back to the global value, got %v", v.LoginLockMinutes)
	}
	if v.LoginMaxFailedAttempts != 3 {
		t.Errorf("unset everywhere must use the code default, got %v", v.LoginMaxFailedAttempts)
	}
}
