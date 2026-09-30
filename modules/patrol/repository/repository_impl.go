package repository

import (
	"secure-patrol-backend/models"
	"secure-patrol-backend/modules/patrol/dto"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type repository struct {
	db *gorm.DB
}

func NewPatrolRepository(db *gorm.DB) PatrolRepository {
	return &repository{db: db}
}

func (r *repository) FindActiveUnitIDs() (ids []int64, err error) {
	err = r.db.Model(&models.Unit{}).Where("is_active = ?", true).Order("id ASC").Pluck("id", &ids).Error
	return ids, err
}

func (r *repository) UnitExists(unitID int64) (bool, error) {
	var count int64
	err := r.db.Model(&models.Unit{}).Where("id = ?", unitID).Count(&count).Error
	return count > 0, err
}

func (r *repository) FindActiveShifts(unitID int64) (shifts []models.PatrolShift, err error) {
	err = r.db.Where("unit_id = ? AND is_active = ?", unitID, true).Order("start_time ASC").Find(&shifts).Error
	return shifts, err
}

func (r *repository) FindAllPatrolPoints(unitID int64) (points []models.PatrolPoint, err error) {
	err = r.db.Where("unit_id = ?", unitID).Order("name ASC").Find(&points).Error
	return points, err
}

func (r *repository) FindPatrolPointByNFCCode(nfcCode string) (point models.PatrolPoint, err error) {
	err = r.db.Where("nfc_code = ?", nfcCode).First(&point).Error
	return point, err
}

// groupColumns selects a group with its unit's code and name (also of deleted
// units, so history keeps them).
const groupColumns = `patrol_groups.*,
		(SELECT u.code FROM units u WHERE u.id = patrol_groups.unit_id) AS unit_code,
		(SELECT u.name FROM units u WHERE u.id = patrol_groups.unit_id) AS unit_name`

func withUnit(db *gorm.DB) *gorm.DB {
	return db.Select(groupColumns)
}

// groupsWithProgress selects groups together with their unit and checklist progress.
func (r *repository) groupsWithProgress() *gorm.DB {
	return r.db.Model(&models.PatrolGroup{}).Select(groupColumns + `,
		(SELECT COUNT(*) FROM patrol_list_items i WHERE i.patrol_group_id = patrol_groups.id AND i.deleted_at IS NULL) AS total_points,
		(SELECT COUNT(*) FROM patrol_list_items i WHERE i.patrol_group_id = patrol_groups.id AND i.deleted_at IS NULL AND i.scan_count > 0) AS scanned_points,
		(SELECT COUNT(*) FROM patrol_scans s WHERE s.patrol_group_id = patrol_groups.id AND s.deleted_at IS NULL) AS total_scans,
		(SELECT COUNT(*) FROM patrol_scans s WHERE s.patrol_group_id = patrol_groups.id AND s.deleted_at IS NULL AND s.condition = 'abnormal') AS abnormal_scans`)
}

func (r *repository) FindGroupContaining(unitID int64, t time.Time) (group models.PatrolGroup, err error) {
	err = r.groupsWithProgress().
		Where("patrol_groups.unit_id = ? AND patrol_groups.start_at <= ? AND patrol_groups.end_at > ?", unitID, t, t).
		Order("patrol_groups.start_at DESC").
		First(&group).Error
	return group, err
}

func (r *repository) FindGroupByShiftDate(shiftID int64, shiftDate time.Time) (group models.PatrolGroup, err error) {
	err = r.groupsWithProgress().
		Where("patrol_groups.patrol_shift_id = ? AND patrol_groups.shift_date = ?", shiftID, shiftDate.Format("2006-01-02")).
		First(&group).Error
	return group, err
}

func (r *repository) FindGroupByID(id int64) (group models.PatrolGroup, err error) {
	err = r.groupsWithProgress().Where("patrol_groups.id = ?", id).First(&group).Error
	return group, err
}

func applyGroupFilter(query *gorm.DB, table string, unitID, shiftID int64, dateFrom, dateTo string) *gorm.DB {
	if unitID > 0 {
		query = query.Where(table+".unit_id = ?", unitID)
	}
	if shiftID > 0 {
		query = query.Where(table+".patrol_shift_id = ?", shiftID)
	}
	if dateFrom != "" {
		query = query.Where(table+".shift_date >= ?", dateFrom)
	}
	if dateTo != "" {
		query = query.Where(table+".shift_date <= ?", dateTo)
	}
	return query
}

func (r *repository) FindGroups(filter dto.GroupFilter) (groups []models.PatrolGroup, total int64, err error) {
	count := applyGroupFilter(r.db.Model(&models.PatrolGroup{}), "patrol_groups", filter.UnitID, filter.ShiftID, filter.DateFrom, filter.DateTo)
	if err = count.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err = applyGroupFilter(r.groupsWithProgress(), "patrol_groups", filter.UnitID, filter.ShiftID, filter.DateFrom, filter.DateTo).
		Order("patrol_groups.start_at DESC").
		Limit(filter.Limit).
		Offset(filter.Offset()).
		Find(&groups).Error

	return groups, total, err
}

func (r *repository) GroupBounds(unitID int64, t time.Time) (latestEnd *time.Time, earliestStart *time.Time, err error) {
	var bounds struct {
		LatestEnd     *time.Time
		EarliestStart *time.Time
	}

	err = r.db.Raw(`SELECT
		(SELECT MAX(end_at) FROM patrol_groups WHERE unit_id = ? AND end_at <= ? AND deleted_at IS NULL) AS latest_end,
		(SELECT MIN(start_at) FROM patrol_groups WHERE unit_id = ? AND start_at > ? AND deleted_at IS NULL) AS earliest_start`,
		unitID, t, unitID, t).
		Scan(&bounds).Error

	return bounds.LatestEnd, bounds.EarliestStart, err
}

func listItemsFromPoints(groupID int64, points []models.PatrolPoint) []models.PatrolListItem {
	items := make([]models.PatrolListItem, 0, len(points))
	for _, point := range points {
		items = append(items, models.PatrolListItem{
			PatrolGroupID:            groupID,
			PatrolPointID:            point.ID,
			Name:                     point.Name,
			Location:                 point.Location,
			NFCCode:                  point.NFCCode,
			Latitude:                 point.Latitude,
			Longitude:                point.Longitude,
			IsLocationMatchRequired:  point.IsLocationMatchRequired,
			IsFaceValidationRequired: point.IsFaceValidationRequired,
		})
	}
	return items
}

// CreateGroup creates the group and copies the patrol points into its patrol list.
func (r *repository) CreateGroup(group *models.PatrolGroup, points []models.PatrolPoint) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Omit(clause.Associations).Create(group).Error; err != nil {
			return err
		}

		items := listItemsFromPoints(group.ID, points)
		if len(items) == 0 {
			return nil
		}
		return tx.Omit(clause.Associations).Create(&items).Error
	})
}

