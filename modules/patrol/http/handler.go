package http

import (
	"errors"
	"mime/multipart"
	"net/http"
	"secure-patrol-backend/helper"
	"secure-patrol-backend/modules/patrol/dto"
	"secure-patrol-backend/modules/patrol/service"
	"secure-patrol-backend/pkg/log"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
)

type PatrolHandler struct {
	service service.PatrolService
	dto     dto.PatrolDto
}

func NewPatrolHandler(service service.PatrolService, dto dto.PatrolDto) *PatrolHandler {
	return &PatrolHandler{service: service, dto: dto}
}

func validationError(c *fiber.Ctx, errs []string) error {
	response := helper.APIResponse("Validation error", http.StatusUnprocessableEntity, "Error", errs)
	return c.Status(http.StatusUnprocessableEntity).JSON(response)
}

// dateRange reads ?date_from and ?date_to (YYYY-MM-DD).
func dateRange(c *fiber.Ctx) (string, string, []string) {
	var errs []string
	from, err := helper.ParseDateQuery(c.Query("date_from"))
	if err != nil {
		errs = append(errs, "date_from: "+err.Error())
	}
	to, err := helper.ParseDateQuery(c.Query("date_to"))
	if err != nil {
		errs = append(errs, "date_to: "+err.Error())
	}
	return from, to, errs
}

func (h *PatrolHandler) GetGroups(c *fiber.Ctx) error {
	from, to, errs := dateRange(c)
	if errs != nil {
		return validationError(c, errs)
	}

	filter := dto.GroupFilter{
		Pagination: helper.NewPagination(c),
		ShiftID:    int64(c.QueryInt("shift_id", 0)),
		DateFrom:   from,
		DateTo:     to,
	}

	groups, total, err := h.service.GetGroups(filter)
	if err != nil {
		return h.errorResponse(c, err)
	}

	data := helper.NewPaginatedData(h.dto.ToGroupDTOs(groups), filter.Pagination, total)
	response := helper.APIResponse("Get patrol groups success", http.StatusOK, "success", data)
	return c.Status(http.StatusOK).JSON(response)
}

func (h *PatrolHandler) GetCurrentGroup(c *fiber.Ctx) error {
	group, items, err := h.service.GetCurrentGroup()
	if err != nil {
		return h.errorResponse(c, err)
	}

	response := helper.APIResponse("Get current patrol group success", http.StatusOK, "success", h.dto.ToGroupDetailDTO(group, items))
	return c.Status(http.StatusOK).JSON(response)
}

func (h *PatrolHandler) GetGroup(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return h.errorResponse(c, service.ErrGroupNotFound)
	}

	group, items, err := h.service.GetGroup(int64(id))
	if err != nil {
		return h.errorResponse(c, err)
	}

	response := helper.APIResponse("Get patrol group success", http.StatusOK, "success", h.dto.ToGroupDetailDTO(group, items))
	return c.Status(http.StatusOK).JSON(response)
}

// GetItems lists patrol points of patrol groups, filtered by shift, period, group or scan status.
func (h *PatrolHandler) GetItems(c *fiber.Ctx) error {
	from, to, errs := dateRange(c)
	status := c.Query("status")
	if status != "" && status != "scanned" && status != "unscanned" {
		errs = append(errs, "status must be scanned or unscanned")
	}
	if errs != nil {
		return validationError(c, errs)
	}

	filter := dto.ItemFilter{
		Pagination: helper.NewPagination(c),
		GroupID:    int64(c.QueryInt("group_id", 0)),
		ShiftID:    int64(c.QueryInt("shift_id", 0)),
		DateFrom:   from,
		DateTo:     to,
		Status:     status,
	}

	items, total, err := h.service.GetItems(filter)
	if err != nil {
		return h.errorResponse(c, err)
	}

	data := helper.NewPaginatedData(h.dto.ToItemDTOs(items, true), filter.Pagination, total)
	response := helper.APIResponse("Get patrol list items success", http.StatusOK, "success", data)
	return c.Status(http.StatusOK).JSON(response)
}

