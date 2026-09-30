package dto

import (
	"bytes"
	"fmt"
	"regexp"
	"secure-patrol-backend/models"
	"time"

	"github.com/xuri/excelize/v2"
)

// OfflineThreshold: a scan received more than this after it was scanned was
// stored on the phone and sent later.
const OfflineThreshold = 5 * time.Minute

var scanExportHeaders = []string{
	"Waktu scan", "Diterima server", "Dikirim offline", "Tanggal shift", "Unit", "Shift",
	"Titik", "Lokasi", "Kondisi", "Catatan", "Petugas", "Email petugas",
}

var scanExportColumnWidths = []float64{20, 20, 15, 14, 24, 14, 26, 30, 14, 45, 26, 32}

// ScanExportMeta describes the export for the "Filter" sheet.
type ScanExportMeta struct {
	ExportedAt time.Time
	ExportedBy string
	Unit       string
	Shift      string
	Point      string
	Officer    string
	DateFrom   string
	DateTo     string
	Location   *time.Location
}

// Deleted accounts get their email renamed to free it (deleted_<id>_<time>_<email>).
var releasedEmail = regexp.MustCompile(`^deleted_\d+_\d+_`)

// excelTime keeps the wall clock of loc. Excel has no time zones and excelize
// writes instants as UTC, so the local time is passed as if it were UTC.
func excelTime(t time.Time, loc *time.Location) time.Time {
	l := t.In(loc)
	return time.Date(l.Year(), l.Month(), l.Day(), l.Hour(), l.Minute(), l.Second(), 0, time.UTC)
}

func conditionLabel(condition string) string {
	if condition == models.PatrolConditionAbnormal {
		return "Tidak Normal"
	}
	return "Normal"
}

// BuildScanExcel writes the scan history as an .xlsx file: one sheet with the
// data (real date cells, frozen header, filterable table) and one describing
// the filter used.
func BuildScanExcel(scans []models.PatrolScan, meta ScanExportMeta) ([]byte, error) {
	f := excelize.NewFile()
	defer f.Close()

	const dataSheet = "Riwayat Scan"
	if err := f.SetSheetName("Sheet1", dataSheet); err != nil {
		return nil, err
	}

	dateTimeFormat := "dd/mm/yyyy hh:mm:ss"
	dateFormat := "dd/mm/yyyy"
	dateTimeStyle, err := f.NewStyle(&excelize.Style{CustomNumFmt: &dateTimeFormat})
	if err != nil {
		return nil, err
	}
	dateStyle, err := f.NewStyle(&excelize.Style{CustomNumFmt: &dateFormat})
	if err != nil {
		return nil, err
	}
	abnormalStyle, err := f.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true, Color: "C00000"}})
	if err != nil {
		return nil, err
	}
	wrapStyle, err := f.NewStyle(&excelize.Style{Alignment: &excelize.Alignment{WrapText: true, Vertical: "top"}})
	if err != nil {
		return nil, err
	}

	sw, err := f.NewStreamWriter(dataSheet)
	if err != nil {
		return nil, err
	}
	for i, width := range scanExportColumnWidths {
		if err := sw.SetColWidth(i+1, i+1, width); err != nil {
			return nil, err
		}
	}
	if err := sw.SetPanes(&excelize.Panes{Freeze: true, YSplit: 1, TopLeftCell: "A2", ActivePane: "bottomLeft"}); err != nil {
		return nil, err
	}

	header := make([]interface{}, len(scanExportHeaders))
	for i, title := range scanExportHeaders {
		header[i] = title
	}
	if err := sw.SetRow("A1", header); err != nil {
		return nil, err
	}

	for i, scan := range scans {
		offline := "Tidak"
		if scan.ReceivedAt.Sub(scan.ScannedAt) > OfflineThreshold {
			offline = "Ya"
		}

		officer := scan.ScannedByUser.Name
		if scan.ScannedByUser.DeletedAt.Valid {
			officer += " (akun dihapus)"
		}

		condition := excelize.Cell{Value: conditionLabel(scan.Condition)}
		if scan.Condition == models.PatrolConditionAbnormal {
			condition.StyleID = abnormalStyle
		}

		shiftDate := scan.PatrolGroup.ShiftDate
		row := []interface{}{
			excelize.Cell{StyleID: dateTimeStyle, Value: excelTime(scan.ScannedAt, meta.Location)},
			excelize.Cell{StyleID: dateTimeStyle, Value: excelTime(scan.ReceivedAt, meta.Location)},
			offline,
			excelize.Cell{StyleID: dateStyle, Value: time.Date(shiftDate.Year(), shiftDate.Month(), shiftDate.Day(), 0, 0, 0, 0, time.UTC)},
			scan.PatrolGroup.UnitName,
			scan.PatrolGroup.ShiftName,
			scan.PatrolListItem.Name,
			scan.PatrolListItem.Location,
			condition,
			excelize.Cell{StyleID: wrapStyle, Value: scan.Note},
			officer,
			releasedEmail.ReplaceAllString(scan.ScannedByUser.Email, ""),
		}

		if err := sw.SetRow(fmt.Sprintf("A%d", i+2), row); err != nil {
			return nil, err
		}
	}

	// A table gives the header style and a filter button on every column.
	if len(scans) > 0 {
		lastCell, _ := excelize.CoordinatesToCellName(len(scanExportHeaders), len(scans)+1)
		if err := sw.AddTable(&excelize.Table{
			Range:          "A1:" + lastCell,
			Name:           "RiwayatScan",
			StyleName:      "TableStyleMedium2",
			ShowRowStripes: boolPtr(true),
		}); err != nil {
			return nil, err
		}
	}

	if err := sw.Flush(); err != nil {
		return nil, err
	}

	if err := writeFilterSheet(f, len(scans), meta); err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func writeFilterSheet(f *excelize.File, rows int, meta ScanExportMeta) error {
	const sheet = "Filter"
	if _, err := f.NewSheet(sheet); err != nil {
		return err
	}

	orAll := func(value string) string {
		if value == "" {
			return "Semua"
		}
		return value
	}

	lines := [][]interface{}{
		{"Diekspor pada", meta.ExportedAt.In(meta.Location).Format("02/01/2006 15:04:05") + " (" + meta.Location.String() + ")"},
		{"Diekspor oleh", meta.ExportedBy},
		{"Unit", orAll(meta.Unit)},
		{"Shift", orAll(meta.Shift)},
		{"Titik", orAll(meta.Point)},
		{"Petugas", orAll(meta.Officer)},
		{"Tanggal shift dari", orAll(meta.DateFrom)},
		{"Tanggal shift sampai", orAll(meta.DateTo)},
		{"Jumlah data", rows},
	}
	for i, line := range lines {
		if err := f.SetSheetRow(sheet, fmt.Sprintf("A%d", i+1), &line); err != nil {
			return err
		}
	}

	bold, err := f.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true}})
	if err != nil {
		return err
	}
	if err := f.SetCellStyle(sheet, "A1", fmt.Sprintf("A%d", len(lines)), bold); err != nil {
		return err
	}
	if err := f.SetColWidth(sheet, "A", "A", 22); err != nil {
		return err
	}
	return f.SetColWidth(sheet, "B", "B", 40)
}

func boolPtr(v bool) *bool { return &v }
