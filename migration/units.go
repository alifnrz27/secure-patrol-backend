package migration

import (
	"log"
	"secure-patrol-backend/models"

	"gorm.io/gorm"
)

// ensureSettingIndexes makes a setting key unique once globally (unit_id NULL)
// and once per unit, replacing the old unique index on the key alone.
func ensureSettingIndexes(db *gorm.DB) error {
	statements := []string{
		"DROP INDEX IF EXISTS idx_system_settings_key",
		"CREATE UNIQUE INDEX IF NOT EXISTS idx_system_settings_global_key ON system_settings (key) WHERE unit_id IS NULL AND deleted_at IS NULL",
		"CREATE UNIQUE INDEX IF NOT EXISTS idx_system_settings_unit_key ON system_settings (unit_id, key) WHERE unit_id IS NOT NULL AND deleted_at IS NULL",
	}
	for _, statement := range statements {
		if err := db.Exec(statement).Error; err != nil {
			return err
		}
	}
	return nil
}

// assignLegacyDataToUnit moves data created before units existed (patrol
// points, shifts, groups and unit-role users without a unit) into one unit, so
// nothing is left without an owner. The unit is created when needed.
func assignLegacyDataToUnit(db *gorm.DB) error {
	var legacy int64
	err := db.Raw(`SELECT
		(SELECT COUNT(*) FROM patrol_points WHERE unit_id = 0) +
		(SELECT COUNT(*) FROM patrol_shifts WHERE unit_id = 0) +
		(SELECT COUNT(*) FROM users u JOIN roles r ON r.id = u.role_id
			WHERE u.unit_id IS NULL AND r.code NOT IN (?, ?))`,
		models.RoleSuperAdmin, models.RoleSecurityManager).Scan(&legacy).Error
	if err != nil || legacy == 0 {
		if err == nil {
			err = syncGroupUnits(db)
		}
		return err
	}

	var unit models.Unit
	if err := db.Order("id ASC").First(&unit).Error; err != nil {
		if err != gorm.ErrRecordNotFound {
			return err
		}

		// Place the unit at the first patrol point, if any; it can be edited later.
		var point models.PatrolPoint
		db.Unscoped().Order("id ASC").Limit(1).Find(&point)

		unit = models.Unit{
			Code:      "UNIT-UTAMA",
			Name:      "Unit Utama",
			Latitude:  point.Latitude,
			Longitude: point.Longitude,
			IsActive:  true,
		}
		if err := db.Create(&unit).Error; err != nil {
			return err
		}
		log.Printf("[migration] created unit %q for data created before units existed", unit.Code)
	}

	return db.Transaction(func(tx *gorm.DB) error {
		statements := []struct {
			sql  string
			args []interface{}
		}{
			{"UPDATE patrol_points SET unit_id = ? WHERE unit_id = 0", []interface{}{unit.ID}},
			{"UPDATE patrol_shifts SET unit_id = ? WHERE unit_id = 0", []interface{}{unit.ID}},
			{`UPDATE users SET unit_id = ? WHERE unit_id IS NULL AND role_id IN
				(SELECT id FROM roles WHERE code NOT IN (?, ?))`,
				[]interface{}{unit.ID, models.RoleSuperAdmin, models.RoleSecurityManager}},
		}
		for _, statement := range statements {
			result := tx.Exec(statement.sql, statement.args...)
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected > 0 {
				log.Printf("[migration] assigned %d row(s) to unit %q", result.RowsAffected, unit.Code)
			}
		}
		if err := syncGroupUnits(tx); err != nil {
			return err
		}

		// A unit always starts with shifts, so its patrols can run right away.
		var shifts int64
		if err := tx.Model(&models.PatrolShift{}).Where("unit_id = ?", unit.ID).Count(&shifts).Error; err != nil {
			return err
		}
		if shifts > 0 {
			return nil
		}
		defaults := models.DefaultShifts(unit.ID)
		return tx.Create(&defaults).Error
	})
}

// syncGroupUnits sets the unit of patrol groups from their shift.
func syncGroupUnits(db *gorm.DB) error {
	return db.Exec(`UPDATE patrol_groups g SET unit_id = s.unit_id
		FROM patrol_shifts s WHERE g.patrol_shift_id = s.id AND g.unit_id = 0`).Error
}
