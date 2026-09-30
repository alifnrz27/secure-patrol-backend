package console

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"secure-patrol-backend/helper"
	"secure-patrol-backend/models"
	"secure-patrol-backend/modules/appclient/repository"
	"secure-patrol-backend/modules/appclient/service"
	auditlogrepository "secure-patrol-backend/modules/auditlog/repository"
	auditlogservice "secure-patrol-backend/modules/auditlog/service"
	"secure-patrol-backend/pkg/nonce"
	"secure-patrol-backend/seeder"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"
)

// Run executes a console command when one is given and reports whether it did.
//
//	go run . create-app-client -name "Secure Patrol Android" -platform android
//	go run . create-super-admin -email superadmin@securepatrol.local
func Run(db *gorm.DB, args []string) bool {
	if len(args) == 0 {
		return false
	}

	switch args[0] {
	case "create-app-client":
		createAppClient(db, args[1:])
	case "create-super-admin":
		createSuperAdmin(db, args[1:])
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\navailable commands: create-app-client, create-super-admin\n", args[0])
		os.Exit(1)
	}

	return true
}

// createAppClient bootstraps an app client from the terminal, needed because
// the API itself can only be called by an existing app client.
func createAppClient(db *gorm.DB, args []string) {
	flags := flag.NewFlagSet("create-app-client", flag.ExitOnError)
	name := flags.String("name", "", "app client name (required)")
	platform := flags.String("platform", "", "android | ios | web | server (required)")
	description := flags.String("description", "", "optional description")
	flags.Parse(args)

	if *name == "" || *platform == "" {
		flags.Usage()
		os.Exit(1)
	}

	appClientService := service.NewAppClientService(repository.NewAppClientRepository(db), nonce.Default())

	client, appKey, err := appClientService.CreateAppClient(models.AppClient{
		Name:        *name,
		Platform:    *platform,
		Description: *description,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "create app client failed: %v\n", err)
		os.Exit(1)
	}

	recordCLIAudit(db, "app-clients", client.ID)

	fmt.Println("App client created. The app key is shown only once, store it securely.")
	fmt.Println()
	fmt.Printf("  Name     : %s\n", client.Name)
	fmt.Printf("  Platform : %s\n", client.Platform)
	fmt.Printf("  App ID   : %s\n", client.AppID)
	fmt.Printf("  App Key  : %s\n", appKey)
}

// createSuperAdmin bootstraps the first Super-Admin account. The password is
// generated and printed once, so it never ends up in shell history.
func createSuperAdmin(db *gorm.DB, args []string) {
	flags := flag.NewFlagSet("create-super-admin", flag.ExitOnError)
	name := flags.String("name", "Super Admin", "display name")
	email := flags.String("email", os.Getenv("SEED_SUPER_ADMIN_EMAIL"), "login email (default SEED_SUPER_ADMIN_EMAIL)")
	flags.Parse(args)

	*email = strings.ToLower(strings.TrimSpace(*email))
	if *email == "" {
		flags.Usage()
		os.Exit(1)
	}

	roleIDs, err := seeder.SeedRoles(db)
	if err != nil {
		fmt.Fprintf(os.Stderr, "seed roles failed: %v\n", err)
		os.Exit(1)
	}

	var existing models.User
	err = db.Where("email = ?", *email).First(&existing).Error
	if err == nil {
		fmt.Fprintf(os.Stderr, "user with email %s already exists\n", *email)
		os.Exit(1)
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		fmt.Fprintf(os.Stderr, "check existing user failed: %v\n", err)
		os.Exit(1)
	}

	random, err := helper.RandomURLToken(12)
	if err != nil {
		fmt.Fprintf(os.Stderr, "generate password failed: %v\n", err)
		os.Exit(1)
	}
	// 16 random base64url characters plus a letter and digit to satisfy the password policy.
	password := random + "a1"

	hash, err := helper.HashPassword(password)
	if err != nil {
		fmt.Fprintf(os.Stderr, "hash password failed: %v\n", err)
		os.Exit(1)
	}

	now := time.Now()
	user := models.User{
		Name:              *name,
		Email:             *email,
		RoleID:            roleIDs[models.RoleSuperAdmin],
		PasswordHash:      hash,
		IsActive:          true,
		PasswordChangedAt: &now,
	}

	if err := db.Create(&user).Error; err != nil {
		fmt.Fprintf(os.Stderr, "create super admin failed: %v\n", err)
		os.Exit(1)
	}

	recordCLIAudit(db, "users", user.ID)

	fmt.Println("Super-Admin created. The password is shown only once, store it securely and change it after the first login.")
	fmt.Println()
	fmt.Printf("  Name     : %s\n", user.Name)
	fmt.Printf("  Email    : %s\n", user.Email)
	fmt.Printf("  Password : %s\n", password)
}

// recordCLIAudit logs a record created from the terminal (no user or IP).
func recordCLIAudit(db *gorm.DB, resource string, id int64) {
	resourceID := strconv.FormatInt(id, 10)
	auditService := auditlogservice.NewAuditLogService(auditlogrepository.NewAuditLogRepository(db))
	err := auditService.Record(models.AuditLog{
		Action:     models.AuditActionCreate,
		Resource:   resource,
		ResourceID: &resourceID,
		Endpoint:   "CLI " + strings.Join(os.Args[1:2], ""),
		Method:     "CLI",
		Path:       strings.Join(os.Args[1:2], ""),
		StatusCode: 0,
		Source:     models.AuditSourceCLI,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: audit log not written: %v\n", err)
	}
}
