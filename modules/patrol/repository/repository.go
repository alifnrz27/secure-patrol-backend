package repository

import (
	"secure-patrol-backend/models"
	"secure-patrol-backend/modules/patrol/dto"
	"time"
)

// FilterNames are the names of the ids used in a filter, for the export.
type FilterNames struct {
	Unit  string
	Area  string
	Shift string
	Point string
	User  string
}

// Patrol groups belong to a unit: every unit has its own shifts, points and groups.
type PatrolRepository interface {
	FindActiveUnitIDs() ([]int64, error)
	UnitExists(unitID int64) (bool, error)
	FindActiveShifts(unitID int64) ([]models.PatrolShift, error)
	FindAllPatrolPoints(unitID int64) ([]models.PatrolPoint, error)
	FindPatrolPointByNFCCode(nfcCode string) (models.PatrolPoint, error)

	FindGroupContaining(unitID int64, t time.Time) (models.PatrolGroup, error)
	FindGroupByShiftDate(shiftID int64, shiftDate time.Time) (models.PatrolGroup, error)
	FindGroupByID(id int64) (models.PatrolGroup, error)
	FindGroups(filter dto.GroupFilter) ([]models.PatrolGroup, int64, error)
	// GroupBounds returns the latest group end at or before t and the earliest
	// group start after t, used to keep new groups from overlapping old ones.
	GroupBounds(unitID int64, t time.Time) (latestEnd *time.Time, earliestStart *time.Time, err error)
	CreateGroup(group *models.PatrolGroup, points []models.PatrolPoint) error
	SyncGroupItems(groupID int64, points []models.PatrolPoint) error

	FindItems(filter dto.ItemFilter) ([]models.PatrolListItem, int64, error)
	// FindItemsByGroup lists the patrol list of a group. With visibleTo > 0 the
	// points assigned only to other officers are left out.
	FindItemsByGroup(groupID int64, visibleTo int64) ([]models.PatrolListItem, error)
	FindItemByID(id int64) (models.PatrolListItem, error)
	// ReplaceAssignees sets the officers assigned to a patrol list item.
	ReplaceAssignees(itemID int64, userIDs []int64, assignedBy int64) error
	// FindAssignableUsers returns the active users among ids that belong to the
	// unit and have an assignable role.
	FindAssignableUsers(unitID int64, ids []int64) ([]models.User, error)
	FindItem(groupID int64, patrolPointID int64) (models.PatrolListItem, error)

	FindScanByClientID(userID int64, clientScanID string) (models.PatrolScan, error)
	FindScanByID(id int64) (models.PatrolScan, error)
	FindScans(filter dto.ScanFilter) ([]models.PatrolScan, int64, error)
	FindScansForExport(filter dto.ScanFilter, limit int, withPhotos bool) ([]models.PatrolScan, error)
	FilterNames(unitID, shiftID, patrolPointID, userID, areaID int64) (FilterNames, error)
	CreateScan(scan *models.PatrolScan) error

	// FindPhotoPathsSince lists scan photos received since t, newest first.
	FindPhotoPathsSince(t time.Time) ([]string, error)
	FindShift(id int64) (models.PatrolShift, error)
	FindUnit(id int64) (models.Unit, error)
	// PointSummary totals the scans per patrol point over the matching groups of one unit.
	PointSummary(unitID int64, filter dto.PointSummaryFilter) (rows []dto.PointSummaryRow, groups int64, err error)
}
