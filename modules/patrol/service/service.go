package service

import (
	"context"
	"errors"
	"fmt"
	"mime/multipart"
	"secure-patrol-backend/helper"
	"secure-patrol-backend/models"
	"secure-patrol-backend/modules/patrol/dto"
	"time"
)

const MaxScanPhotos = helper.MaxScanPhotos

// MaxExportRows keeps an export small enough to build in memory.
const MaxExportRows = 50000

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
	ErrDateRangeInvalid      = errors.New("date_from must not be after date_to")
)

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
	AppClientID    int64
}

type PatrolService interface {
	// EnsureGroupAt returns the patrol group containing t, creating it from the
	// shift settings when it does not exist yet.
	EnsureGroupAt(t time.Time) (models.PatrolGroup, error)
	RunScheduler(ctx context.Context)

	GetGroups(filter dto.GroupFilter) ([]models.PatrolGroup, int64, error)
	GetGroup(id int64) (models.PatrolGroup, []models.PatrolListItem, error)
	GetCurrentGroup() (models.PatrolGroup, []models.PatrolListItem, error)
	GetItems(filter dto.ItemFilter) ([]models.PatrolListItem, int64, error)

	// Scan records an NFC scan. duplicate is true when the same client_scan_id
	// was already stored (e.g. an offline sync retry) and the stored scan is returned.
	Scan(input ScanInput) (scan models.PatrolScan, duplicate bool, err error)
	GetScans(actor Actor, filter dto.ScanFilter) ([]models.PatrolScan, int64, error)
	GetScan(actor Actor, id int64) (models.PatrolScan, error)
	GetScanPhotoPath(actor Actor, scanID int64, photoID int64) (string, error)
	// ExportScans returns the scans for the Excel export, oldest first.
	ExportScans(actor Actor, filter dto.ScanFilter) ([]models.PatrolScan, error)
	FilterNames(filter dto.ScanFilter) (shift, point, user string, err error)
}
