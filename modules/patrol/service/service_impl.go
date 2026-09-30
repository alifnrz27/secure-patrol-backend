package service

import (
	"context"
	"errors"
	"secure-patrol-backend/helper"
	"secure-patrol-backend/models"
	"secure-patrol-backend/modules/patrol/dto"
	"secure-patrol-backend/modules/patrol/repository"
	patrolpointservice "secure-patrol-backend/modules/patrolpoint/service"
	shiftservice "secure-patrol-backend/modules/patrolshift/service"
	settingservice "secure-patrol-backend/modules/setting/service"
	"secure-patrol-backend/pkg/log"
	"strings"
	"time"

	"gorm.io/gorm"
)

const (
	scanPhotoDir = "patrol-scans"
	// Tolerance for device clocks running slightly ahead of the server.
	maxClockSkew = 5 * time.Minute
)

type service struct {
	repo repository.PatrolRepository
}

func NewPatrolService(repo repository.PatrolRepository) PatrolService {
	return &service{repo: repo}
}

func (s *service) EnsureGroupAt(t time.Time) (models.PatrolGroup, error) {
	group, err := s.repo.FindGroupContaining(t)
	if err == nil {
		if time.Now().Before(group.EndAt) {
			if err := s.syncItems(group.ID); err != nil {
				return group, err
			}
		}
		return group, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return group, err
	}

	shifts, err := s.repo.FindActiveShifts()
	if err != nil {
		return group, err
	}

	resolved, ok := shiftservice.Resolve(shifts, t, helper.AppLocation())
	if !ok {
		return group, ErrNoActiveShift
	}

	shiftDate := time.Date(resolved.Date.Year(), resolved.Date.Month(), resolved.Date.Day(), 0, 0, 0, 0, time.UTC)

	// The group exists but was shortened because the shift times changed after it was created.
	if existing, err := s.repo.FindGroupByShiftDate(resolved.Shift.ID, shiftDate); err == nil {
		return existing, nil
	}

	// Never overlap groups created with older shift settings.
	startAt, endAt := resolved.StartAt, resolved.EndAt
	latestEnd, earliestStart, err := s.repo.GroupBounds(t)
	if err != nil {
		return group, err
	}
	if latestEnd != nil && latestEnd.After(startAt) {
		startAt = *latestEnd
	}
	if earliestStart != nil && earliestStart.Before(endAt) {
		endAt = *earliestStart
	}

	points, err := s.repo.FindAllPatrolPoints()
	if err != nil {
		return group, err
	}

	group = models.PatrolGroup{
		PatrolShiftID: resolved.Shift.ID,
		ShiftName:     resolved.Shift.Name,
		ShiftDate:     shiftDate,
		StartAt:       startAt,
		EndAt:         endAt,
	}

	if err := s.repo.CreateGroup(&group, points); err != nil {
		// Another request created it at the same time.
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return s.repo.FindGroupByShiftDate(resolved.Shift.ID, shiftDate)
		}
		return group, err
	}

	log.Infof("patrol group %d created: %s %s", group.ID, group.ShiftName, shiftDate.Format("2006-01-02"))
	return s.repo.FindGroupByID(group.ID)
}

func (s *service) syncItems(groupID int64) error {
	points, err := s.repo.FindAllPatrolPoints()
	if err != nil {
		return err
	}
	return s.repo.SyncGroupItems(groupID, points)
}

// RunScheduler creates the current patrol group every minute so the patrol
// list is ready before the first scan of a shift.
func (s *service) RunScheduler(ctx context.Context) {
	run := func() {
		if _, err := s.EnsureGroupAt(time.Now()); err != nil && !errors.Is(err, ErrNoActiveShift) {
			log.Errorf("patrol scheduler: %v", err)
		}
	}

	run()
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			run()
		}
	}
}

func (s *service) GetGroups(filter dto.GroupFilter) ([]models.PatrolGroup, int64, error) {
	return s.repo.FindGroups(filter)
}

func (s *service) GetGroup(id int64) (models.PatrolGroup, []models.PatrolListItem, error) {
	group, err := s.repo.FindGroupByID(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return group, nil, ErrGroupNotFound
	}
	if err != nil {
		return group, nil, err
	}

	items, err := s.repo.FindItemsByGroup(group.ID)
	return group, items, err
}

func (s *service) GetCurrentGroup() (models.PatrolGroup, []models.PatrolListItem, error) {
	group, err := s.EnsureGroupAt(time.Now())
	if err != nil {
		return group, nil, err
	}
	return s.GetGroup(group.ID)
}

func (s *service) GetItems(filter dto.ItemFilter) ([]models.PatrolListItem, int64, error) {
	return s.repo.FindItems(filter)
}

