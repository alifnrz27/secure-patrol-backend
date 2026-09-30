package repository

import (
	"secure-patrol-backend/models"
	"secure-patrol-backend/modules/patrol/dto"
	"time"
)

type PatrolRepository interface {
	FindActiveShifts() ([]models.PatrolShift, error)
	FindAllPatrolPoints() ([]models.PatrolPoint, error)
	FindPatrolPointByNFCCode(nfcCode string) (models.PatrolPoint, error)

	FindGroupContaining(t time.Time) (models.PatrolGroup, error)
	FindGroupByShiftDate(shiftID int64, shiftDate time.Time) (models.PatrolGroup, error)
	FindGroupByID(id int64) (models.PatrolGroup, error)
	FindGroups(filter dto.GroupFilter) ([]models.PatrolGroup, int64, error)
	// GroupBounds returns the latest group end at or before t and the earliest
	// group start after t, used to keep new groups from overlapping old ones.
	GroupBounds(t time.Time) (latestEnd *time.Time, earliestStart *time.Time, err error)
	CreateGroup(group *models.PatrolGroup, points []models.PatrolPoint) error
	SyncGroupItems(groupID int64, points []models.PatrolPoint) error

	FindItems(filter dto.ItemFilter) ([]models.PatrolListItem, int64, error)
	FindItemsByGroup(groupID int64) ([]models.PatrolListItem, error)
	FindItem(groupID int64, patrolPointID int64) (models.PatrolListItem, error)

	FindScanByClientID(userID int64, clientScanID string) (models.PatrolScan, error)
	FindScanByID(id int64) (models.PatrolScan, error)
	FindScans(filter dto.ScanFilter) ([]models.PatrolScan, int64, error)
	FindScansForExport(filter dto.ScanFilter, limit int) ([]models.PatrolScan, error)
	FilterNames(shiftID, patrolPointID, userID int64) (shift, point, user string, err error)
	CreateScan(scan *models.PatrolScan) error
}
