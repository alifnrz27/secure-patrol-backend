package dto

import (
	"bytes"
	"fmt"
	"regexp"
	"runtime"
	"secure-patrol-backend/models"
	"secure-patrol-backend/pkg/log"
	"sync"
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

// Photo columns of an export with photos. Thumbnails are made at twice the
// displayed size so they stay sharp when zoomed in Excel.
var photoHeaders = []string{"Foto 1", "Foto 2", "Foto 3"}

const (
	photoColumnWidth = 24 // characters, about 170 px
	thumbMaxWidth    = 320
	thumbMaxHeight   = 240
	thumbScale       = 0.5  // shown at 160 x 120 px at most
	photoRowHeight   = 94.0 // points: 120 px + margin
	thumbQuality     = 75
	photoMissingText = "foto tidak ditemukan"
)

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
	// IncludePhotos adds the scan photos as thumbnails (Foto 1-3 columns).
	IncludePhotos bool
	// Thumbnail returns the JPEG thumbnail of a stored photo; nil uses
	// ScanPhotoThumbnail (cached next to the photo).
	Thumbnail func(path string) ([]byte, error)
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
// the filter used. With meta.IncludePhotos the photos are embedded as
// thumbnails; pictures need the regular (non streaming) writer, so exports
// with photos are limited to fewer rows by the caller.
func BuildScanExcel(scans []models.PatrolScan, meta ScanExportMeta) ([]byte, error) {
	f := excelize.NewFile()
	defer f.Close()

	const dataSheet = "Riwayat Scan"
	if err := f.SetSheetName("Sheet1", dataSheet); err != nil {
		return nil, err
	}

	styles, err := newExportStyles(f)
	if err != nil {
		return nil, err
	}

	headers := append([]string{}, scanExportHeaders...)
	widths := append([]float64{}, scanExportColumnWidths...)
	if meta.IncludePhotos {
		headers = append(headers, photoHeaders...)
		for range photoHeaders {
			widths = append(widths, photoColumnWidth)
		}
	}

	header := make([]excelize.Cell, len(headers))
	for i, title := range headers {
		header[i] = excelize.Cell{Value: title}
	}

	if meta.IncludePhotos {
		err = writeRowsWithPhotos(f, dataSheet, header, widths, scans, meta, styles)
	} else {
		err = writeRowsStreaming(f, dataSheet, header, widths, scans, meta, styles)
	}
	if err != nil {
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

type exportStyles struct {
	dateTime, date, abnormal, wrap, top int
}

func newExportStyles(f *excelize.File) (exportStyles, error) {
	var styles exportStyles
	var err error
	dateTimeFormat := "dd/mm/yyyy hh:mm:ss"
	dateFormat := "dd/mm/yyyy"
	if styles.dateTime, err = f.NewStyle(&excelize.Style{CustomNumFmt: &dateTimeFormat, Alignment: &excelize.Alignment{Vertical: "top"}}); err != nil {
		return styles, err
	}
	if styles.date, err = f.NewStyle(&excelize.Style{CustomNumFmt: &dateFormat, Alignment: &excelize.Alignment{Vertical: "top"}}); err != nil {
		return styles, err
	}
	if styles.abnormal, err = f.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true, Color: "C00000"}, Alignment: &excelize.Alignment{Vertical: "top"}}); err != nil {
		return styles, err
	}
	if styles.wrap, err = f.NewStyle(&excelize.Style{Alignment: &excelize.Alignment{WrapText: true, Vertical: "top"}}); err != nil {
		return styles, err
	}
	styles.top, err = f.NewStyle(&excelize.Style{Alignment: &excelize.Alignment{Vertical: "top"}})
	return styles, err
}

// scanRow is one data row without the photo columns.
func scanRow(scan models.PatrolScan, meta ScanExportMeta, styles exportStyles) []excelize.Cell {
	offline := "Tidak"
	if scan.ReceivedAt.Sub(scan.ScannedAt) > OfflineThreshold {
		offline = "Ya"
	}

	officer := scan.ScannedByUser.Name
	if scan.ScannedByUser.DeletedAt.Valid {
		officer += " (akun dihapus)"
	}

	condition := excelize.Cell{StyleID: styles.top, Value: conditionLabel(scan.Condition)}
	if scan.Condition == models.PatrolConditionAbnormal {
		condition.StyleID = styles.abnormal
	}

	shiftDate := scan.PatrolGroup.ShiftDate
	text := func(value string) excelize.Cell { return excelize.Cell{StyleID: styles.top, Value: value} }
	return []excelize.Cell{
		{StyleID: styles.dateTime, Value: excelTime(scan.ScannedAt, meta.Location)},
		{StyleID: styles.dateTime, Value: excelTime(scan.ReceivedAt, meta.Location)},
		text(offline),
		{StyleID: styles.date, Value: time.Date(shiftDate.Year(), shiftDate.Month(), shiftDate.Day(), 0, 0, 0, 0, time.UTC)},
		text(scan.PatrolGroup.UnitName),
		text(scan.PatrolGroup.ShiftName),
		text(scan.PatrolListItem.Name),
		text(scan.PatrolListItem.Location),
		condition,
		{StyleID: styles.wrap, Value: scan.Note},
		text(officer),
		text(releasedEmail.ReplaceAllString(scan.ScannedByUser.Email, "")),
	}
}

