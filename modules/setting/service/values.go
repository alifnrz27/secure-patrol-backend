package service

import (
	"strconv"
	"time"
)

// Values are the effective settings, typed for use in code.
type Values struct {
	PatrolLocationRadiusMeters float64
	PatrolMaxOfflineHours      int
	FaceMobileAccuracy         float64
	FaceMatchMinScore          float64
	FacePhotoValidation        bool
	FaceMinSizeRatio           float64
	FaceMaxTiltDegrees         float64
	FaceMaxTurnRatio           float64
	LoginMaxFailedAttempts     int
	LoginLockMinutes           int
	AccessTokenTTLMinutes      int
	RefreshTokenTTLDays        int

	// Raw holds the canonical text value of every key, defaults included.
	Raw map[string]string
}

func (v Values) PatrolMaxOfflineAge() time.Duration {
	return time.Duration(v.PatrolMaxOfflineHours) * time.Hour
}

func (v Values) LoginLockDuration() time.Duration {
	return time.Duration(v.LoginLockMinutes) * time.Minute
}

func (v Values) AccessTokenTTL() time.Duration {
	return time.Duration(v.AccessTokenTTLMinutes) * time.Minute
}

func (v Values) RefreshTokenTTL() time.Duration {
	return time.Duration(v.RefreshTokenTTLDays) * 24 * time.Hour
}

// PublicMap returns the settings sent to the apps (see Definition.Public).
func (v Values) PublicMap() map[string]interface{} {
	result := map[string]interface{}{}
	for _, def := range Definitions {
		if def.Public {
			result[def.Key] = def.Typed(v.Raw[def.Key])
		}
	}
	return result
}

// Map returns every setting with its JSON type, keyed by setting key.
func (v Values) Map() map[string]interface{} {
	result := make(map[string]interface{}, len(Definitions))
	for _, def := range Definitions {
		result[def.Key] = def.Typed(v.Raw[def.Key])
	}
	return result
}

// buildValues applies the stored layers over the defaults, in order (e.g. the
// global values, then a unit's overrides). A stored value that is no longer
// valid (e.g. edited directly in the database) is ignored.
func buildValues(layers []map[string]string, warn func(key, value string, err error)) Values {
	raw := make(map[string]string, len(Definitions))
	for _, def := range Definitions {
		raw[def.Key] = def.Default
		for _, stored := range layers {
			value, ok := stored[def.Key]
			if !ok {
				continue
			}
			normalized, err := def.normalize(value)
			if err != nil {
				if warn != nil {
					warn(def.Key, value, err)
				}
				continue
			}
			raw[def.Key] = normalized
		}
	}

	number := func(key string) float64 { v, _ := strconv.ParseFloat(raw[key], 64); return v }
	integer := func(key string) int { v, _ := strconv.Atoi(raw[key]); return v }
	boolean := func(key string) bool { v, _ := strconv.ParseBool(raw[key]); return v }

	return Values{
		PatrolLocationRadiusMeters: number("patrol_location_radius_meters"),
		PatrolMaxOfflineHours:      integer("patrol_max_offline_hours"),
		FaceMobileAccuracy:         number("face_mobile_accuracy"),
		FaceMatchMinScore:          number("face_match_min_score"),
		FacePhotoValidation:        boolean("face_photo_validation"),
		FaceMinSizeRatio:           number("face_min_size_ratio"),
		FaceMaxTiltDegrees:         number("face_max_tilt_degrees"),
		FaceMaxTurnRatio:           number("face_max_turn_ratio"),
		LoginMaxFailedAttempts:     integer("login_max_failed_attempts"),
		LoginLockMinutes:           integer("login_lock_minutes"),
		AccessTokenTTLMinutes:      integer("access_token_ttl_minutes"),
		RefreshTokenTTLDays:        integer("refresh_token_ttl_days"),
		Raw:                        raw,
	}
}

// DefaultValues are the built-in defaults, used before the database is reachable.
func DefaultValues() Values {
	return buildValues(nil, nil)
}
