# Instruksi untuk Claude: Update Aplikasi Mobile — Daftar Titik yang Ditugaskan

Kamu melanjutkan aplikasi mobile Secure Patrol yang sudah dibangun dari `MOBILE_CLAUDE_PROMPT.md` (dan update
sebelumnya). Backend sekarang mendukung **penugasan titik patroli ke petugas** per shift. Terapkan perubahan di
bawah tanpa merusak fitur yang ada; pelajari dulu kode Dashboard, cache, dan alur scan, lalu ikuti polanya.

Kontrak API: `/docs/openapi.yaml` di backend (environment development).

## 1. Cara kerja di backend

- Setiap unit punya tombol **ON/OFF** penugasan (diatur Kepala/Admin Keamanan di web; aplikasi tidak perlu
  membaca setting ini).
- Kepala/Admin Keamanan menugaskan **satu atau beberapa petugas** (role Admin Keamanan atau Tim Keamanan) ke
  titik pada **shift yang sedang berjalan**. Shift berikutnya mulai tanpa penugasan.
- Jika ON, server **sudah menyaring** daftar titik untuk Admin/Tim Keamanan di aplikasi mobile:
  **titik yang ditugaskan ke user yang login + titik yang tidak ditugaskan ke siapa pun**. Titik yang hanya
  ditugaskan ke petugas lain tidak dikirim.
- Jika OFF, semua titik dikirim seperti biasa (field `assignees` tetap ada).
- Kepala Keamanan selalu menerima semua titik.
- **Jangan menyaring sendiri di aplikasi.** Selalu tampilkan apa yang dikirim server.

## 2. Perubahan respons

`GET /api/v1/patrol-groups/current` (juga `/patrol-groups/{id}` dan `/patrol-list-items`): setiap item punya
field baru `assignees`:

```json
{
  "id": 12, "patrol_point_id": 2, "area_id": 3, "area_name": "Parkir",
  "name": "Parking Area", "location": "Basement 1", "nfc_code": "DUMMY-NFC-0002",
  "is_scanned": false, "scan_count": 0,
  "assignees": [
    { "id": 5, "name": "Tim Keamanan", "email": "tim@securepatrol.local" },
    { "id": 9, "name": "Tim Dua", "email": "tim2@securepatrol.local" }
  ]
}
```

- `assignees: []` = titik untuk semua petugas.
- `assignees` berisi `user.id` yang login = **tugas user ini** (bisa bersama petugas lain).
- `progress` pada group tetap dihitung untuk **seluruh titik unit**, bukan hanya titik user.

Parser harus menerima `assignees` kosong maupun berisi, dan tetap mengabaikan field yang tidak dikenal.

## 3. Tampilan Dashboard

Urutan dari server sudah per area lalu nama. Tampilkan:

1. **Bagian "Tugas Anda"** (hanya jika ada): titik yang `assignees`-nya berisi user yang login. Beri penanda
   jelas (badge "Tugas Anda"). Jika titik juga ditugaskan ke orang lain, tampilkan "Bersama: <nama lain>".
2. **Bagian "Titik umum"**: titik dengan `assignees` kosong (boleh dikerjakan siapa saja).
3. Di dalam setiap bagian, tetap kelompokkan per area (`area_name`, kosong = "Lainnya") seperti sebelumnya.

Jika tidak ada satu pun titik yang ditugaskan ke user, tampilkan daftar seperti biasa tanpa bagian "Tugas Anda".

**Progres:**
- Header utama: **progres tugas Anda**, dihitung di aplikasi dari item "Tugas Anda" (`is_scanned`), mis.
  "Tugas Anda 2/4".
- Progres titik umum: "Titik umum 3/6".
- Progres seluruh unit (`progress` dari server) boleh tetap ditampilkan kecil sebagai informasi.

**Filter:** tambahkan chip "Tugas Anda" di samping filter yang sudah ada (Semua / Belum / Sudah / Tidak Normal / area).

## 4. Memperbarui daftar

Penugasan bisa berubah kapan saja selama shift. Ambil ulang `GET /api/v1/patrol-groups/current`:

- saat aplikasi kembali ke foreground,
- saat pull-to-refresh,
- setiap 2 menit selama Dashboard terbuka dan online,
- setelah scan terkirim (sudah ada).

Jika sebuah titik **hilang** dari daftar setelah refresh (dialihkan ke petugas lain) dan ada scan pending lokal
untuk titik itu, **jangan hapus scan pending-nya**; tetap kirim seperti biasa. Tampilkan titik tersebut di bagian
kecil "Menunggu sinkron" sampai terkirim.

Saat offline, pakai daftar cache terakhir (sudah ada). Saat shift berganti, penugasan kosong lagi dan daftar
kembali berisi semua titik setelah refresh.

## 5. Scan titik yang bukan tugas Anda

Server **tetap menerima** scan titik mana pun di unit user (penugasan hanya mengatur daftar yang tampil). Pada
alur pencarian titik setelah NFC dipindai (`MOBILE_CLAUDE_PROMPT.md` bagian 7.2):

- Ditemukan di daftar group aktif → alur normal.
- Tidak ada di daftar group aktif tetapi ditemukan di cache `/patrol-points` atau lewat
  `GET /api/v1/patrol-points/by-nfc` → tampilkan konfirmasi **"Titik ini bukan tugas Anda pada shift ini. Tetap
  lanjutkan scan?"**. Jika dilanjutkan, alur scan normal (validasi lokasi/wajah tetap berlaku).
- Tidak ditemukan sama sekali → "Tag NFC ini tidak terdaftar di unit Anda." (seperti sebelumnya).

## 6. Hal yang tidak berubah

- Tidak ada endpoint baru untuk mobile; penugasan hanya diatur dari web.
- Tidak ada error baru untuk mobile.
- Login, setting di login (`settings`), antrian scan offline, dan validasi lokasi/wajah tetap sama.

## 7. Kriteria selesai

Uji di environment development:

1. Di web: login Kepala Keamanan, nyalakan **Penugasan titik per petugas** di Pengaturan Sistem unit, lalu pada
   group yang sedang berjalan tugaskan titik A ke `tim@securepatrol.local`, titik B ke petugas lain.

Lalu cek di mobile:

- [ ] Login `tim@`: muncul bagian **Tugas Anda** berisi titik A, bagian **Titik umum** berisi titik tanpa petugas;
      titik B tidak tampil.
- [ ] Header menampilkan progres "Tugas Anda x/y" yang berubah setelah scan titik A.
- [ ] Titik yang ditugaskan ke `tim@` dan petugas lain menampilkan "Bersama: …".
- [ ] Penugasan diubah di web → setelah pull-to-refresh / 2 menit / kembali ke foreground, daftar ikut berubah.
- [ ] Scan NFC titik B → muncul konfirmasi "bukan tugas Anda"; jika dilanjutkan, scan tersimpan (201).
- [ ] Penugasan dimatikan di web → setelah refresh semua titik tampil.
- [ ] Scan pending untuk titik yang kemudian dialihkan tetap terkirim.
- [ ] Login `kepala@` di mobile → semua titik tampil.
- [ ] Parsing tetap berhasil untuk item dengan `assignees` kosong dan berisi.

## 8. Laporan akhir

Laporkan file yang diubah, cara menguji, hasil setiap kriteria di bagian 7, keputusan yang kamu ambil sendiri,
dan permintaan ke backend jika ada.
