package dto

import "secure-patrol-backend/helper"

// Dates are YYYY-MM-DD and filter on the patrol group's shift date. UnitID 0
// means every unit; the service always sets it to the unit of a unit user.

type GroupFilter struct {
	helper.Pagination
	UnitID   int64
	ShiftID  int64
	DateFrom string
	DateTo   string
}

type ItemFilter struct {
	helper.Pagination
	UnitID   int64
	GroupID  int64
	ShiftID  int64
	DateFrom string
	DateTo   string
	Status   string // scanned | unscanned
}

type ScanFilter struct {
	helper.Pagination
	UnitID        int64
	GroupID       int64
	ShiftID       int64
	DateFrom      string
	DateTo        string
	ScannedBy     int64
	PatrolPointID int64
	Condition     string
}
