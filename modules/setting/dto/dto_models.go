package dto

import (
	"encoding/json"
	"time"
)

type SettingDTO struct {
	Key string `json:"key"`
	// Level is "global" or "unit": which level this list shows.
	Level        string      `json:"level"`
	Group        string      `json:"group"`
	Type         string      `json:"type"`
	Value        interface{} `json:"value"`
	DefaultValue interface{} `json:"default_value"`
	IsDefault    bool        `json:"is_default"`
	// GlobalValue is the value set by the head office; a unit follows it unless it has its own value.
	GlobalValue interface{} `json:"global_value"`
	// IsInherited is true when a unit has no own value and follows the global value.
	IsInherited bool       `json:"is_inherited"`
	Min         *float64   `json:"min"`
	Max         *float64   `json:"max"`
	Unit        string     `json:"unit"`
	Description string     `json:"description"`
	UpdatedBy   *int64     `json:"updated_by"`
	UpdatedAt   *time.Time `json:"updated_at"`
}

// UpdateSettingsRequest changes several settings at once, e.g.
// {"values": {"patrol_location_radius_meters": 150, "face_match_min_score": null}}.
// A null value resets that setting to its default.
type UpdateSettingsRequest struct {
	Values map[string]json.RawMessage `json:"values"`
}
