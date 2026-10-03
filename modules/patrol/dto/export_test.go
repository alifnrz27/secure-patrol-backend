package dto

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"image/jpeg"
	"secure-patrol-backend/pkg/imageutil"
	"testing"
	"time"

	"secure-patrol-backend/models"

	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"
)

func TestBuildScanExcel(t *testing.T) {
	jakarta := time.FixedZone("Asia/Jakarta", 7*60*60)
	scannedAt := time.Date(2026, 9, 29, 1, 15, 30, 0, time.UTC) // 08:15:30 WIB
	shiftDate := time.Date(2026, 9, 29, 0, 0, 0, 0, time.UTC)

	scans := []models.PatrolScan{
		{
			ScannedAt:      scannedAt,
			ReceivedAt:     scannedAt.Add(2 * time.Second),
			Condition:      models.PatrolConditionNormal,
			PatrolGroup:    models.PatrolGroup{UnitName: "Unit Utama", ShiftName: "Shift 1", ShiftDate: shiftDate},
			PatrolListItem: models.PatrolListItem{Name: "Main Gate", Location: "Building A"},
			ScannedByUser:  models.User{Name: "Budi", Email: "budi@securepatrol.local"},
		},
		{
			ScannedAt:      scannedAt,
			ReceivedAt:     scannedAt.Add(3 * time.Hour), // sent later from the phone
			Condition:      models.PatrolConditionAbnormal,
			Note:           "Pintu tidak terkunci",
			PatrolGroup:    models.PatrolGroup{UnitName: "Unit Utama", ShiftName: "Shift 1", ShiftDate: shiftDate},
			PatrolListItem: models.PatrolListItem{Name: "Server Room", Location: "Building A - 3rd Floor"},
			ScannedByUser: models.User{
				Name:      "Andi",
				Email:     "deleted_7_1790668800_andi@securepatrol.local",
				DeletedAt: gorm.DeletedAt{Time: scannedAt, Valid: true},
			},
		},
	}

	content, err := BuildScanExcel(scans, ScanExportMeta{
		ExportedAt: scannedAt, ExportedBy: "Super Admin", Shift: "Shift 1", Location: jakarta,
	})
	if err != nil {
		t.Fatal(err)
	}

	f, err := excelize.OpenReader(bytes.NewReader(content))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	rows, err := f.GetRows("Riwayat Scan")
	if err != nil {
		t.Fatal(err)
	}

	want := [][]string{
		{"Waktu scan", "Diterima server", "Dikirim offline", "Tanggal shift", "Unit", "Shift", "Titik", "Lokasi", "Kondisi", "Catatan", "Petugas", "Email petugas"},
		{"29/09/2026 08:15:30", "29/09/2026 08:15:32", "Tidak", "29/09/2026", "Unit Utama", "Shift 1", "Main Gate", "Building A", "Normal", "", "Budi", "budi@securepatrol.local"},
		{"29/09/2026 08:15:30", "29/09/2026 11:15:30", "Ya", "29/09/2026", "Unit Utama", "Shift 1", "Server Room", "Building A - 3rd Floor", "Tidak Normal", "Pintu tidak terkunci", "Andi (akun dihapus)", "andi@securepatrol.local"},
	}
	if len(rows) != len(want) {
		t.Fatalf("got %d rows, want %d: %v", len(rows), len(want), rows)
	}
	for i := range want {
		for j := range want[i] {
			got := ""
			if j < len(rows[i]) {
				got = rows[i][j]
			}
			if got != want[i][j] {
				t.Errorf("row %d col %d: got %q, want %q", i+1, j+1, got, want[i][j])
			}
		}
	}

	// Time columns are real date cells, not text.
	if cellType, _ := f.GetCellType("Riwayat Scan", "A2"); cellType == excelize.CellTypeSharedString || cellType == excelize.CellTypeInlineString {
		t.Error("A2 must be a date cell, got a text cell")
	}

	filterRows, _ := f.GetRows("Filter")
	summary := map[string]string{}
	for _, r := range filterRows {
		if len(r) == 2 {
			summary[r[0]] = r[1]
		}
	}
	for key, value := range map[string]string{"Diekspor oleh": "Super Admin", "Unit": "Semua", "Shift": "Shift 1", "Titik": "Semua", "Jumlah data": "2"} {
		if summary[key] != value {
			t.Errorf("filter sheet %q: got %q, want %q", key, summary[key], value)
		}
	}
}

