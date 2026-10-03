package models

import (
	"time"

	"gorm.io/gorm"
)

// Role codes used for authorization checks.
const (
	RoleSuperAdmin      = "super_admin"
	RoleSecurityManager = "security_manager"
	RoleSecurityHead    = "security_head"
	RoleSecurityAdmin   = "security_admin"
	RoleSecurityTeam    = "security_team"
)

// CentralRoles work for the whole organization and do not belong to a unit.
// Every other role (including custom roles) belongs to exactly one unit.
var CentralRoles = []string{RoleSuperAdmin, RoleSecurityManager}

// UnitManagerRoles manage the data of their own unit.
var UnitManagerRoles = []string{RoleSecurityHead, RoleSecurityAdmin}

// AssignableRoles can be assigned to patrol points of the running shift.
var AssignableRoles = []string{RoleSecurityAdmin, RoleSecurityTeam}

func IsCentralRole(code string) bool {
	return code == RoleSuperAdmin || code == RoleSecurityManager
}

type Role struct {
	ID          int64          `json:"id" gorm:"primaryKey"`
	Code        string         `json:"code" gorm:"type:varchar(50);uniqueIndex;not null"`
	Name        string         `json:"name" gorm:"type:varchar(100);not null"`
	Description string         `json:"description" gorm:"type:varchar(255)"`
	IsSystem    bool           `json:"is_system" gorm:"not null"`
	IsActive    bool           `json:"is_active" gorm:"not null"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`
}
