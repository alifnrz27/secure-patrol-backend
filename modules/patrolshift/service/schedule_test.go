package service

import (
	"testing"
	"time"

	"secure-patrol-backend/models"
)

var jakarta = time.FixedZone("WIB", 7*60*60)

func shift(name, start, end string) models.PatrolShift {
	return models.PatrolShift{Name: name, StartTime: start, EndTime: end, IsActive: true}
}

var defaultShifts = []models.PatrolShift{
	shift("Shift 1", "08:00", "16:00"),
	shift("Shift 2", "16:00", "24:00"),
	shift("Shift 3", "00:00", "08:00"),
}

func at(day, hour, minute, second int) time.Time {
	return time.Date(2026, 9, day, hour, minute, second, 0, jakarta)
}

func TestResolveCutOffMovesToNextShift(t *testing.T) {
	cases := []struct {
		at        time.Time
		wantShift string
		wantDay   int
	}{
		{at(29, 8, 0, 0), "Shift 1", 29},
		{at(29, 15, 59, 59), "Shift 1", 29},
		{at(29, 16, 0, 0), "Shift 2", 29}, // cut-off of Shift 1
		{at(29, 23, 59, 59), "Shift 2", 29},
		{at(30, 0, 0, 0), "Shift 3", 30}, // cut-off of Shift 2
		{at(30, 7, 59, 59), "Shift 3", 30},
		{at(30, 8, 0, 0), "Shift 1", 30},
	}

	for _, c := range cases {
		got, ok := Resolve(defaultShifts, c.at, jakarta)
		if !ok || got.Shift.Name != c.wantShift || got.Date.Day() != c.wantDay {
			t.Errorf("%s: got %q day %d (ok=%v), want %q day %d", c.at.Format(time.RFC3339), got.Shift.Name, got.Date.Day(), ok, c.wantShift, c.wantDay)
		}
	}
}

func TestResolveShiftCrossingMidnightBelongsToStartDay(t *testing.T) {
	shifts := []models.PatrolShift{shift("Day", "06:00", "22:00"), shift("Night", "22:00", "06:00")}

	got, ok := Resolve(shifts, at(30, 2, 30, 0), jakarta)
	if !ok || got.Shift.Name != "Night" || got.Date.Day() != 29 {
		t.Fatalf("got %q day %d, want Night day 29", got.Shift.Name, got.Date.Day())
	}
	if !got.StartAt.Equal(at(29, 22, 0, 0)) || !got.EndAt.Equal(at(30, 6, 0, 0)) {
		t.Fatalf("window %s - %s", got.StartAt, got.EndAt)
	}
}

func TestResolveGapAndInactiveShift(t *testing.T) {
	shifts := []models.PatrolShift{shift("Morning", "08:00", "12:00")}
	if _, ok := Resolve(shifts, at(29, 13, 0, 0), jakarta); ok {
		t.Fatal("13:00 is outside every shift")
	}

	inactive := shift("Morning", "08:00", "12:00")
	inactive.IsActive = false
	if _, ok := Resolve([]models.PatrolShift{inactive}, at(29, 9, 0, 0), jakarta); ok {
		t.Fatal("inactive shifts must be ignored")
	}
}

func TestBoundsValidation(t *testing.T) {
	valid := map[[2]string]int{
		{"08:00", "16:00"}: 480,
		{"16:00", "24:00"}: 480,
		{"16:00", "00:00"}: 480,
		{"22:00", "06:00"}: 480,
		{"00:00", "24:00"}: 1440,
	}
	for times, want := range valid {
		_, duration, err := Bounds(shift("x", times[0], times[1]))
		if err != nil || duration != want {
			t.Errorf("%v: duration %d err %v, want %d", times, duration, err, want)
		}
	}

	for _, times := range [][2]string{{"08:00", "08:00"}, {"24:00", "08:00"}, {"8:00", "16:00"}, {"08:60", "16:00"}, {"25:00", "16:00"}, {"00:00", "00:00"}, {"ab:cd", "16:00"}} {
		if _, _, err := Bounds(shift("x", times[0], times[1])); err == nil {
			t.Errorf("%v should be invalid", times)
		}
	}
}

func TestOverlaps(t *testing.T) {
	cases := []struct {
		a, b models.PatrolShift
		want bool
	}{
		{shift("a", "08:00", "16:00"), shift("b", "16:00", "24:00"), false},
		{shift("a", "08:00", "16:00"), shift("b", "15:00", "23:00"), true},
		{shift("a", "22:00", "06:00"), shift("b", "05:00", "09:00"), true},
		{shift("a", "22:00", "06:00"), shift("b", "06:00", "22:00"), false},
		{shift("a", "16:00", "24:00"), shift("b", "00:00", "08:00"), false},
		{shift("a", "00:00", "24:00"), shift("b", "10:00", "11:00"), true},
	}
	for _, c := range cases {
		got, err := Overlaps(c.a, c.b)
		if err != nil || got != c.want {
			t.Errorf("%s-%s vs %s-%s: got %v err %v, want %v", c.a.StartTime, c.a.EndTime, c.b.StartTime, c.b.EndTime, got, err, c.want)
		}
		if reverse, _ := Overlaps(c.b, c.a); reverse != c.want {
			t.Errorf("overlap must be symmetric for %v / %v", c.a, c.b)
		}
	}
}
