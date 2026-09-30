package dto

import "secure-patrol-backend/helper"

// Dates are YYYY-MM-DD and filter on the patrol group's shift date.

type GroupFilter struct {
	helper.Pagination
	ShiftID  int64
	DateFrom string
	DateTo   string
}

type ItemFilter struct {
	helper.Pagination
	GroupID  int64
	ShiftID  int64
	DateFrom string
	DateTo   string
	Status   string // scanned | unscanned
}

type ScanFilter struct {
	helper.Pagination
	GroupID       int64
	ShiftID       int64
	DateFrom      string
	DateTo        string
	ScannedBy     int64
	PatrolPointID int64
	Condition     string
}