// SyncGroupItems adds patrol points missing from the list and refreshes the
// copied data of existing items. It is only called while a group is running,
// so finished groups stay frozen.
func (r *repository) SyncGroupItems(groupID int64, points []models.PatrolPoint) error {
	items := listItemsFromPoints(groupID, points)
	if len(items) == 0 {
		return nil
	}

	return r.db.Omit(clause.Associations).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "patrol_group_id"}, {Name: "patrol_point_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"name", "location", "nfc_code", "latitude", "longitude",
			"is_location_match_required", "is_face_validation_required", "updated_at",
		}),
	}).Create(&items).Error
}

func (r *repository) FindItems(filter dto.ItemFilter) (items []models.PatrolListItem, total int64, err error) {
	query := r.db.Model(&models.PatrolListItem{}).
		Joins("JOIN patrol_groups ON patrol_groups.id = patrol_list_items.patrol_group_id AND patrol_groups.deleted_at IS NULL")
	query = applyGroupFilter(query, "patrol_groups", filter.UnitID, filter.ShiftID, filter.DateFrom, filter.DateTo)

	if filter.GroupID > 0 {
		query = query.Where("patrol_list_items.patrol_group_id = ?", filter.GroupID)
	}
	switch filter.Status {
	case "scanned":
		query = query.Where("patrol_list_items.scan_count > 0")
	case "unscanned":
		query = query.Where("patrol_list_items.scan_count = 0")
	}
	if filter.Search != "" {
		like := "%" + filter.Search + "%"
		query = query.Where(
			"patrol_list_items.name ILIKE ? OR patrol_list_items.location ILIKE ? OR patrol_list_items.nfc_code ILIKE ?",
			like, like, like,
		)
	}

	if err = query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err = query.
		Preload("PatrolGroup", withUnit).
		Preload("LastScannedByUser", unscoped).
		Order("patrol_groups.start_at DESC, patrol_list_items.name ASC").
		Limit(filter.Limit).
		Offset(filter.Offset()).
		Find(&items).Error

	return items, total, err
}

func (r *repository) FindItemsByGroup(groupID int64) (items []models.PatrolListItem, err error) {
	err = r.db.Preload("LastScannedByUser", unscoped).
		Where("patrol_group_id = ?", groupID).
		Order("name ASC").
		Find(&items).Error
	return items, err
}

func (r *repository) FindItem(groupID int64, patrolPointID int64) (item models.PatrolListItem, err error) {
	err = r.db.Where("patrol_group_id = ? AND patrol_point_id = ?", groupID, patrolPointID).First(&item).Error
	return item, err
}

// unscoped keeps soft deleted users visible in history (who scanned stays known
// after the account is deleted).
func unscoped(db *gorm.DB) *gorm.DB {
	return db.Unscoped()
}

func (r *repository) scanQuery() *gorm.DB {
	return r.db.Model(&models.PatrolScan{}).
		Preload("ScannedByUser", unscoped).
		Preload("PatrolGroup", withUnit).
		Preload("PatrolListItem").
		Preload("Photos", func(db *gorm.DB) *gorm.DB { return db.Order("sort_order ASC") })
}

