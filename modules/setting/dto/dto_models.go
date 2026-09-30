package dto

import (
	"encoding/json"
	"time"
)

type SettingDTO struct {
	Key          string      `json:"key"`
	Group        string      `json:"group"`
	Type         string      `json:"type"`
	Value        interface{} `json:"value"`
	DefaultValue interface{} `json:"default_value"`
	IsDefault    bool        `json:"is_default"`
	Min          *float64    `json:"min"`
	Max          *float64    `json:"max"`
	Unit         string      `json:"unit"`
	Description  string      `json:"description"`
	UpdatedBy    *int64      `json:"updated_by"`
	UpdatedAt    *time.Time  `json:"updated_at"`
}

// UpdateSettingsRequest changes several settings at once, e.g.
// {"values": {"patrol_location_radius_meters": 150, "face_match_min_score": null}}.
// A null value resets that setting to its default.
type UpdateSettingsRequest struct {
	Values map[string]json.RawMessage `json:"values"`
}
