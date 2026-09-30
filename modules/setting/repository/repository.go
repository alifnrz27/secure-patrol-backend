package repository

import "secure-patrol-backend/models"

// A setting row with unit_id NULL is the global value; a row with a unit_id
// overrides the global value for that unit.
type SettingRepository interface {
	FindGlobal() ([]models.SystemSetting, error)
	FindByUnit(unitID int64) ([]models.SystemSetting, error)
	// InsertMissingGlobal creates the given global settings that do not exist yet.
	InsertMissingGlobal(settings []models.SystemSetting) (int64, error)
	// Save writes several values in one transaction; unitID nil writes global values.
	Save(unitID *int64, values map[string]string, updatedBy int64) error
	// DeleteUnitOverrides removes unit values so the unit follows the global value again.
	DeleteUnitOverrides(unitID int64, keys []string) error
}
