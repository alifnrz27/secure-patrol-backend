package service

import (
	"fmt"
	"strconv"
	"strings"
)

// Setting value types.
const (
	TypeInteger = "integer"
	TypeNumber  = "number"
	TypeBoolean = "boolean"
)

// Definition describes one setting: its type, default and allowed range.
type Definition struct {
	Key         string
	Group       string
	Type        string
	Default     string
	Min         float64
	Max         float64
	Unit        string
	Description string
	// Public settings are returned to the apps on login, refresh and profile.
	Public bool
}

// Definitions lists every adjustable setting. Missing rows are created with
// their default at startup, and invalid stored values fall back to the default.
var Definitions = []Definition{
	{Key: "patrol_location_radius_meters", Public: true, Group: "patrol", Type: TypeNumber, Default: "100", Min: 1, Max: 10000, Unit: "meters",
		Description: "Maximum distance between the phone and a patrol point that requires a location match"},
	{Key: "patrol_max_offline_hours", Public: true, Group: "patrol", Type: TypeInteger, Default: "24", Min: 1, Max: 720, Unit: "hours",
		Description: "Oldest offline scan that is still accepted"},
	{Key: "face_mobile_accuracy", Public: true, Group: "face", Type: TypeNumber, Default: "0.75", Min: 0, Max: 1,
		Description: "Face match score (0-1) the mobile app needs to mark a face as verified"},
	{Key: "face_match_min_score", Group: "face", Type: TypeNumber, Default: "0", Min: 0, Max: 1,
		Description: "Minimum face match score the server enforces on scans; 0 disables the check (only face_verified is checked)"},
	{Key: "face_photo_validation", Group: "face", Type: TypeBoolean, Default: "true",
		Description: "Check reference face photos (exactly one frontal face) when a user is created or the photo is changed"},
	{Key: "face_min_size_ratio", Group: "face", Type: TypeNumber, Default: "0.2", Min: 0.05, Max: 0.9,
		Description: "Minimum face width relative to the shorter side of a reference face photo"},
	{Key: "face_max_tilt_degrees", Group: "face", Type: TypeNumber, Default: "20", Min: 1, Max: 45, Unit: "degrees",
		Description: "Maximum head tilt in a reference face photo"},
	{Key: "face_max_turn_ratio", Group: "face", Type: TypeNumber, Default: "0.12", Min: 0.01, Max: 0.5,
		Description: "Maximum head turn in a reference face photo"},
	{Key: "login_max_failed_attempts", Public: true, Group: "security", Type: TypeInteger, Default: "3", Min: 1, Max: 20, Unit: "attempts",
		Description: "Wrong passwords in a row before the account is locked"},
	{Key: "login_lock_minutes", Public: true, Group: "security", Type: TypeInteger, Default: "5", Min: 1, Max: 1440, Unit: "minutes",
		Description: "How long a locked account stays locked"},
	{Key: "access_token_ttl_minutes", Public: true, Group: "security", Type: TypeInteger, Default: "60", Min: 5, Max: 1440, Unit: "minutes",
		Description: "Lifetime of an access token"},
	{Key: "refresh_token_ttl_days", Public: true, Group: "security", Type: TypeInteger, Default: "30", Min: 1, Max: 365, Unit: "days",
		Description: "How long a user stays signed in without entering the password"},
}

func definition(key string) (Definition, bool) {
	for _, def := range Definitions {
		if def.Key == key {
			return def, true
		}
	}
	return Definition{}, false
}

// normalize validates a raw value against the definition and returns its canonical text.
func (d Definition) normalize(raw string) (string, error) {
	raw = strings.TrimSpace(raw)

	switch d.Type {
	case TypeBoolean:
		value, err := strconv.ParseBool(raw)
		if err != nil {
			return "", fmt.Errorf("%s must be true or false", d.Key)
		}
		return strconv.FormatBool(value), nil

	case TypeInteger:
		value, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			return "", fmt.Errorf("%s must be a whole number", d.Key)
		}
		if float64(value) < d.Min || float64(value) > d.Max {
			return "", fmt.Errorf("%s must be between %s and %s", d.Key, formatNumber(d.Min), formatNumber(d.Max))
		}
		return strconv.FormatInt(value, 10), nil

	default: // TypeNumber
		value, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			return "", fmt.Errorf("%s must be a number", d.Key)
		}
		if value < d.Min || value > d.Max {
			return "", fmt.Errorf("%s must be between %s and %s", d.Key, formatNumber(d.Min), formatNumber(d.Max))
		}
		return strconv.FormatFloat(value, 'f', -1, 64), nil
	}
}

// Typed converts a canonical text value to its JSON type (number or boolean).
func (d Definition) Typed(value string) interface{} {
	switch d.Type {
	case TypeBoolean:
		v, _ := strconv.ParseBool(value)
		return v
	case TypeInteger:
		v, _ := strconv.ParseInt(value, 10, 64)
		return v
	default:
		v, _ := strconv.ParseFloat(value, 64)
		return v
	}
}

func formatNumber(value float64) string {
	return strconv.FormatFloat(value, 'f', -1, 64)
}
