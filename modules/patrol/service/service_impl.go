package service

import (
	"context"
	"errors"
	"secure-patrol-backend/helper"
	"secure-patrol-backend/models"
	licenseservice "secure-patrol-backend/modules/license/service"
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

func (s *service) EnsureGroupAt(unitID int64, t time.Time) (models.PatrolGroup, error) {
	group, err := s.repo.FindGroupContaining(unitID, t)
	if err == nil {
		if time.Now().Before(group.EndAt) {
			if err := s.syncItems(group); err != nil {
				return group, err
			}
		}
		return group, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return group, err
	}

	shifts, err := s.repo.FindActiveShifts(unitID)
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
	latestEnd, earliestStart, err := s.repo.GroupBounds(unitID, t)
	if err != nil {
		return group, err
	}
	if latestEnd != nil && latestEnd.After(startAt) {
		startAt = *latestEnd
	}
	if earliestStart != nil && earliestStart.Before(endAt) {
		endAt = *earliestStart
	}

	points, err := s.repo.FindAllPatrolPoints(unitID)
	if err != nil {
		return group, err
	}

	group = models.PatrolGroup{
		UnitID:        unitID,
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

	log.Infof("patrol group %d created: unit %d %s %s", group.ID, unitID, group.ShiftName, shiftDate.Format("2006-01-02"))
	return s.repo.FindGroupByID(group.ID)
}

func (s *service) syncItems(group models.PatrolGroup) error {
	points, err := s.repo.FindAllPatrolPoints(group.UnitID)
	if err != nil {
		return err
	}
	return s.repo.SyncGroupItems(group.ID, points)
}

// RunScheduler creates the current patrol group of every active unit each
// minute, so the patrol list is ready before the first scan of a shift.
func (s *service) RunScheduler(ctx context.Context) {
	run := func() {
		unitIDs, err := s.repo.FindActiveUnitIDs()
		if err != nil {
			log.Errorf("patrol scheduler: %v", err)
			return
		}
		licenses := licenseservice.Instance()
		if licenses != nil && licenses.Status().Locked() {
			return
		}
		now := time.Now()
		for _, unitID := range unitIDs {
			if licenses != nil {
				if allowed, err := licenses.UnitAllowed(unitID); err != nil || !allowed {
					continue
				}
			}
			if _, err := s.EnsureGroupAt(unitID, now); err != nil && !errors.Is(err, ErrNoActiveShift) {
				log.Errorf("patrol scheduler: unit %d: %v", unitID, err)
			}
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

func (s *service) GetGroups(actor Actor, filter dto.GroupFilter) ([]models.PatrolGroup, int64, error) {
	filter.UnitID = actor.scope().UnitFilter(filter.UnitID)
	return s.repo.FindGroups(filter)
}

func (s *service) GetGroup(actor Actor, id int64) (models.PatrolGroup, []models.PatrolListItem, error) {
	group, err := s.repo.FindGroupByID(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return group, nil, ErrGroupNotFound
	}
	if err != nil {
		return group, nil, err
	}
	if !actor.scope().CanAccessUnit(group.UnitID) {
		return models.PatrolGroup{}, nil, ErrGroupNotFound
	}

	items, err := s.repo.FindItemsByGroup(group.ID)
	return group, items, err
}

func (s *service) GetCurrentGroup(actor Actor, unitID int64) (models.PatrolGroup, []models.PatrolListItem, error) {
	unitID = actor.scope().UnitFilter(unitID)
	if unitID <= 0 {
		return models.PatrolGroup{}, nil, ErrUnitRequired
	}
	if actor.UnitID == nil {
		exists, err := s.repo.UnitExists(unitID)
		if err != nil {
			return models.PatrolGroup{}, nil, err
		}
		if !exists {
			return models.PatrolGroup{}, nil, ErrUnitNotFound
		}
	}

	group, err := s.EnsureGroupAt(unitID, time.Now())
	if err != nil {
		return group, nil, err
	}
	return s.GetGroup(actor, group.ID)
}

func (s *service) GetItems(actor Actor, filter dto.ItemFilter) ([]models.PatrolListItem, int64, error) {
	filter.UnitID = actor.scope().UnitFilter(filter.UnitID)
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

	// Head office users do not patrol; officers only patrol their own unit.
	if input.UnitID == nil {
		return models.PatrolScan{}, false, ErrScanNeedsUnit
	}
	unitID := *input.UnitID
	if licenses := licenseservice.Instance(); licenses != nil {
		if licenses.Status().Locked() {
			return models.PatrolScan{}, false, licenseservice.ErrLicenseInactive
		}
		if allowed, err := licenses.UnitAllowed(unitID); err != nil || !allowed {
			return models.PatrolScan{}, false, licenseservice.ErrUnitOverLicense
		}
	}

	now := time.Now()
	settings := settingservice.ForUnit(input.UnitID)
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
	if point.UnitID != unitID {
		return models.PatrolScan{}, false, ErrPointOfOtherUnit
	}

	group, err := s.EnsureGroupAt(unitID, scannedAt)
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

	// Prepare the export thumbnails in the background, so a later Excel export
	// with photos does not have to decode the full photos.
	go makeThumbnails(photos)

	stored, err := s.repo.FindScanByID(scan.ID)
	return stored, false, err
}

func makeThumbnails(photos []models.PatrolScanPhoto) {
	for _, photo := range photos {
		if _, err := dto.ScanPhotoThumbnail(photo.Path); err != nil {
			log.Warnf("patrol: thumbnail of %s: %v", photo.Path, err)
		}
	}
}

// thumbnailBackfillDays covers photos uploaded before thumbnails were cached.
const thumbnailBackfillDays = 90

// BackfillThumbnails creates missing export thumbnails of recent photos, one
// at a time so it does not slow down the server.
func (s *service) BackfillThumbnails(ctx context.Context) {
	paths, err := s.repo.FindPhotoPathsSince(time.Now().AddDate(0, 0, -thumbnailBackfillDays))
	if err != nil {
		log.Errorf("patrol: thumbnail backfill: %v", err)
		return
	}
	created := 0
	for _, path := range paths {
		if ctx.Err() != nil {
			return
		}
		if dto.HasScanPhotoThumbnail(path) {
			continue
		}
		if _, err := dto.ScanPhotoThumbnail(path); err == nil {
			created++
		}
	}
	if created > 0 {
		log.Infof("patrol: created %d export thumbnail(s) for older photos", created)
	}
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
	filter.UnitID = actor.scope().UnitFilter(filter.UnitID)
	if !canSeeAllScans(actor) {
		filter.ScannedBy = actor.UserID
	}
	return s.repo.FindScans(filter)
}

func (s *service) ExportScans(actor Actor, filter dto.ScanFilter, withPhotos bool) ([]models.PatrolScan, error) {
	if filter.DateFrom == "" || filter.DateTo == "" {
		return nil, ErrExportRangeRequired
	}
	from, errFrom := time.Parse("2006-01-02", filter.DateFrom)
	to, errTo := time.Parse("2006-01-02", filter.DateTo)
	if errFrom != nil || errTo != nil || from.After(to) {
		return nil, ErrDateRangeInvalid
	}

	// Both dates are included: 2026-10-01 to 2026-10-07 is 7 days.
	days := int(to.Sub(from).Hours()/24) + 1
	settings := settingservice.ForUnit(actor.UnitID)
	maxDays, maxRows, tooLarge := settings.ExportMaxRangeDays, MaxExportRows, ErrExportTooLarge
	if withPhotos {
		maxDays, maxRows, tooLarge = settings.ExportPhotoMaxRangeDays, MaxExportPhotoRows, ErrExportPhotosTooLarge
	}
	if days > maxDays {
		return nil, &ExportRangeError{MaxDays: maxDays, WithPhotos: withPhotos}
	}

	filter.UnitID = actor.scope().UnitFilter(filter.UnitID)
	if !canSeeAllScans(actor) {
		filter.ScannedBy = actor.UserID
	}

	scans, err := s.repo.FindScansForExport(filter, maxRows+1, withPhotos)
	if err != nil {
		return nil, err
	}
	if len(scans) > maxRows {
		return nil, tooLarge
	}
	return scans, nil
}

func (s *service) FilterNames(filter dto.ScanFilter) (repository.FilterNames, error) {
	return s.repo.FilterNames(filter.UnitID, filter.ShiftID, filter.PatrolPointID, filter.ScannedBy, filter.AreaID)
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
	if !actor.scope().CanAccessUnit(scan.PatrolGroup.UnitID) {
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

func (s *service) PointSummary(actor Actor, filter dto.PointSummaryFilter) (dto.PointSummaryDTO, error) {
	var result dto.PointSummaryDTO
	if filter.DateFrom != "" && filter.DateTo != "" && filter.DateFrom > filter.DateTo {
		return result, ErrDateRangeInvalid
	}

	var unitID int64
	switch {
	case filter.GroupID > 0:
		group, _, err := s.GetGroup(actor, filter.GroupID)
		if err != nil {
			return result, err
		}
		unitID = group.UnitID
		filter.ShiftID = group.PatrolShiftID
		filter.DateFrom = group.ShiftDate.Format("2006-01-02")
		filter.DateTo = filter.DateFrom
		result.GroupID = &group.ID
	case filter.ShiftID > 0:
		shift, err := s.repo.FindShift(filter.ShiftID)
		if errors.Is(err, gorm.ErrRecordNotFound) || (err == nil && !actor.scope().CanAccessUnit(shift.UnitID)) {
			return result, ErrShiftNotFound
		}
		if err != nil {
			return result, err
		}
		unitID = shift.UnitID
	default:
		return result, ErrSummaryTargetMissing
	}

	shift, err := s.repo.FindShift(filter.ShiftID)
	if err != nil {
		return result, err
	}
	unit, err := s.repo.FindUnit(unitID)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return result, err
	}

	rows, groups, err := s.repo.PointSummary(unitID, filter)
	if err != nil {
		return result, err
	}

	result.Unit = dto.UnitSummaryDTO{ID: unitID, Code: unit.Code, Name: unit.Name}
	result.Shift = dto.ShiftSummaryDTO{ID: shift.ID, Name: shift.Name}
	result.DateFrom, result.DateTo = filter.DateFrom, filter.DateTo
	if filter.AreaID > 0 {
		result.AreaID = &filter.AreaID
	}
	result.Groups = groups
	result.Items = rows
	for _, row := range rows {
		result.Totals.Points++
		if row.TotalScans > 0 {
			result.Totals.ScannedPoints++
		}
		result.Totals.TotalScans += row.TotalScans
		result.Totals.AbnormalScans += row.AbnormalScans
	}
	result.Totals.UnscannedPoints = result.Totals.Points - result.Totals.ScannedPoints
	return result, nil
}