func (r *repository) FindScanByClientID(userID int64, clientScanID string) (scan models.PatrolScan, err error) {
	err = r.scanQuery().
		Where("patrol_scans.scanned_by = ? AND patrol_scans.client_scan_id = ?", userID, clientScanID).
		First(&scan).Error
	return scan, err
}

func (r *repository) FindScanByID(id int64) (scan models.PatrolScan, err error) {
	err = r.scanQuery().Where("patrol_scans.id = ?", id).First(&scan).Error
	return scan, err
}

func (r *repository) scanFilterQuery(filter dto.ScanFilter) *gorm.DB {
	query := r.db.Model(&models.PatrolScan{}).
		Joins("JOIN patrol_groups ON patrol_groups.id = patrol_scans.patrol_group_id AND patrol_groups.deleted_at IS NULL")
	query = applyGroupFilter(query, "patrol_groups", filter.UnitID, filter.ShiftID, filter.DateFrom, filter.DateTo)

	if filter.GroupID > 0 {
		query = query.Where("patrol_scans.patrol_group_id = ?", filter.GroupID)
	}
	if filter.ScannedBy > 0 {
		query = query.Where("patrol_scans.scanned_by = ?", filter.ScannedBy)
	}
	if filter.PatrolPointID > 0 {
		query = query.Where("patrol_scans.patrol_point_id = ?", filter.PatrolPointID)
	}
	if filter.Condition != "" {
		query = query.Where("patrol_scans.condition = ?", filter.Condition)
	}
	return query
}

// FindScansForExport returns up to limit matching scans, oldest first.
func (r *repository) FindScansForExport(filter dto.ScanFilter, limit int) (scans []models.PatrolScan, err error) {
	var ids []int64
	err = r.scanFilterQuery(filter).
		Order("patrol_scans.scanned_at ASC, patrol_scans.id ASC").
		Limit(limit).
		Pluck("patrol_scans.id", &ids).Error
	if err != nil || len(ids) == 0 {
		return []models.PatrolScan{}, err
	}

	// Photos are not part of the export.
	err = r.db.Model(&models.PatrolScan{}).
		Preload("ScannedByUser", unscoped).
		Preload("PatrolGroup", withUnit).
		Preload("PatrolListItem").
		Where("patrol_scans.id IN ?", ids).
		Order("patrol_scans.scanned_at ASC, patrol_scans.id ASC").
		Find(&scans).Error
	return scans, err
}

// FilterNames resolves the ids used in an export filter to names, including
// deleted records, for the filter summary sheet.
func (r *repository) FilterNames(unitID, shiftID, patrolPointID, userID int64) (names FilterNames, err error) {
	lookup := func(model interface{}, id int64, column string, out *string) error {
		if id <= 0 {
			return nil
		}
		return r.db.Unscoped().Model(model).Where("id = ?", id).Select(column).Scan(out).Error
	}
	if err = lookup(&models.Unit{}, unitID, "name", &names.Unit); err != nil {
		return
	}
	if err = lookup(&models.PatrolShift{}, shiftID, "name", &names.Shift); err != nil {
		return
	}
	if err = lookup(&models.PatrolPoint{}, patrolPointID, "name", &names.Point); err != nil {
		return
	}
	err = lookup(&models.User{}, userID, "name", &names.User)
	return
}

func (r *repository) FindScans(filter dto.ScanFilter) (scans []models.PatrolScan, total int64, err error) {
	query := r.scanFilterQuery(filter)

	if err = query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var ids []int64
	err = query.
		Order("patrol_scans.scanned_at DESC, patrol_scans.id DESC").
		Limit(filter.Limit).
		Offset(filter.Offset()).
		Pluck("patrol_scans.id", &ids).Error
	if err != nil || len(ids) == 0 {
		return []models.PatrolScan{}, total, err
	}

	err = r.scanQuery().
		Where("patrol_scans.id IN ?", ids).
		Order("patrol_scans.scanned_at DESC, patrol_scans.id DESC").
		Find(&scans).Error

	return scans, total, err
}

// CreateScan stores the scan with its photos and updates the list item summary.
// Offline scans can arrive out of order, so the "last scan" fields only move
// forward in time.
func (r *repository) CreateScan(scan *models.PatrolScan) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Omit("ScannedByUser", "PatrolGroup", "PatrolListItem").Create(scan).Error; err != nil {
			return err
		}

		isLatest := "(last_scanned_at IS NULL OR last_scanned_at <= ?)"
		return tx.Model(&models.PatrolListItem{}).
			Where("id = ?", scan.PatrolListItemID).
			Updates(map[string]interface{}{
				"scan_count":      gorm.Expr("scan_count + 1"),
				"last_condition":  gorm.Expr("CASE WHEN "+isLatest+" THEN ? ELSE last_condition END", scan.ScannedAt, scan.Condition),
				"last_scanned_by": gorm.Expr("CASE WHEN "+isLatest+" THEN ? ELSE last_scanned_by END", scan.ScannedAt, scan.ScannedBy),
				"last_scanned_at": gorm.Expr("CASE WHEN "+isLatest+" THEN ? ELSE last_scanned_at END", scan.ScannedAt, scan.ScannedAt),
				"updated_at":      time.Now(),
			}).Error
	})
}
