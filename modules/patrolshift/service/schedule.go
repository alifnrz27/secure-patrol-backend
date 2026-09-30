package service

import (
	"errors"
	"secure-patrol-backend/models"
	"strconv"
	"strings"
	"time"
)

const minutesPerDay = 24 * 60

var (
	ErrShiftTimeInvalid     = errors.New("time must use the HH:MM format between 00:00 and 24:00 (24:00 only as end time)")
	ErrShiftDurationInvalid = errors.New("start time and end time must be different")
)

// ParseClock converts "HH:MM" (00:00 - 24:00) to minutes since midnight.
func ParseClock(value string) (int, error) {
	value = strings.TrimSpace(value)
	if len(value) != 5 || value[2] != ':' {
		return 0, ErrShiftTimeInvalid
	}

	hour, errHour := strconv.Atoi(value[:2])
	minute, errMinute := strconv.Atoi(value[3:])
	if errHour != nil || errMinute != nil || hour < 0 || hour > 24 || minute < 0 || minute > 59 || (hour == 24 && minute != 0) {
		return 0, ErrShiftTimeInvalid
	}

	return hour*60 + minute, nil
}

// Bounds returns the start minute and duration of a shift. A shift whose end
// is not after its start crosses midnight, e.g. 22:00-06:00 lasts 8 hours.
func Bounds(shift models.PatrolShift) (start int, duration int, err error) {
	start, err = ParseClock(shift.StartTime)
	if err != nil {
		return 0, 0, err
	}
	end, err := ParseClock(shift.EndTime)
	if err != nil {
		return 0, 0, err
	}

	if start == minutesPerDay {
		return 0, 0, ErrShiftTimeInvalid
	}
	if start == end || (start == 0 && end == 0) {
		return 0, 0, ErrShiftDurationInvalid
	}

	if end > start {
		return start, end - start, nil
	}
	return start, end + minutesPerDay - start, nil
}

// Window returns when the shift starting on date (local midnight) starts and ends.
func Window(shift models.PatrolShift, date time.Time) (time.Time, time.Time, error) {
	start, duration, err := Bounds(shift)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}

	startAt := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location()).
		Add(time.Duration(start) * time.Minute)
	return startAt, startAt.Add(time.Duration(duration) * time.Minute), nil
}

// ResolvedShift is the shift occurrence that contains a moment.
type ResolvedShift struct {
	Shift   models.PatrolShift
	Date    time.Time // local midnight of the day the shift starts
	StartAt time.Time
	EndAt   time.Time
}

// Resolve finds the active shift containing t. Windows are [start, end): the
// end time is the cut-off, so a scan exactly at the end belongs to the next shift.
func Resolve(shifts []models.PatrolShift, t time.Time, loc *time.Location) (ResolvedShift, bool) {
	local := t.In(loc)
	today := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, loc)

	// A shift that crosses midnight started yesterday.
	for _, day := range []time.Time{today, today.AddDate(0, 0, -1)} {
		for _, shift := range shifts {
			if !shift.IsActive {
				continue
			}
			startAt, endAt, err := Window(shift, day)
			if err != nil {
				continue
			}
			if !t.Before(startAt) && t.Before(endAt) {
				return ResolvedShift{Shift: shift, Date: day, StartAt: startAt, EndAt: endAt}, true
			}
		}
	}

	return ResolvedShift{}, false
}

// Overlaps reports whether two shifts cover a common moment of the day.
func Overlaps(a, b models.PatrolShift) (bool, error) {
	ref := time.Date(2000, 1, 2, 0, 0, 0, 0, time.UTC)

	aStart, aEnd, err := Window(a, ref)
	if err != nil {
		return false, err
	}

	for _, offset := range []int{-1, 0, 1} {
		bStart, bEnd, err := Window(b, ref.AddDate(0, 0, offset))
		if err != nil {
			return false, err
		}
		if aStart.Before(bEnd) && bStart.Before(aEnd) {
			return true, nil
		}
	}

	return false, nil
}
