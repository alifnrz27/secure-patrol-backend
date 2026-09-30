package models

import (
	"time"

	"gorm.io/gorm"
)

// Audit log actions. Only data changes are recorded.
const (
	AuditActionCreate = "create"
	AuditActionUpdate = "update"
	AuditActionDelete = "delete"
)

// Audit log sources.
const (
	AuditSourceAPI = "api"
	AuditSourceCLI = "cli"
)

// AuditLog records who changed which data, when and from where. Request bodies
// are never stored because they can contain passwords and face photos.
type AuditLog struct {
	ID          int64          `json:"id" gorm:"primaryKey"`
	Action      string         `json:"action" gorm:"type:varchar(10);not null;index"`
	Resource    string         `json:"resource" gorm:"type:varchar(100);not null;index"`
	ResourceID  *string        `json:"resource_id" gorm:"type:varchar(64);index"`
	Endpoint    string         `json:"endpoint" gorm:"type:varchar(150);not null"`
	Method      string         `json:"method" gorm:"type:varchar(10);not null"`
	Path        string         `json:"path" gorm:"type:varchar(255);not null"`
	StatusCode  int            `json:"status_code" gorm:"not null"`
	UserID      *int64         `json:"user_id" gorm:"index"`
	User        *User          `json:"-" gorm:"foreignKey:UserID"`
	RoleCode    string         `json:"role_code" gorm:"type:varchar(50)"`
	AppClientID *int64         `json:"app_client_id"`
	AppPlatform string         `json:"app_platform" gorm:"type:varchar(20)"`
	IPAddress   string         `json:"ip_address" gorm:"type:varchar(45)"`
	UserAgent   string         `json:"user_agent" gorm:"type:varchar(255)"`
	Source      string         `json:"source" gorm:"type:varchar(10);not null"`
	CreatedAt   time.Time      `json:"created_at" gorm:"index"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`
}