func (h *PatrolHandler) Scan(c *fiber.Ctx) error {
	var req dto.ScanRequest
	if err := c.BodyParser(&req); err != nil {
		response := helper.APIResponse("Invalid request body", http.StatusBadRequest, "Error", err.Error())
		return c.Status(http.StatusBadRequest).JSON(response)
	}

	if errs := helper.ValidateStruct(req); errs != nil {
		return validationError(c, errs)
	}

	var scannedAt *time.Time
	if req.ScannedAt != "" {
		parsed, err := time.Parse(time.RFC3339, req.ScannedAt)
		if err != nil {
			return validationError(c, []string{"scanned_at must be an RFC3339 date time, e.g. 2026-09-29T08:15:00+07:00"})
		}
		scannedAt = &parsed
	}

	scan, duplicate, err := h.service.Scan(service.ScanInput{
		ClientScanID:   req.ClientScanID,
		NFCCode:        req.NFCCode,
		Condition:      req.Condition,
		Note:           req.Note,
		Latitude:       *req.Latitude,
		Longitude:      *req.Longitude,
		ScannedAt:      scannedAt,
		FaceVerified:   req.FaceVerified != nil && *req.FaceVerified,
		FaceMatchScore: req.FaceMatchScore,
		Photos:         scanPhotos(c),
		UserID:         helper.CurrentUserID(c),
		AppClientID:    helper.CurrentAppClientID(c),
	})
	if err != nil {
		return h.errorResponse(c, err)
	}

	if duplicate {
		response := helper.APIResponse("Patrol scan was already recorded", http.StatusOK, "success", h.dto.ToScanDTO(scan))
		return c.Status(http.StatusOK).JSON(response)
	}

	response := helper.APIResponse("Patrol scan success", http.StatusCreated, "success", h.dto.ToScanDTO(scan))
	return c.Status(http.StatusCreated).JSON(response)
}

func (h *PatrolHandler) GetScans(c *fiber.Ctx) error {
	from, to, errs := dateRange(c)
	condition := c.Query("condition")
	if condition != "" && condition != "normal" && condition != "abnormal" {
		errs = append(errs, "condition must be normal or abnormal")
	}
	if errs != nil {
		return validationError(c, errs)
	}

	filter := dto.ScanFilter{
		Pagination:    helper.NewPagination(c),
		GroupID:       int64(c.QueryInt("group_id", 0)),
		ShiftID:       int64(c.QueryInt("shift_id", 0)),
		DateFrom:      from,
		DateTo:        to,
		ScannedBy:     int64(c.QueryInt("scanned_by", 0)),
		PatrolPointID: int64(c.QueryInt("patrol_point_id", 0)),
		Condition:     condition,
	}

	scans, total, err := h.service.GetScans(actor(c), filter)
	if err != nil {
		return h.errorResponse(c, err)
	}

	data := helper.NewPaginatedData(h.dto.ToScanDTOs(scans), filter.Pagination, total)
	response := helper.APIResponse("Get patrol scans success", http.StatusOK, "success", data)
	return c.Status(http.StatusOK).JSON(response)
}

