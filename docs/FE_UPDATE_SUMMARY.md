# Ringkasan Update Backend untuk Frontend (Web & Mobile)

Ringkasan semua perubahan backend setelah prompt awal (`WEB_CLAUDE_PROMPT.md`, `MOBILE_CLAUDE_PROMPT.md`).
Detail lengkap ada di file yang disebut di setiap bagian; kontrak API ada di `/docs/openapi.yaml` (development).

| # | Fitur | Web | Mobile | Detail |
|---|---|---|---|---|
| 1 | Multi unit | Banyak | Sedikit | `UPDATE_UNIT_WEB_PROMPT.md`, `UPDATE_UNIT_MOBILE_PROMPT.md` |
| 2 | Akun pusat tidak bisa login mobile | — | Pesan error | `UPDATE_UNIT_MOBILE_PROMPT.md` bagian 2 |
| 3 | Nama & logo aplikasi (branding) | Halaman + tampilan | Tampilan | Web 10.9, Mobile 4.0 / 7a |
| 4 | License | Halaman + banner + layar terkunci | Pesan error + banner | Web 10.10 / 12b, Mobile 7b |
| 5 | Export Excel: rentang wajib + foto | Dialog export | — | Web bagian 9 |
| 6 | Total patroli per titik per shift | Rekap & laporan | — | Web bagian 9 |
| 7 | Area titik patroli (gedung/lantai/parkir) | Menu Area + field di titik + filter | Kelompokkan daftar per area | Web 10.0a, Mobile bagian 6 |

---

## 1. Multi unit

**Konsep:** Super-Admin & Manager Keamanan = **pusat** (`user.unit = null`, lihat semua unit). Kepala, Admin, Tim
Keamanan = **unit** (`user.unit` terisi, hanya data unitnya). Manager hanya memantau (baca saja). Titik patroli,
shift, dan user unit dikelola Kepala/Admin unit; pusat tidak bisa membuat titik/shift.

- Login/refresh/`/auth/me`: field baru `user.unit_id`, `user.unit` (`id`, `code`, `name`, `latitude`, `longitude`, `is_active`).
- `settings` di login sudah sesuai unit user.
- Endpoint baru `/api/v1/units` (CRUD hanya Super-Admin).
- Parameter `unit_id` (hanya berlaku untuk user pusat) di: users, patrol-points, patrol-shifts, patrol-groups,
  patrol-groups/current (wajib untuk pusat), patrol-list-items, patrol-scans, export, settings, audit-logs.
- Field `unit_id` di titik patroli, shift, audit log; `unit` di group; `unit_id`/`unit_name` di `group` scan.
- Form user: `unit_id` (wajib untuk role unit bila dibuat Super-Admin; Kepala/Admin selalu membuat di unitnya).
- Setting per unit: `level`, `global_value`, `is_inherited`; `null` = ikuti nilai pusat.
- Help desk hanya ditulis Super-Admin. Kepala/Admin kini bisa membaca role (untuk form user).
- Error 403 baru: `your unit is inactive, contact the head office` (unit dinonaktifkan; muncul juga saat bekerja).

**Web:** pemilih unit di header (pusat), menu Unit, tabel hak akses baru, dashboard ringkasan per unit, kolom Unit.
**Mobile:** tampilkan nama unit; 403 unit nonaktif → logout dengan pesan, **antrian scan tidak dihapus**;
403 scan `this patrol point belongs to another unit` = gagal permanen.

## 2. Akun pusat tidak bisa login di mobile

Super-Admin dan Manager Keamanan ditolak di App Client android/ios: 403
`your role is not allowed to sign in on this platform`.
**Mobile:** tampilkan "Akun pusat hanya dapat digunakan melalui web admin."

## 3. Nama & logo aplikasi

- `GET /api/v1/branding` — **tanpa login** (cukup signature): `app_name`, `logo_base64`, `logo_mime_type`,
  `logo_updated_at`. `logo_base64 = null` → pakai logo bawaan.
- `PUT /api/v1/branding` (Super-Admin, web, multipart): `app_name` (wajib, maks 100), `logo` (JPEG/PNG maks 1 MB,
  opsional), `remove_logo=true`.

**Web:** nama & logo di halaman login, sidebar, `document.title`, favicon; menu **Tampilan Aplikasi** (Super-Admin).
**Mobile:** nama & logo di splash, login, header Dashboard; cache untuk offline, unduh ulang hanya jika
`logo_updated_at` berubah.

## 4. License

Backend wajib punya license aktif (batas unit aktif, batas App Client aktif, tanggal berakhir, masa tenggang).

- **Cek tanpa login:** `GET /api/v1/license/status` (cukup signature) → `status`, `locked`, `message`,
  `expires_at`, `grace_until`, `days_left`. Dipanggil saat aplikasi dibuka; `locked = true` → layar terkunci.
- Field baru `license` di login/refresh/`/auth/me`: `status` (`missing`/`active`/`grace`/`expired`/`invalid`),
  `expires_at`, `grace_until`, `days_left`.
- **Terkunci** (`missing`/`expired`/`invalid`): hanya Super-Admin bisa login dan hanya memakai `/auth/*`,
  `/app-config`, `/license`. Lainnya 403 `license is not active, contact your administrator`.
