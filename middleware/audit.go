package middleware

import (
	"encoding/json"
	"net/http"
	"regexp"
	"secure-patrol-backend/helper"
	"secure-patrol-backend/models"
	"secure-patrol-backend/modules/auditlog/service"
	"secure-patrol-backend/pkg/log"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
)

var numericSegment = regexp.MustCompile(`^\d+$`)

// Session endpoints are not data changes and are not audited.
var auditExcludedPaths = map[string]bool{
	"auth/login":      true,
	"auth/refresh":    true,
	"auth/logout":     true,
	"auth/logout-all": true,
}

// AuditLog records every successful create, update and delete made through the
// API: who (user, role), when, from where (IP, user agent, app platform) and
// what (endpoint and resource id). It must be used after BearerAuth.
func AuditLog(auditService service.AuditLogService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		err := c.Next()

		method := c.Method()
		if err != nil || (method != http.MethodPost && method != http.MethodPut && method != http.MethodDelete) {
			return err
		}

		status := c.Response().StatusCode()
		if status < 200 || status >= 300 {
			return nil
		}

		path := strings.Trim(strings.TrimPrefix(c.Path(), "/api/v1"), "/")
		if auditExcludedPaths[path] {
			return nil
		}

		segments := strings.Split(path, "/")
		template := make([]string, len(segments))
		var resourceID string
		for i, segment := range segments {
			if numericSegment.MatchString(segment) {
				template[i] = ":id"
				if resourceID == "" {
					resourceID = segment
				}
				continue
			}
			template[i] = segment
		}

		action := models.AuditActionUpdate
		switch {
		case method == http.MethodDelete:
			action = models.AuditActionDelete
		case method == http.MethodPost && resourceID == "":
			// Creates answer 201. A 200 here means nothing new was stored,
			// e.g. an offline scan that was already recorded.
			if status != http.StatusCreated {
				return nil
			}
			action = models.AuditActionCreate
		}

		resource := segments[0]
		if path == "auth/change-password" {
			resource, resourceID = "users", strconv.FormatInt(helper.CurrentUserID(c), 10)
		}
		if resourceID == "" {
			resourceID = responseDataID(c.Response().Body())
		}

		entry := models.AuditLog{
			Action:      action,
			Resource:    resource,
			Endpoint:    method + " /" + strings.Join(template, "/"),
			Method:      method,
			Path:        c.Path(),
			StatusCode:  status,
			RoleCode:    helper.CurrentRoleCode(c),
			AppPlatform: helper.CurrentAppPlatform(c),
			IPAddress:   c.IP(),
			UserAgent:   c.Get(fiber.HeaderUserAgent),
			Source:      models.AuditSourceAPI,
		}
		if resourceID != "" {
			entry.ResourceID = &resourceID
		}
		if userID := helper.CurrentUserID(c); userID > 0 {
			entry.UserID = &userID
		}
		entry.UnitID = helper.CurrentScope(c).UnitID
		if appClientID := helper.CurrentAppClientID(c); appClientID > 0 {
			entry.AppClientID = &appClientID
		}

		// The change already happened; a failed audit write is logged, not returned.
		if recordErr := auditService.Record(entry); recordErr != nil {
			log.Errorf("audit log: %v", recordErr)
		}
		return nil
	}
}

// responseDataID reads data.id from a JSON API response (the id of a created record).
func responseDataID(body []byte) string {
	var envelope struct {
		Data struct {
			ID json.Number `json:"id"`
		} `json:"data"`
	}
	if json.Unmarshal(body, &envelope) != nil {
		return ""
	}
	return envelope.Data.ID.String()
}
