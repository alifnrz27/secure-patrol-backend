package service

import (
	"context"
	"errors"
	"fmt"
	"mime/multipart"
	"secure-patrol-backend/helper"
	"secure-patrol-backend/models"
	"secure-patrol-backend/modules/patrol/dto"
	"secure-patrol-backend/modules/patrol/repository"
	"time"
)

const MaxScanPhotos = helper.MaxScanPhotos

// MaxExportRows keeps an export small enough to build in memory. Exports with
// photos embed a thumbnail per photo, so they allow fewer rows. Both are a
// safety net on top of the date range limits in the settings.
const (
	MaxExportRows      = 50000
	MaxExportPhotoRows = 2000
)

var (
	ErrGroupNotFound         = errors.New("patrol group not found")
	ErrScanNotFound          = errors.New("patrol scan not found")
	ErrNoActiveShift         = errors.New("there is no active patrol shift at the scan time")
	ErrNFCNotRegistered      = errors.New("nfc tag is not registered as a patrol point")
	ErrScannedAtInFuture     = errors.New("scanned_at cannot be in the future")
	ErrScanTooOld            = errors.New("scan is too old to be submitted")
	ErrNoteRequired          = errors.New("note is required when the condition is abnormal")
	ErrTooManyPhotos         = fmt.Errorf("a scan can have at most %d photos", MaxScanPhotos)
	ErrFaceNotVerified       = errors.New("face validation is required for this patrol point and was not verified")
	ErrFaceMatchScoreMissing = errors.New("face_match_score is required for this patrol point")
	ErrFaceMatchScoreTooLow  = errors.New("face match score is below the minimum required")
	ErrExportTooLarge        = fmt.Errorf("export is limited to %d rows, narrow the filter (for example the date range)", MaxExportRows)
	ErrExportPhotosTooLarge  = fmt.Errorf("an export with photos is limited to %d rows, narrow the filter (for example the date range)", MaxExportPhotoRows)
	ErrExportRangeRequired   = errors.New("date_from and date_to are required for an export")
	ErrSummaryTargetMissing  = errors.New("group_id or shift_id is required")
	ErrShiftNotFound         = errors.New("patrol shift not found")
	ErrDateRangeInvalid      = errors.New("date_from must not be after date_to")
	ErrUnitRequired          = errors.New("unit_id is required")
	ErrUnitNotFound          = errors.New("unit not found")
	ErrScanNeedsUnit         = errors.New("only unit users can scan patrol points")
	ErrPointOfOtherUnit      = errors.New("this patrol point belongs to another unit")
)

// ExportRangeError is returned when the export date range is longer than the
// settings allow.
type ExportRangeError struct {
	MaxDays    int
	WithPhotos bool
}

func (e *ExportRangeError) Error() string {
	if e.WithPhotos {
		if e.MaxDays == 1 {
			return "an export with photos can cover only 1 day, use the same date_from and date_to"
		}
		return fmt.Sprintf("an export with photos can cover at most %d days, narrow date_from and date_to", e.MaxDays)
	}
	return fmt.Sprintf("an export can cover at most %d days, narrow date_from and date_to", e.MaxDays)
}

// LocationOutOfRangeError is returned when the officer is too far from the patrol point.
type LocationOutOfRangeError struct {
	Distance    float64
	MaxDistance float64
}

func (e *LocationOutOfRangeError) Error() string {
	return fmt.Sprintf("you are %.0f m away from the patrol point, the maximum allowed distance is %.0f m", e.Distance, e.MaxDistance)
}

// Actor is the logged in user performing the request.
type Actor struct {
	UserID   int64
	RoleCode string
	// UnitID is the unit of a unit user; nil for head office users.
	UnitID *int64
}

func (a Actor) scope() helper.Scope {
	return helper.Scope{UserID: a.UserID, RoleCode: a.RoleCode, UnitID: a.UnitID}
}

// ScanInput is one NFC scan sent by the mobile app. ScannedAt is the moment
// the tag was scanned on the device, so scans stored offline and sent later
// still land in the right shift.
type ScanInput struct {
	ClientScanID   string
	NFCCode        string
	Condition      string
	Note           string
	Latitude       float64
	Longitude      float64
	ScannedAt      *time.Time
	FaceVerified   bool
	FaceMatchScore *float64
	Photos         []*multipart.FileHeader
	UserID         int64
	// UnitID is the unit of the officer; only points of this unit can be scanned.
	UnitID      *int64
	AppClientID int64
}

// Unit users only reach the groups, items and scans of their own unit; head
// office users reach every unit and can filter on one.
type PatrolService interface {
	// EnsureGroupAt returns the unit's patrol group containing t, creating it
	// from the unit's shift settings when it does not exist yet.
	EnsureGroupAt(unitID int64, t time.Time) (models.PatrolGroup, error)
	RunScheduler(ctx context.Context)
	// BackfillThumbnails creates missing export thumbnails of recent scan photos.
	BackfillThumbnails(ctx context.Context)

	GetGroups(actor Actor, filter dto.GroupFilter) ([]models.PatrolGroup, int64, error)
	GetGroup(actor Actor, id int64) (models.PatrolGroup, []models.PatrolListItem, error)
	// GetCurrentGroup returns the running group of the actor's unit; head office
	// users pass the unit.
	GetCurrentGroup(actor Actor, unitID int64) (models.PatrolGroup, []models.PatrolListItem, error)
	GetItems(actor Actor, filter dto.ItemFilter) ([]models.PatrolListItem, int64, error)

	// Scan records an NFC scan. duplicate is true when the same client_scan_id
	// was already stored (e.g. an offline sync retry) and the stored scan is returned.
	Scan(input ScanInput) (scan models.PatrolScan, duplicate bool, err error)
	GetScans(actor Actor, filter dto.ScanFilter) ([]models.PatrolScan, int64, error)
	GetScan(actor Actor, id int64) (models.PatrolScan, error)
	GetScanPhotoPath(actor Actor, scanID int64, photoID int64) (string, error)
	// ExportScans returns the scans for the Excel export, oldest first. With
	// withPhotos the photos are loaded too, and the stricter limits apply.
	ExportScans(actor Actor, filter dto.ScanFilter, withPhotos bool) ([]models.PatrolScan, error)
	FilterNames(filter dto.ScanFilter) (repository.FilterNames, error)
	// PointSummary returns the patrol total per patrol point of one shift in one
	// unit: of one group, or of a shift over a range of shift dates.
	PointSummary(actor Actor, filter dto.PointSummaryFilter) (dto.PointSummaryDTO, error)
}
