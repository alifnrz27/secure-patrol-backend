package seeder

import (
	"log"
	"os"
	"secure-patrol-backend/helper"
	"secure-patrol-backend/models"
	"strings"
	"time"

	"gorm.io/gorm"
)

var defaultRoles = []models.Role{
	{Code: models.RoleSuperAdmin, Name: "Super-Admin", Description: "Akses penuh ke seluruh sistem"},
	{Code: models.RoleSecurityManager, Name: "Manager Keamanan", Description: "Mengelola operasional keamanan dan pengguna"},
	{Code: models.RoleSecurityHead, Name: "Kepala Keamanan", Description: "Memimpin dan mengawasi tim keamanan"},
	{Code: models.RoleSecurityAdmin, Name: "Admin Keamanan", Description: "Mengelola data administrasi keamanan dan pengguna"},
	{Code: models.RoleSecurityTeam, Name: "Tim Keamanan", Description: "Petugas patroli keamanan di lapangan"},
}

// SeedRoles makes sure the default system roles exist and returns their IDs by code.
// It is idempotent: existing roles are left untouched.
func SeedRoles(db *gorm.DB) (map[string]int64, error) {
	roleIDs := make(map[string]int64, len(defaultRoles))

	for _, role := range defaultRoles {
		role.IsSystem = true
		role.IsActive = true

		if err := db.Where(models.Role{Code: role.Code}).
			Attrs(role).
			FirstOrCreate(&role).Error; err != nil {
			return nil, err
		}

		roleIDs[role.Code] = role.ID
	}

	return roleIDs, nil
}

// Seed inserts the default roles, a Super-Admin and one dummy user per role.
// It only runs when SEED_DUMMY_DATA=true and is idempotent: existing rows
// (matched by their unique fields) are left untouched.
func Seed(db *gorm.DB) error {
	roleIDs, err := SeedRoles(db)
	if err != nil {
		return err
	}

	password := os.Getenv("SEED_DEFAULT_PASSWORD")
	generated := false
	if password == "" {
		random, err := helper.RandomURLToken(12)
		if err != nil {
			return err
		}
		password = random + "1a"
		generated = true
	}

	if err := helper.ValidatePasswordStrength(password); err != nil {
		return err
	}

	superAdminEmail := strings.ToLower(strings.TrimSpace(os.Getenv("SEED_SUPER_ADMIN_EMAIL")))
	if superAdminEmail == "" {
		superAdminEmail = "superadmin@securepatrol.local"
	}

	users := []struct {
		name     string
		email    string
		roleCode string
	}{
		{"Super Admin", superAdminEmail, models.RoleSuperAdmin},
		{"Manager Keamanan", "manager@securepatrol.local", models.RoleSecurityManager},
		{"Kepala Keamanan", "kepala@securepatrol.local", models.RoleSecurityHead},
		{"Admin Keamanan", "admin@securepatrol.local", models.RoleSecurityAdmin},
		{"Tim Keamanan", "tim@securepatrol.local", models.RoleSecurityTeam},
	}

	hash, err := helper.HashPassword(password)
	if err != nil {
		return err
	}

	now := time.Now()
	created := 0

	for _, u := range users {
		user := models.User{
			Name:              u.name,
			Email:             u.email,
			RoleID:            roleIDs[u.roleCode],
			PasswordHash:      hash,
			IsActive:          true,
			PasswordChangedAt: &now,
		}

		result := db.Where(models.User{Email: u.email}).Attrs(user).FirstOrCreate(&user)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected > 0 {
			created++
		}
	}

	log.Printf("[seeder] seeded %d role(s) and %d new user(s)", len(defaultRoles), created)

	if err := seedPatrolPoints(db); err != nil {
		return err
	}

	if err := seedHelpDeskArticles(db); err != nil {
		return err
	}

	if generated && created > 0 {
		log.Printf("[seeder] SEED_DEFAULT_PASSWORD is empty, generated password for new users: %s", password)
	}

	return nil
}

// EnsureDefaultShifts creates the default patrol shifts the first time the app
// starts. It never runs again once any shift was created (even if deleted later),
// so admin changes are kept.
func EnsureDefaultShifts(db *gorm.DB) error {
	var count int64
	if err := db.Unscoped().Model(&models.PatrolShift{}).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	shifts := []models.PatrolShift{
		{Name: "Shift 1", StartTime: "08:00", EndTime: "16:00", IsActive: true},
		{Name: "Shift 2", StartTime: "16:00", EndTime: "24:00", IsActive: true},
		{Name: "Shift 3", StartTime: "00:00", EndTime: "08:00", IsActive: true},
	}
	if err := db.Create(&shifts).Error; err != nil {
		return err
	}

	log.Printf("[seeder] created %d default patrol shift(s)", len(shifts))
	return nil
}

func seedPatrolPoints(db *gorm.DB) error {
	points := []models.PatrolPoint{
		{Name: "Main Gate", Location: "Building A - Front Entrance", NFCCode: "DUMMY-NFC-0001", Latitude: -6.2250138, Longitude: 106.8008324, IsLocationMatchRequired: true, IsFaceValidationRequired: true},
		{Name: "Parking Area", Location: "Basement 1", NFCCode: "DUMMY-NFC-0002", Latitude: -6.2252431, Longitude: 106.8011502, IsLocationMatchRequired: true, IsFaceValidationRequired: false},
		{Name: "Server Room", Location: "Building A - 3rd Floor", NFCCode: "DUMMY-NFC-0003", Latitude: -6.2249215, Longitude: 106.8009751, IsLocationMatchRequired: false, IsFaceValidationRequired: true},
	}

	created := 0
	for _, point := range points {
		result := db.Where(models.PatrolPoint{NFCCode: point.NFCCode}).Attrs(point).FirstOrCreate(&point)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected > 0 {
			created++
		}
	}

	log.Printf("[seeder] seeded %d new patrol point(s)", created)
	return nil
}
