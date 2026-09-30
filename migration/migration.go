package migration

import (
	"log"
	"secure-patrol-backend/models"

	"gorm.io/gorm"
)

func Migrate(db *gorm.DB) error {
	migrationModels := []any{
		&models.Role{},
		&models.User{},
		&models.UserSession{},
		&models.AppClient{},
		&models.PatrolPoint{},
		&models.PatrolShift{},
		&models.PatrolGroup{},
		&models.PatrolListItem{},
		&models.PatrolScan{},
		&models.PatrolScanPhoto{},
		&models.HelpDeskArticle{},
		&models.AuditLog{},
		&models.SystemSetting{},
	}

	for _, model := range migrationModels {
		if err := migrateModel(db, model); err != nil {
			return err
		}
	}

	return ensureCaseInsensitiveUniqueIndexes(db)
}

// ensureCaseInsensitiveUniqueIndexes adds database level guarantees on top of the
// normalization done in the services: "Budi@x.com" and "budi@x.com" are the same
// email, and "04:a2:1f" and "04:A2:1F" are the same NFC tag.
func ensureCaseInsensitiveUniqueIndexes(db *gorm.DB) error {
	indexes := []struct {
		name string
		sql  string
	}{
		{"idx_users_email_lower", "CREATE UNIQUE INDEX IF NOT EXISTS idx_users_email_lower ON users (LOWER(email))"},
		{"idx_patrol_points_nfc_code_upper", "CREATE UNIQUE INDEX IF NOT EXISTS idx_patrol_points_nfc_code_upper ON patrol_points (UPPER(nfc_code))"},
	}

	for _, index := range indexes {
		if err := db.Exec(index.sql).Error; err != nil {
			log.Printf("[migration] could not create unique index %q, check for duplicate values that differ only in letter case", index.name)
			return err
		}
	}

	return nil
}

// migrateModel runs AutoMigrate for a single model and logs whether the
// table was created, had new column(s) added, or was already up to date.
func migrateModel(db *gorm.DB, model any) error {
	stmt := &gorm.Statement{DB: db}
	if err := stmt.Parse(model); err != nil {
		return err
	}

	tableName := stmt.Schema.Table
	migrator := db.Migrator()

	if !migrator.HasTable(model) {
		log.Printf("[migration] table %q does not exist, creating...", tableName)

		if err := db.AutoMigrate(model); err != nil {
			return err
		}

		log.Printf("[migration] table %q created", tableName)
		return nil
	}

	existingColumns, err := migrator.ColumnTypes(model)
	if err != nil {
		return err
	}

	existing := make(map[string]struct{}, len(existingColumns))
	for _, column := range existingColumns {
		existing[column.Name()] = struct{}{}
	}

	// Remove legacy MyTelkomsel user columns.
	if tableName == "users" {
		legacyColumns := []string{
			"user_app_id",
			"gcm_app_id",
			"token",
		}

		for _, column := range legacyColumns {
			if _, exists := existing[column]; exists {
				log.Printf(
					"[migration] dropping legacy column %q.%q",
					tableName,
					column,
				)

				if err := migrator.DropColumn(model, column); err != nil {
					return err
				}

				delete(existing, column)
			}
		}
	}

	var newColumns []string

	for _, field := range stmt.Schema.Fields {
		if field.DBName == "" || field.IgnoreMigration {
			continue
		}

		if _, ok := existing[field.DBName]; !ok {
			newColumns = append(newColumns, field.DBName)
		}
	}

	var newIndexes []string

	for name := range stmt.Schema.ParseIndexes() {
		if !migrator.HasIndex(model, name) {
			newIndexes = append(newIndexes, name)
		}
	}

	if len(newColumns) == 0 && len(newIndexes) == 0 {
		log.Printf(
			"[migration] table %q already exists, no changes needed",
			tableName,
		)

		return nil
	}

	log.Printf(
		"[migration] table %q exists, applying change(s): new column(s) %v, new index(es) %v",
		tableName,
		newColumns,
		newIndexes,
	)

	if err := db.AutoMigrate(model); err != nil {
		return err
	}

	log.Printf(
		"[migration] table %q updated: new column(s) %v, new index(es) %v",
		tableName,
		newColumns,
		newIndexes,
	)

	return nil
}