func (s *service) Scan(input ScanInput) (models.PatrolScan, bool, error) {
	input.ClientScanID = strings.TrimSpace(input.ClientScanID)
	input.Note = strings.TrimSpace(input.Note)

	// Offline sync retries send the same client_scan_id again.
	if existing, err := s.repo.FindScanByClientID(input.UserID, input.ClientScanID); err == nil {
		return existing, true, nil
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return existing, false, err
	}

	now := time.Now()
	settings := settingservice.Current()
	scannedAt := now
	if input.ScannedAt != nil {
		scannedAt = *input.ScannedAt
		if scannedAt.After(now.Add(maxClockSkew)) {
			return models.PatrolScan{}, false, ErrScannedAtInFuture
		}
		if now.Sub(scannedAt) > settings.PatrolMaxOfflineAge() {
			return models.PatrolScan{}, false, ErrScanTooOld
		}
	}

	if input.Condition == models.PatrolConditionAbnormal && input.Note == "" {
		return models.PatrolScan{}, false, ErrNoteRequired
	}
	if len(input.Photos) > MaxScanPhotos {
		return models.PatrolScan{}, false, ErrTooManyPhotos
	}

	point, err := s.repo.FindPatrolPointByNFCCode(patrolpointservice.NormalizeNFCCode(input.NFCCode))
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return models.PatrolScan{}, false, ErrNFCNotRegistered
	}
	if err != nil {
		return models.PatrolScan{}, false, err
	}

	group, err := s.EnsureGroupAt(scannedAt)
	if err != nil {
		return models.PatrolScan{}, false, err
	}

	item, err := s.repo.FindItem(group.ID, point.ID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		// The point was added after the group was last synced.
		if err := s.repo.SyncGroupItems(group.ID, []models.PatrolPoint{point}); err != nil {
			return models.PatrolScan{}, false, err
		}
		item, err = s.repo.FindItem(group.ID, point.ID)
	}
	if err != nil {
		return models.PatrolScan{}, false, err
	}

	radius := settings.PatrolLocationRadiusMeters
	distance := helper.DistanceMeters(item.Latitude, item.Longitude, input.Latitude, input.Longitude)
	if item.IsLocationMatchRequired && distance > radius {
		return models.PatrolScan{}, false, &LocationOutOfRangeError{Distance: distance, MaxDistance: radius}
	}

	// Face matching runs on the device; the server enforces the reported result.
	if item.IsFaceValidationRequired {
		if !input.FaceVerified {
			return models.PatrolScan{}, false, ErrFaceNotVerified
		}
		if minScore := settings.FaceMatchMinScore; minScore > 0 {
			if input.FaceMatchScore == nil {
				return models.PatrolScan{}, false, ErrFaceMatchScoreMissing
			}
			if *input.FaceMatchScore < minScore {
				return models.PatrolScan{}, false, ErrFaceMatchScoreTooLow
			}
		}
	}

	// Photos are stored last, once the scan is known to be valid.
	var photos []models.PatrolScanPhoto
	for i, file := range input.Photos {
		path, err := helper.SaveImage(file, scanPhotoDir)
		if err != nil {
			deletePhotos(photos)
			return models.PatrolScan{}, false, err
		}
		photos = append(photos, models.PatrolScanPhoto{Path: path, SortOrder: i + 1})
	}

	scan := models.PatrolScan{
		ClientScanID:     input.ClientScanID,
		ScannedBy:        input.UserID,
		PatrolGroupID:    group.ID,
		PatrolListItemID: item.ID,
		PatrolPointID:    point.ID,
		AppClientID:      input.AppClientID,
		NFCCode:          item.NFCCode,
		Condition:        input.Condition,
		Note:             input.Note,
		Latitude:         input.Latitude,
		Longitude:        input.Longitude,
		DistanceMeters:   distance,
		IsLocationValid:  distance <= radius,
		IsFaceVerified:   input.FaceVerified,
		FaceMatchScore:   input.FaceMatchScore,
		ScannedAt:        scannedAt,
		ReceivedAt:       now,
		Photos:           photos,
	}

	if err := s.repo.CreateScan(&scan); err != nil {
		deletePhotos(photos)
		// Two retries of the same offline scan arrived at the same time.
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			existing, findErr := s.repo.FindScanByClientID(input.UserID, input.ClientScanID)
			if findErr == nil {
				return existing, true, nil
			}
		}
		return models.PatrolScan{}, false, err
	}

	stored, err := s.repo.FindScanByID(scan.ID)
	return stored, false, err
}

func deletePhotos(photos []models.PatrolScanPhoto) {
	for _, photo := range photos {
		helper.DeleteStoredFile(photo.Path)
	}
}

// canSeeAllScans: security team members only see their own scans.
func canSeeAllScans(actor Actor) bool {
	return actor.RoleCode != models.RoleSecurityTeam
}

func (s *service) GetScans(actor Actor, filter dto.ScanFilter) ([]models.PatrolScan, int64, error) {
	if !canSeeAllScans(actor) {
		filter.ScannedBy = actor.UserID
	}
	return s.repo.FindScans(filter)
}

func (s *service) ExportScans(actor Actor, filter dto.ScanFilter) ([]models.PatrolScan, error) {
	if filter.DateFrom != "" && filter.DateTo != "" && filter.DateFrom > filter.DateTo {
		return nil, ErrDateRangeInvalid
	}
	if !canSeeAllScans(actor) {
		filter.ScannedBy = actor.UserID
	}

	scans, err := s.repo.FindScansForExport(filter, MaxExportRows+1)
	if err != nil {
		return nil, err
	}
	if len(scans) > MaxExportRows {
		return nil, ErrExportTooLarge
	}
	return scans, nil
}

func (s *service) FilterNames(filter dto.ScanFilter) (string, string, string, error) {
	return s.repo.FilterNames(filter.ShiftID, filter.PatrolPointID, filter.ScannedBy)
}

func (s *service) GetScan(actor Actor, id int64) (models.PatrolScan, error) {
	scan, err := s.repo.FindScanByID(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return scan, ErrScanNotFound
	}
	if err != nil {
		return scan, err
	}

	if !canSeeAllScans(actor) && scan.ScannedBy != actor.UserID {
		return models.PatrolScan{}, ErrScanNotFound
	}
	return scan, nil
}

func (s *service) GetScanPhotoPath(actor Actor, scanID int64, photoID int64) (string, error) {
	scan, err := s.GetScan(actor, scanID)
	if err != nil {
		return "", err
	}

	for _, photo := range scan.Photos {
		if photo.ID == photoID {
			return helper.StoragePath(photo.Path)
		}
	}
	return "", ErrScanNotFound
}