func TestBuildScanExcelEmpty(t *testing.T) {
	content, err := BuildScanExcel(nil, ScanExportMeta{ExportedAt: time.Now(), Location: time.UTC})
	if err != nil {
		t.Fatal(err)
	}
	f, err := excelize.OpenReader(bytes.NewReader(content))
	if err != nil {
		t.Fatal(err)
	}
	rows, _ := f.GetRows("Riwayat Scan")
	if len(rows) != 1 || rows[0][0] != "Waktu scan" {
		t.Fatalf("empty export must contain only the header, got %v", rows)
	}
}

func testJPEG(t *testing.T, width, height int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.RGBA{uint8(x), uint8(y), 120, 255})
		}
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, nil); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestBuildScanExcelWithPhotos(t *testing.T) {
	jakarta := time.FixedZone("Asia/Jakarta", 7*60*60)
	at := time.Date(2026, 9, 29, 1, 15, 30, 0, time.UTC)
	stored := map[string][]byte{
		"patrol-scans/a.jpg": testJPEG(t, 1600, 1200),
		"patrol-scans/b.jpg": testJPEG(t, 600, 1200),
	}
	scan := func(id int64, paths ...string) models.PatrolScan {
		var photos []models.PatrolScanPhoto
		for i, path := range paths {
			photos = append(photos, models.PatrolScanPhoto{Path: path, SortOrder: i + 1})
		}
		return models.PatrolScan{
			ID: id, ScannedAt: at, ReceivedAt: at, Condition: models.PatrolConditionNormal,
			PatrolGroup: models.PatrolGroup{ShiftDate: at}, Photos: photos,
		}
	}
	scans := []models.PatrolScan{
		scan(1, "patrol-scans/a.jpg", "patrol-scans/b.jpg", "patrol-scans/missing.jpg"),
		scan(2),
	}

	content, err := BuildScanExcel(scans, ScanExportMeta{
		ExportedAt: at, Location: jakarta, IncludePhotos: true,
		Thumbnail: func(path string) ([]byte, error) {
			data, ok := stored[path]
			if !ok {
				return nil, errors.New("not found")
			}
			thumb, _, _, err := imageutil.Thumbnail(data, thumbMaxWidth, thumbMaxHeight, thumbQuality)
			return thumb, err
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	f, err := excelize.OpenReader(bytes.NewReader(content))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	rows, _ := f.GetRows("Riwayat Scan")
	if got := rows[0][12:]; len(got) != 3 || got[0] != "Foto 1" || got[2] != "Foto 3" {
		t.Fatalf("photo headers = %v", got)
	}

	for cell, wantSize := range map[string][2]int{"M2": {320, 240}, "N2": {120, 240}} {
		pics, err := f.GetPictures("Riwayat Scan", cell)
		if err != nil || len(pics) != 1 {
			t.Fatalf("%s: %d pictures, err %v", cell, len(pics), err)
		}
		cfg, _, err := image.DecodeConfig(bytes.NewReader(pics[0].File))
		if err != nil || cfg.Width != wantSize[0] || cfg.Height != wantSize[1] {
			t.Errorf("%s: thumbnail %dx%d (err %v), want %v", cell, cfg.Width, cfg.Height, err, wantSize)
		}
		if len(pics[0].File) > 60*1024 {
			t.Errorf("%s: thumbnail too large: %d bytes", cell, len(pics[0].File))
		}
	}
	if value, _ := f.GetCellValue("Riwayat Scan", "O2"); value != "foto tidak ditemukan" {
		t.Errorf("missing photo cell = %q", value)
	}
	if pics, _ := f.GetPictures("Riwayat Scan", "M3"); len(pics) != 0 {
		t.Error("scan without photos must have no picture")
	}
	if height, _ := f.GetRowHeight("Riwayat Scan", 2); height != photoRowHeight {
		t.Errorf("row with photos has height %v", height)
	}

	filter, _ := f.GetRows("Filter")
	found := false
	for _, line := range filter {
		if len(line) == 2 && line[0] == "Foto" && line[1] == "Ya (thumbnail)" {
			found = true
		}
	}
	if !found {
		t.Errorf("filter sheet has no Foto line: %v", filter)
	}
}
