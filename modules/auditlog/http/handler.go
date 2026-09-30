package http

import (
	"net/http"
	"secure-patrol-backend/helper"
	"secure-patrol-backend/models"
	"secure-patrol-backend/modules/auditlog/dto"
	"secure-patrol-backend/modules/auditlog/service"
	"secure-patrol-backend/pkg/log"

	"github.com/gofiber/fiber/v2"
)

type AuditLogHandler struct {
	service service.AuditLogService
	dto     dto.AuditLogDto
}

func NewAuditLogHandler(service service.AuditLogService, dto dto.AuditLogDto) *AuditLogHandler {
	return &AuditLogHandler{service: service, dto: dto}
}

func (h *AuditLogHandler) GetLogs(c *fiber.Ctx) error {
	var errs []string

	action := c.Query("action")
	if action != "" && action != models.AuditActionCreate && action != models.AuditActionUpdate && action != models.AuditActionDelete {
		errs = append(errs, "action must be create, update or delete")
	}
	dateFrom, err := helper.ParseDateQuery(c.Query("date_from"))
	if err != nil {
		errs = append(errs, "date_from: "+err.Error())
	}
	dateTo, err := helper.ParseDateQuery(c.Query("date_to"))
	if err != nil {
		errs = append(errs, "date_to: "+err.Error())
	}
	if errs != nil {
		response := helper.APIResponse("Validation error", http.StatusUnprocessableEntity, "Error", errs)
		return c.Status(http.StatusUnprocessableEntity).JSON(response)
	}

	filter := dto.AuditLogFilter{
		Pagination: helper.NewPagination(c),
		UserID:     int64(c.QueryInt("user_id", 0)),
		UnitID:     int64(c.QueryInt("unit_id", 0)),
		Action:     action,
		Resource:   c.Query("resource"),
		DateFrom:   dateFrom,
		DateTo:     dateTo,
	}

	logs, total, err := h.service.GetLogs(filter)
	if err != nil {
		log.Errorf("audit log handler: %v", err)
		response := helper.APIResponse("Internal server error", http.StatusInternalServerError, "Error", nil)
		return c.Status(http.StatusInternalServerError).JSON(response)
	}

	data := helper.NewPaginatedData(h.dto.ToAuditLogDTOs(logs), filter.Pagination, total)
	response := helper.APIResponse("Get audit logs success", http.StatusOK, "success", data)
	return c.Status(http.StatusOK).JSON(response)
}