// table gives the header style and a filter button on every column.
func table(columns, rows int) *excelize.Table {
	lastCell, _ := excelize.CoordinatesToCellName(columns, rows+1)
	return &excelize.Table{Range: "A1:" + lastCell, Name: "RiwayatScan", StyleName: "TableStyleMedium2", ShowRowStripes: boolPtr(true)}
}

func toInterfaces(cells []excelize.Cell) []interface{} {
	values := make([]interface{}, len(cells))
	for i, cell := range cells {
		values[i] = cell
	}
	return values
}

func writeRowsStreaming(f *excelize.File, sheet string, header []excelize.Cell, widths []float64,
	scans []models.PatrolScan, meta ScanExportMeta, styles exportStyles) error {
	sw, err := f.NewStreamWriter(sheet)
	if err != nil {
		return err
	}
	for i, width := range widths {
		if err := sw.SetColWidth(i+1, i+1, width); err != nil {
			return err
		}
	}
	if err := sw.SetPanes(&excelize.Panes{Freeze: true, YSplit: 1, TopLeftCell: "A2", ActivePane: "bottomLeft"}); err != nil {
		return err
	}
	if err := sw.SetRow("A1", toInterfaces(header)); err != nil {
		return err
	}
	for i, scan := range scans {
		if err := sw.SetRow(fmt.Sprintf("A%d", i+2), toInterfaces(scanRow(scan, meta, styles))); err != nil {
			return err
		}
	}
	if len(scans) > 0 {
		if err := sw.AddTable(table(len(header), len(scans))); err != nil {
			return err
		}
	}
	return sw.Flush()
}

func writeRowsWithPhotos(f *excelize.File, sheet string, header []excelize.Cell, widths []float64,
	scans []models.PatrolScan, meta ScanExportMeta, styles exportStyles) error {
	for i, width := range widths {
		column, _ := excelize.ColumnNumberToName(i + 1)
		if err := f.SetColWidth(sheet, column, column, width); err != nil {
			return err
		}
	}
	if err := f.SetPanes(sheet, &excelize.Panes{Freeze: true, YSplit: 1, TopLeftCell: "A2", ActivePane: "bottomLeft"}); err != nil {
		return err
	}

	setRow := func(row int, cells []excelize.Cell) error {
		for col, cell := range cells {
			name, _ := excelize.CoordinatesToCellName(col+1, row)
			if err := f.SetCellValue(sheet, name, cell.Value); err != nil {
				return err
			}
			if cell.StyleID != 0 {
				if err := f.SetCellStyle(sheet, name, name, cell.StyleID); err != nil {
					return err
				}
			}
		}
		return nil
	}
	if err := setRow(1, header); err != nil {
		return err
	}

	thumbs := loadThumbnails(scans, meta.Thumbnail)
	firstPhotoColumn := len(scanExportHeaders) + 1

	for i, scan := range scans {
		row := i + 2
		if err := setRow(row, scanRow(scan, meta, styles)); err != nil {
			return err
		}
		if len(scan.Photos) == 0 {
			continue
		}
		if err := f.SetRowHeight(sheet, row, photoRowHeight); err != nil {
			return err
		}
		for p, photo := range scan.Photos {
			if p >= len(photoHeaders) {
				break
			}
			cell, _ := excelize.CoordinatesToCellName(firstPhotoColumn+p, row)
			thumb, ok := thumbs[photo.Path]
			if !ok {
				if err := f.SetCellValue(sheet, cell, photoMissingText); err != nil {
					return err
				}
				continue
			}
			err := f.AddPictureFromBytes(sheet, cell, &excelize.Picture{
				Extension: ".jpg",
				File:      thumb,
				Format: &excelize.GraphicOptions{
					ScaleX: thumbScale, ScaleY: thumbScale, OffsetX: 3, OffsetY: 3,
					Positioning: "oneCell", AltText: fmt.Sprintf("Foto %d scan %d", p+1, scan.ID),
				},
			})
			if err != nil {
				return err
			}
		}
	}

	if len(scans) > 0 {
		return f.AddTable(sheet, table(len(header), len(scans)))
	}
	return nil
}

// loadThumbnails loads the thumbnails of all photos in parallel. Photos that
// cannot be read (e.g. a missing file) are left out and shown as
// "foto tidak ditemukan".
func loadThumbnails(scans []models.PatrolScan, load func(path string) ([]byte, error)) map[string][]byte {
	if load == nil {
		load = ScanPhotoThumbnail
	}

	paths := make(chan string)
	results := make(map[string][]byte)
	var mu sync.Mutex
	var wg sync.WaitGroup

	for w := 0; w < runtime.NumCPU(); w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for path := range paths {
				thumb, err := load(path)
				if err != nil {
					log.Warnf("export: no thumbnail for photo %s: %v", path, err)
					continue
				}
				mu.Lock()
				results[path] = thumb
				mu.Unlock()
			}
		}()
	}

	seen := map[string]bool{}
	for _, scan := range scans {
		for p, photo := range scan.Photos {
			if p < len(photoHeaders) && !seen[photo.Path] {
				seen[photo.Path] = true
				paths <- photo.Path
			}
		}
	}
	close(paths)
	wg.Wait()
	return results
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
		{"Foto", map[bool]string{true: "Ya (thumbnail)", false: "Tidak"}[meta.IncludePhotos]},
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