- `GET /api/v1/license`, `PUT /api/v1/license` `{"code": "SPL1...."}` (Super-Admin, web).
- 403 baru: `the license unit limit has been reached (N)`, `the license app client limit has been reached (N)`,
  `your unit exceeds the license limit, contact the head office`, `this app client exceeds the license limit`.

**Web:** cek `/license/status` saat dibuka → jika terkunci tampilkan layar "License belum aktif" + tombol login
Super-Admin → setelah login hanya halaman **License** (status, Install ID, pemakaian unit/App Client, form
pasang/perbarui license, pesan error 422). Banner kuning ≤ 30 hari, merah `grace`.
**Mobile:** cek `/license/status` saat dibuka → jika terkunci tampilkan pesan pengganti layar login;
403 license tidak aktif / unit melebihi license → logout dengan pesan, antrian scan tetap; 403 App Client
melebihi license → pesan, jangan logout; banner jika `status = grace`.

## 5. Export Excel: rentang wajib dan foto

`GET /api/v1/patrol-scans/export`:

- `date_from` dan `date_to` **wajib**; panjang maksimal dari setting `export_max_range_days` (default **7 hari**).
- `include_photos=true` → kolom **Foto 1–3** berisi thumbnail foto scan; rentang maksimal
  `export_photo_max_range_days` (default **1 hari**), maks 2.000 baris, nama file `riwayat-scan-foto_...`.
- Kolom baru **Unit** (setelah Tanggal shift); filter `unit_id` untuk pusat.
- Error 422 baru: `date_from and date_to are required for an export`, `an export can cover at most N days, ...`,
  `an export with photos can cover only 1 day, ...`, `an export with photos is limited to 2000 rows, ...`.

**Web:** dialog export dengan tanggal wajib, batasi pemilih tanggal sesuai setting (dari `GET /api/v1/settings`),
checkbox **"Sertakan foto"** (otomatis 1 hari), loading yang cukup lama (bisa ±30 detik).

## 6. Total patroli per titik dalam satu shift

`GET /api/v1/patrol-point-summary?group_id=` (satu shift pada satu tanggal) atau
`?shift_id=&date_from=&date_to=` (unit mengikuti shift). Akses: Super-Admin, Manager, Kepala/Admin (unit sendiri).

Response: `unit`, `shift`, `group_id`, `groups`, `totals` (`points`, `scanned_points`, `unscanned_points`,
`total_scans`, `abnormal_scans`), `items[]` per titik (`name`, `location`, `nfc_code`, `total_scans`,
`normal_scans`, `abnormal_scans`, `officers`, `groups`, `scanned_groups`, `first_scanned_at`, `last_scanned_at`),
termasuk titik yang belum di-scan.

**Web:** tab **Rekap per titik** di detail group (Monitoring) dan rekap per titik di halaman **Laporan**
(tabel + grafik + CSV; sorot titik yang tidak di-scan).

---

## 7. Area titik patroli

Titik patroli dalam satu unit bisa dikelompokkan ke **area** (gedung, lantai, parkir). Di API bernama `area`
(karena "group" sudah berarti group patroli shift per tanggal).

- Master area: `GET/POST /api/v1/patrol-areas`, `GET/PUT/DELETE /api/v1/patrol-areas/{id}` — dikelola Kepala/Admin
  Keamanan unit (web); semua user bisa membaca area unitnya. Field: `name` (unik per unit), `description`,
  `patrol_points_count`. Hapus hanya jika area kosong (409).
- Titik patroli: field `area_id` (opsional) dan `area` (`{id, name}` atau null); filter `?area_id=`.
  422 `area not found in this unit`.
- Daftar patroli shift, scan, dan rekap per titik membawa `area_id` + `area_name` (nama area **saat shift
  berjalan**, jadi riwayat tidak berubah jika area diganti nama). Urutan sudah per area lalu nama.
- Filter `area_id` di: `/patrol-points`, `/patrol-list-items`, `/patrol-scans`, `/patrol-scans/export`,
  `/patrol-point-summary`.
- Export Excel: kolom baru **Area** setelah Shift; sheet Filter berisi baris Area.

**Web:** menu **Area** (CRUD), select area di form titik, kolom dan filter area di daftar titik, monitoring
(kelompok per area + subtotal), riwayat scan, export, dan rekap per titik.
**Mobile:** daftar patroli di Dashboard dikelompokkan per area dengan progres per area dan chip filter area.

---

## Setting baru (Pengaturan Sistem)

| Key | Default | Keterangan |
|---|---|---|
| `export_max_range_days` | 7 | Rentang maksimal export Excel |
| `export_photo_max_range_days` | 1 | Rentang maksimal export Excel dengan foto |

Total setting sekarang 14 (grup baru: `export`).

## Data uji (development)

`superadmin@`, `manager@` (pusat, hanya web); `kepala@`, `admin@`, `tim@securepatrol.local` (Unit Utama,
`UNIT-UTAMA`). Server development harus sudah dipasang license; jika semua login selain Super-Admin ditolak 403
license, minta license ke tim backend.
