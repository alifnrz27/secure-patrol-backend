package dto

import (
	"bytes"
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
			PatrolGroup:    models.PatrolGroup{ShiftName: "Shift 1", ShiftDate: shiftDate},
			PatrolListItem: models.PatrolListItem{Name: "Main Gate", Location: "Building A"},
			ScannedByUser:  models.User{Name: "Budi", Email: "budi@securepatrol.local"},
		},
		{
			ScannedAt:      scannedAt,
			ReceivedAt:     scannedAt.Add(3 * time.Hour), // sent later from the phone
			Condition:      models.PatrolConditionAbnormal,
			Note:           "Pintu tidak terkunci",
			PatrolGroup:    models.PatrolGroup{ShiftName: "Shift 1", ShiftDate: shiftDate},
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
		{"Waktu scan", "Diterima server", "Dikirim offline", "Tanggal shift", "Shift", "Titik", "Lokasi", "Kondisi", "Catatan", "Petugas", "Email petugas"},
		{"29/09/2026 08:15:30", "29/09/2026 08:15:32", "Tidak", "29/09/2026", "Shift 1", "Main Gate", "Building A", "Normal", "", "Budi", "budi@securepatrol.local"},
		{"29/09/2026 08:15:30", "29/09/2026 11:15:30", "Ya", "29/09/2026", "Shift 1", "Server Room", "Building A - 3rd Floor", "Tidak Normal", "Pintu tidak terkunci", "Andi (akun dihapus)", "andi@securepatrol.local"},
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
	for key, value := range map[string]string{"Diekspor oleh": "Super Admin", "Shift": "Shift 1", "Titik": "Semua", "Jumlah data": "2"} {
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
