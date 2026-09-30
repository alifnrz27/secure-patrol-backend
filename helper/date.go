package helper

import (
	"fmt"
	"math"
	"strings"
	"time"
)

func LocalToUTC(localDatetime string, tzInSeconds int) (time.Time, error) {
	layoutFormat := "2006-01-02 15:04:05"
	localDatetimeParsed, err := time.Parse(layoutFormat, localDatetime)
	if err != nil {
		return localDatetimeParsed, err
	}

	utcDatetime := localDatetimeParsed.Add(-time.Second * time.Duration(tzInSeconds))

	return utcDatetime, nil
}

func FormatDatetime(datetime string) (time.Time, error) {
	layoutFormat := "2006-01-02 15:04:05"
	datetimeParsed, err := time.Parse(layoutFormat, datetime)
	if err != nil {
		return datetimeParsed, err
	}

	return datetimeParsed, nil
}

func FormatDatetimeByTimezone(datetime string, timezone string) (time.Time, error) {
	layoutFormat := "2006-01-02 15:04:05"
	tz, _ := time.LoadLocation(timezone)
	datetimeParsed, err := time.ParseInLocation(layoutFormat, datetime, tz)
	if err != nil {
		return datetimeParsed, err
	}

	return datetimeParsed, nil
}

func FormatMysqlDatetime(datetime string) (time.Time, error) {
	layoutFormat := time.RFC3339
	datetimeParsed, err := time.Parse(layoutFormat, datetime)
	if err != nil {
		return datetimeParsed, err
	}

	return datetimeParsed, nil
}

func CalculateQueueDelay(flightSchedule string, originTimezone int, reminderConfigInMinutes int) int64 {
	formattedFlightSchedule, _ := FormatDatetime(flightSchedule)

	currentLocalTime := time.Now().UTC().Add(time.Second * time.Duration(originTimezone))

	delayInMiliseconds := formattedFlightSchedule.Sub(currentLocalTime).Milliseconds()
	reminderConfigInMilliseconds := int64(reminderConfigInMinutes) * int64(time.Minute/time.Millisecond)

	return delayInMiliseconds - reminderConfigInMilliseconds
}

func ReverseQueueDelay(queueDelayInMilliseconds int64, originTimezone int) (time.Time, time.Time) {
	orgReminderSchedule := time.Now().UTC().Add(time.Second * time.Duration(originTimezone)).Add(time.Millisecond * time.Duration(queueDelayInMilliseconds))
	systemReminderSchedule := time.Now().Add(time.Millisecond * time.Duration(queueDelayInMilliseconds))

	return orgReminderSchedule, systemReminderSchedule
}

func DateToString(datetime time.Time) string {
	layoutFormat := "2006-01-02 15:04:05"
	return datetime.Format(layoutFormat)
}

func DateToStringDDMMYYYY(datetime time.Time) string {
	layoutFormat := "2006-01-02"
	return datetime.Format(layoutFormat)
}

func DateToStringYYYYMMDD(datetime time.Time) string {
	layoutFormat := "2006-01-02"
	return datetime.Format(layoutFormat)
}

func DateToStringHHMM(datetime time.Time) string {
	layoutFormat := "15:04"
	return datetime.Format(layoutFormat)
}

func DaysToMilliseconds(days int) int64 {
	milliseconds := int64(days * 24 * 60 * 60 * 1000)
	return milliseconds
}

func DateStringToString(date string) string {
	dateParsed, _ := time.Parse("2006-01-02T15:04:05-07:00", date)
	dateFormat := dateParsed.Format("2006-01-02")
	return dateFormat
}

func LocalTimeTzStringToUTC(localDatetime string, timezoneString string) (time.Time, error) {
	loc, _ := time.LoadLocation(timezoneString)
	_, tzOffset := time.Now().In(loc).Zone()

	layoutFormat := "2006-01-02 15:04:05"
	localDatetimeParsed, err := time.Parse(layoutFormat, localDatetime)
	if err != nil {
		return localDatetimeParsed, err
	}

	utcDatetime := localDatetimeParsed.Add(-time.Second * time.Duration(tzOffset))

	return utcDatetime, nil
}

func TimeDiff(tTime time.Time) time.Duration {
	tNow := time.Now()
	// loc, _ := time.LoadLocation("Asia/Jakarta")
	// tNow := time.Now().UTC().Add(time.Minute * -275)
	return tTime.Sub(tNow)
}

func TimeDurationToString1(loc *time.Location, tTime time.Time) string {
	// Extract hours, minutes, and seconds
	hours1 := int(time.Duration(TimeDiff(tTime.In(loc)).Hours()))
	minutes1 := int(time.Duration(TimeDiff(tTime.In(loc)).Minutes())) % 60
	// seconds1 := int(time.Duration(TimeDiff(tTime.In(loc).Add(time.Hour*24)).Seconds())) % 60

	// Build the readable string
	part := ""
	parts := []string{}
	if math.Abs(float64(hours1)) > 0 {
		if hours1 < 0 {
			part = fmt.Sprintf("%d jam", hours1*-1)
		} else {
			part = fmt.Sprintf("%d jam", hours1)
		}
		parts = append(parts, part)
	}
	if math.Abs(float64(minutes1)) > 0 {
		if minutes1 < 0 {
			part = fmt.Sprintf("%d menit", minutes1*-1)
		} else {
			part = fmt.Sprintf("%d menit", minutes1)
		}
		parts = append(parts, part)
	}

	return strings.Join(parts, " ")
}

func GetNightWindow(now time.Time, jakarta *time.Location) (time.Time, time.Time) {
	nowJakarta := now.In(jakarta)
	today := time.Date(nowJakarta.Year(), nowJakarta.Month(), nowJakarta.Day(), 0, 0, 0, 0, jakarta)

	windowStart := time.Date(today.Year(), today.Month(), today.Day(), 16, 0, 0, 0, jakarta)
	windowEnd := windowStart.Add(22 * time.Hour) // → 12:00 next day

	// If we're past midnight but before noon, shift back to last night's window
	if nowJakarta.Hour() < 14 {
		windowStart = windowStart.Add(-24 * time.Hour)
		windowEnd = windowEnd.Add(-24 * time.Hour)
	}

	return windowStart, windowEnd
}

// ParseDateQuery validates an optional YYYY-MM-DD query value.
func ParseDateQuery(value string) (string, error) {
	if value == "" {
		return "", nil
	}
	if _, err := time.Parse("2006-01-02", value); err != nil {
		return "", fmt.Errorf("date must use the YYYY-MM-DD format")
	}
	return value, nil
}