// ExportScans downloads the scan history as an Excel file. Filters: shift_id,
// patrol_point_id, scanned_by, date_from and date_to (shift date).
func (h *PatrolHandler) ExportScans(c *fiber.Ctx) error {
	from, to, errs := dateRange(c)
	if errs != nil {
		return validationError(c, errs)
	}

	filter := dto.ScanFilter{
		ShiftID:       int64(c.QueryInt("shift_id", 0)),
		PatrolPointID: int64(c.QueryInt("patrol_point_id", 0)),
		ScannedBy:     int64(c.QueryInt("scanned_by", 0)),
		DateFrom:      from,
		DateTo:        to,
	}

	scans, err := h.service.ExportScans(actor(c), filter)
	if err != nil {
		return h.errorResponse(c, err)
	}

	shift, point, officer, err := h.service.FilterNames(filter)
	if err != nil {
		return h.errorResponse(c, err)
	}

	exportedBy := ""
	if user, ok := c.Locals(helper.LocalUserID).(int64); ok {
		exportedBy = strconv.FormatInt(user, 10)
		if _, _, name, err := h.service.FilterNames(dto.ScanFilter{ScannedBy: user}); err == nil && name != "" {
			exportedBy = name
		}
	}

	now := time.Now()
	content, err := dto.BuildScanExcel(scans, dto.ScanExportMeta{
		ExportedAt: now,
		ExportedBy: exportedBy,
		Shift:      shift,
		Point:      point,
		Officer:    officer,
		DateFrom:   from,
		DateTo:     to,
		Location:   helper.AppLocation(),
	})
	if err != nil {
		return h.errorResponse(c, err)
	}

	filename := "riwayat-scan_" + now.In(helper.AppLocation()).Format("20060102-150405") + ".xlsx"
	c.Set(fiber.HeaderContentType, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Set(fiber.HeaderContentDisposition, `attachment; filename="`+filename+`"`)
	c.Set(fiber.HeaderCacheControl, "private, no-store")
	return c.Send(content)
}

func (h *PatrolHandler) GetScan(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return h.errorResponse(c, service.ErrScanNotFound)
	}

	scan, err := h.service.GetScan(actor(c), int64(id))
	if err != nil {
		return h.errorResponse(c, err)
	}

	response := helper.APIResponse("Get patrol scan success", http.StatusOK, "success", h.dto.ToScanDTO(scan))
	return c.Status(http.StatusOK).JSON(response)
}

func (h *PatrolHandler) GetScanPhoto(c *fiber.Ctx) error {
	scanID, err := c.ParamsInt("id")
	if err != nil {
		return h.errorResponse(c, service.ErrScanNotFound)
	}
	photoID, err := c.ParamsInt("photo_id")
	if err != nil {
		return h.errorResponse(c, service.ErrScanNotFound)
	}

	path, err := h.service.GetScanPhotoPath(actor(c), int64(scanID), int64(photoID))
	if err != nil {
		return h.errorResponse(c, err)
	}

	c.Set(fiber.HeaderCacheControl, "private, no-store")
	return c.SendFile(path)
}

func (h *PatrolHandler) errorResponse(c *fiber.Ctx, err error) error {
	code := http.StatusInternalServerError
	message := "Internal server error"

	var locationErr *service.LocationOutOfRangeError

	switch {
	case errors.Is(err, service.ErrGroupNotFound),
		errors.Is(err, service.ErrScanNotFound),
		errors.Is(err, service.ErrNFCNotRegistered):
		code, message = http.StatusNotFound, err.Error()
	case errors.As(err, &locationErr),
		errors.Is(err, service.ErrNoActiveShift),
		errors.Is(err, service.ErrScannedAtInFuture),
		errors.Is(err, service.ErrScanTooOld),
		errors.Is(err, service.ErrNoteRequired),
		errors.Is(err, service.ErrTooManyPhotos),
		errors.Is(err, service.ErrFaceNotVerified),
		errors.Is(err, service.ErrFaceMatchScoreMissing),
		errors.Is(err, service.ErrFaceMatchScoreTooLow),
		errors.Is(err, helper.ErrFileTooLarge),
		errors.Is(err, helper.ErrFileTypeNotAllowed),
		errors.Is(err, service.ErrExportTooLarge),
		errors.Is(err, service.ErrDateRangeInvalid):
		code, message = http.StatusUnprocessableEntity, err.Error()
	default:
		log.Errorf("patrol handler: %v", err)
	}

	response := helper.APIResponse(message, code, "Error", nil)
	return c.Status(code).JSON(response)
}

func actor(c *fiber.Ctx) service.Actor {
	return service.Actor{
		UserID:   helper.CurrentUserID(c),
		RoleCode: helper.CurrentRoleCode(c),
	}
}

// scanPhotos accepts the files sent as "photos" (repeated) or "photos[]".
func scanPhotos(c *fiber.Ctx) []*multipart.FileHeader {
	form, err := c.MultipartForm()
	if err != nil {
		return nil
	}
	return append(form.File["photos"], form.File["photos[]"]...)
}
