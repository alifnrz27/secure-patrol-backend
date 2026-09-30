package dto

import (
	"secure-patrol-backend/helper"
	"time"
)

type AuditLogFilter struct {
	helper.Pagination
	UserID   int64
	Action   string
	Resource string
	DateFrom string
	DateTo   string
}

type AuditUserDTO struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	Email    string `json:"email"`
	RoleCode string `json:"role_code"`
}

type AuditLogDTO struct {
	ID          int64         `json:"id"`
	Action      string        `json:"action"`
	Resource    string        `json:"resource"`
	ResourceID  *string       `json:"resource_id"`
	Endpoint    string        `json:"endpoint"`
	Method      string        `json:"method"`
	Path        string        `json:"path"`
	StatusCode  int           `json:"status_code"`
	User        *AuditUserDTO `json:"user"`
	AppPlatform string        `json:"app_platform"`
	IPAddress   string        `json:"ip_address"`
	UserAgent   string        `json:"user_agent"`
	Source      string        `json:"source"`
	CreatedAt   time.Time     `json:"created_at"`
}
