# Instruksi untuk Claude: Web Admin Secure Patrol (React)

Kamu akan membangun **web admin Secure Patrol** dengan **React** di repository ini. Web ini dipakai manajemen
dan admin keamanan untuk memantau patroli dan mengelola seluruh data. Backend sudah jadi dan sudah dites;
tugasmu membangun antarmuka yang memakainya dengan benar.

Baca seluruh dokumen ini sebelum menulis kode. Semua path, field, aturan akses, dan pesan error di bawah adalah
perilaku backend yang sebenarnya — jangan menebak field lain. Spesifikasi lengkap ada di `docs/openapi.yaml`
pada backend, dan bisa dicoba interaktif di `http://<server>/docs` saat backend berjalan dengan `APP_ENV=development`.

## 0. Cara kerja

1. **Pelajari repository dulu.** Jika sudah ada project React, ikuti struktur, library, dan pola yang ada.
2. Jika repository kosong, gunakan stack di bagian 2. Jangan menambahkan library di luar daftar tanpa alasan kuat.
3. Kerjakan bertahap sesuai urutan di bagian 14; pastikan setiap tahap berjalan sebelum lanjut.
4. Teks UI dalam **Bahasa Indonesia**. Nama kode, variabel, dan komentar dalam bahasa Inggris.
5. Jika butuh sesuatu dari backend yang tidak ada di dokumen ini, **jangan mengakalinya di frontend** — catat di
   laporan akhir (bagian 15).

## 1. Ruang lingkup

Web mencakup **semua fitur backend kecuali scan NFC** (scan hanya dari aplikasi mobile; backend menolak scan
dari web dengan 403).

| Menu | Isi |
|---|---|
| Dashboard | Shift berjalan: progres, status terakhir tiap titik NFC, peta, temuan tidak normal, scan terbaru; refresh otomatis |
| Monitoring Patroli | Daftar group (shift per tanggal) dengan progres → detail checklist dan scan per group |
| Titik per Shift/Periode | Daftar titik dari group yang dipilih dengan status sudah/belum di-scan |
| Riwayat Scan | Tabel dengan filter lengkap, detail scan (foto, peta, jarak, validasi), ekspor CSV |
| Laporan | Tingkat penyelesaian per hari per shift dan temuan tidak normal untuk rentang tanggal, grafik, ekspor CSV |
| Unit | Master data unit: kode, nama, lokasi di peta, status aktif (Super-Admin) |
| Area | Area besar dalam unit (gedung, lantai, parkir) untuk mengelompokkan titik patroli |
| Titik Patroli | CRUD dengan pemilih lokasi di peta |
| Pengaturan Shift | CRUD dengan visualisasi timeline 24 jam |
| Pengguna | CRUD, upload foto wajah, reset password, aktif/nonaktif |
| Role | CRUD |
| App Client | Buat App ID/Key, rotasi key, aktif/nonaktif, masa berlaku |
| Help Desk | CRUD artikel (Markdown dengan preview), draft/terbit, urutkan dengan drag & drop |
| Audit Log | Riwayat create/update/delete: waktu, pengguna, aksi, data, IP (Super-Admin & Manager Keamanan) |
| Tampilan Aplikasi | Nama dan logo aplikasi (Super-Admin) |
| License | Status license, Install ID, pemakaian unit/App Client, pasang license (Super-Admin) |
| Pengaturan Sistem | Radius scan, batas offline, akurasi wajah, lockout login, masa berlaku token, validasi foto wajah |
| Profil | Data diri, foto, ganti password, logout semua perangkat |

## 2. Stack (jika repository kosong)

- **Vite + React 18 + TypeScript** (strict).
- **React Router** untuk routing; **TanStack Query** untuk data server (cache, invalidasi setelah mutasi).
- **react-hook-form + zod** untuk form dan validasi.
- **Mantine** (komponen, tabel, date picker, dropzone, notifikasi, modal) — atau pustaka UI yang sudah ada di repo.
- **@noble/hashes** untuk SHA-256/HMAC (bekerja juga di HTTP non-HTTPS saat development, tidak seperti `crypto.subtle`).
- **react-leaflet + Leaflet** dengan tile OpenStreetMap (sertakan atribusi) untuk peta.
- **@dnd-kit** untuk drag & drop urutan help desk.
- **react-markdown + remark-gfm** untuk render Markdown (tanpa HTML mentah).
- **dayjs** dengan plugin `utc` dan `timezone` untuk tanggal.
- **Recharts** untuk grafik laporan.
- **Vitest + Testing Library** untuk test.

## 3. Konfigurasi

| Variabel | Contoh | Keterangan |
|---|---|---|
| `VITE_API_BASE_URL` | `https://api.example.com` | Tanpa trailing slash; semua path API diawali `/api/v1` |
| `VITE_APP_ID` | `sp_web_pol5uwnk5ctvbpzi_2idbmv` | App Client platform **`web`** |
| `VITE_APP_KEY` | `spk_...` | Untuk menandatangani request |

- App Client **harus ber-platform `web`**: fitur manajemen (user, role, app client, titik patroli, shift,
  help desk) menolak platform lain dengan 403 `This feature is only available on the web platform`.
- App Client dibuat oleh backend: `go run . create-app-client -name "Secure Patrol Web" -platform web`.
- Karena berjalan di browser, App Key ikut terbawa di bundle JavaScript dan **tidak benar-benar rahasia**.
  Perlindungan sebenarnya adalah login user, role, dan pembatasan platform. Gunakan App Client web tersendiri
  (terpisah dari mobile) agar bisa dirotasi tanpa mengganggu aplikasi lain. Jangan commit file `.env`.
- CORS sudah diizinkan dari semua origin oleh backend, termasuk header `Authorization`, `X-App-Id`,
  `X-Timestamp`, `X-Nonce`, `X-Signature`.

## 4. Kontrak API umum

### 4.1 Format response

```json
{ "meta": { "message": "Get users success", "code": 200, "status": "success" }, "data": { } }
```

Daftar berhalaman:

```json
{ "data": { "items": [ ], "pagination": { "page": 1, "limit": 10, "total": 42, "total_pages": 5 } } }
```

Query standar: `page` (mulai 1), `limit` (1–100; nilai di luar rentang menjadi 10), `search`.

Error: `meta.status = "Error"`, `meta.message` berisi pesan (bahasa Inggris), `data` bisa `null`, string, atau
array pesan validasi seperti `["email failed on 'email' validation"]` (nama field memakai nama JSON/form).

### 4.2 Tanda tangan request (wajib untuk SEMUA request ke `/api/v1`)

Setiap request, termasuk login, membawa:

| Header | Isi |
|---|---|
| `X-App-Id` | App ID |
| `X-Timestamp` | Unix time dalam **detik**, dari jam yang sudah dikoreksi (4.3) |
| `X-Nonce` | 16–64 karakter acak, baru untuk setiap request dan setiap retry (misalnya 16 byte `crypto.getRandomValues` → hex) |
| `X-Signature` | Hex lowercase dari HMAC-SHA256 di bawah |

```
payload   = METHOD + "\n" + REQUEST_URI + "\n" + X_TIMESTAMP + "\n" + X_NONCE + "\n" + hex(SHA256(BODY))
signature = hex( HMAC_SHA256(key = UTF8(APP_KEY), message = UTF8(payload)) )
```

- `REQUEST_URI` = `pathname + search` persis seperti yang dikirim, contoh `/api/v1/users?page=2&search=budi`.
  Bangun URL dengan `URLSearchParams` lebih dulu, lalu tanda tangani string yang sama.
- `BODY` = byte yang benar-benar dikirim. Tanpa body → SHA-256 string kosong.
- **Multipart (`FormData`):** browser membangun ulang body multipart dengan boundary acak jika kamu mengirim
  `FormData` langsung, sehingga signature tidak cocok. Serialisasikan dulu:

  ```ts
  const response = new Response(formData);
  const contentType = response.headers.get("content-type")!; // berisi boundary
  const bytes = new Uint8Array(await response.arrayBuffer());
  // tanda tangani `bytes`, lalu kirim:
  fetch(url, { method, headers: { ...signed, "Content-Type": contentType }, body: bytes });
  ```

  **Jangan** membungkus `bytes` ke dalam `Blob` dengan `type` berisi boundary lalu mengandalkan browser untuk
  mengisi `Content-Type`: browser mengubah `type` Blob menjadi huruf kecil, sedangkan boundary Chrome
  (`----WebKitFormBoundaryAbC...`) mengandung huruf besar. Server lalu tidak menemukan batas part dan menjawab
  400 `cannot read multipart/form-data body: multipart: NextPart: EOF`. Kirim `Uint8Array` dengan header
  `Content-Type` eksplisit seperti contoh di atas.

- Buat satu fungsi `apiFetch` yang dipakai **semua** request (termasuk unduhan gambar) dan menambahkan signature,
  `Authorization`, penanganan 401, dan parsing envelope.

**Test vector** (dihitung oleh kode backend; wajib jadi unit test):

App Key uji: `spk_TEST-ONLY-4pZt7Qm2Xv9Lr3Ks8Wd1Nf6Hy0Bc5Ja`

| # | Method & URI | Timestamp | Nonce | Body | Signature |
|---|---|---|---|---|---|
| 1 | `POST /api/v1/auth/login` | `1790668800` | `0123456789abcdef0123456789abcdef` | `{"email":"tim@securepatrol.local","password":"Password123"}` | `f5681cea7af562190e884bc0b1aff7a303367cf687ede9ec9c70d6e2c64da48c` |
| 2 | `GET /api/v1/patrol-groups/current` | `1790668800` | `fedcba9876543210fedcba9876543210` | (kosong) | `e4ac3c5d488438c9e829dfd0091a6eb540d3bd4fc4682f85195859e15b7f1f12` |
| 3 | `GET /api/v1/help-desk-articles?category=faq&page=1` | `1790668801` | `a1b2c3d4e5f60718293a4b5c6d7e8f90` | (kosong) | `ccd21a96183515fc2ae3ad4aed4de79b217933250e232f99128a6892c767c2c3` |

SHA-256 body #1 = `511f90493776b18136558905986a0abb6bb1e4bdc68e5bcd1374196004f0fc24`.

Implementasi referensi yang sudah terbukti bekerja: `docs/signer.js` di backend (dipakai halaman `/docs`).

### 4.3 Koreksi jam

Server menolak request jika `X-Timestamp` selisih lebih dari 300 detik, dan jam komputer pengguna bisa meleset.
Simpan `clockOffset = serverTime − Date.now()`. Ambil `serverTime` dari header **`Date`** pada response mana pun
(backend mengekspos header ini untuk CORS, termasuk pada response 401 dan `GET {BASE_URL}/health` yang tidak butuh
autentikasi) atau dari `config.server_time`. Saat aplikasi dimuat, panggil `GET /health` sekali untuk mengisi
offset. Jika menerima 401 `"Unauthorized app"` dengan `data = "request timestamp is invalid or outside the allowed window"`,
perbarui offset dari header `Date` response tersebut lalu ulangi **sekali**.

### 4.4 Penanganan status HTTP

| Kondisi | Tindakan |
|---|---|
| 401 `meta.message = "Unauthorized app"` | Masalah App ID/Key, jam, atau nonce. Jam → koreksi & ulang sekali; nonce → ulang dengan nonce baru; lainnya → tampilkan "Konfigurasi aplikasi tidak valid". **Jangan logout** |
| 401 `meta.message = "Unauthorized"`, `data = "Token is expired"` | Refresh (5.2) lalu ulangi request |
| 401 `"Unauthorized"` dengan `data` lain | Coba refresh sekali; gagal → sesi berakhir, arahkan ke login |
| 403 `your unit is inactive, contact the head office` | Unit user dinonaktifkan: akhiri sesi, kembali ke login dengan pesan "Unit Anda sedang dinonaktifkan, hubungi pusat" |
| 403 `license is not active, contact your administrator` | License tidak aktif: lihat 10.10 (akhiri sesi / arahkan Super-Admin ke halaman License) |
| 403 `this app client exceeds the license limit` | Tampilkan halaman "Aplikasi ini melebihi batas license" (10.10) |
| 403 lainnya | Tampilkan halaman/notifikasi "Anda tidak memiliki akses" |
| 404 | "Data tidak ditemukan" |
| 409 | Konflik data (email/kode NFC/kode role/kode unit sudah dipakai, shift tumpang tindih, unit masih berisi data) — tampilkan pada field terkait |
| 413 | Body > 20 MB |
| 422 | Validasi / aturan bisnis — petakan ke field form jika bisa, jika tidak tampilkan `meta.message` |
| 429 | Akun terkunci 5 menit setelah 3 kali salah password (login). Tampilkan hitung mundur dari `data.retry_after_seconds` (juga header `Retry-After`); `data.locked_until` berisi waktu terbuka |
| 5xx / jaringan | "Terjadi gangguan, coba lagi" dengan tombol ulangi |

### 4.5 Gambar terproteksi

`/api/v1/users/{id}/face-photo`, `/api/v1/auth/me/face-photo`, dan `/api/v1/patrol-scans/{id}/photos/{photo_id}`
memerlukan signature dan token, sehingga **tidak bisa dipakai langsung di `<img src>`**. Buat komponen
`<SecureImage path=... />` yang mengambil gambar lewat `apiFetch` sebagai blob, menampilkan
`URL.createObjectURL(blob)`, dan memanggil `URL.revokeObjectURL` saat unmount. Cache per path dengan TanStack Query.
Foto wajah adalah data biometrik: jangan simpan di localStorage atau cache persisten.

## 5. Autentikasi & sesi

### 5.1 Login — `POST /api/v1/auth/login`

Request `{"email", "password"}` (trim email). Response berisi `access_token`, `expires_at`, `refresh_token`,
`user` (termasuk `role.code`, `unit_id`, `unit` — `null` untuk user pusat —, `face_photo_base64`,
`face_photo_mime_type`), `settings` (sesuai unit user), dan `config`:

```json
"config": {
  "server_time": "2026-09-29T15:29:58+07:00",
  "timezone": "Asia/Jakarta",
  "request_timestamp_tolerance_seconds": 300,
  "location_radius_meters": 100,
  "face_match_min_score": null,
  "max_offline_hours": 24,
  "max_scan_photos": 3,
  "max_photo_size_bytes": 5242880
}
```

- Tampilkan avatar dari `face_photo_base64` (`data:{mime};base64,...`), hanya di memori.
- Error: 401 `email or password is incorrect`, 403 `account is inactive`, 403 `your unit is inactive, contact the
  head office`, 429 akun terkunci, 422 validasi.
- **Web hanya untuk Super-Admin, Manager Keamanan, Kepala Keamanan, dan Admin Keamanan.** Role lain (Tim Keamanan
  dan role kustom) ditolak server dengan 403 `your role is not allowed to sign in on this platform` — tampilkan
  "Akun Anda hanya dapat digunakan melalui aplikasi mobile." Jika pesan yang sama muncul pada request lain atau saat
  refresh (role user diubah), akhiri sesi dan kembali ke halaman login dengan pesan tersebut.

### 5.2 Token dan multi-tab

- **Access token di memori** (state/store), bukan di localStorage.
- **Refresh token** di `localStorage` agar sesi bertahan saat reload. Karena itu terapkan Content Security Policy
  yang ketat dan **jangan pernah** render HTML mentah (lihat help desk).
- **Refresh token hanya bisa dipakai sekali**; memakai token lama mematikan seluruh sesi. Browser bisa membuka
  banyak tab, jadi refresh harus single-flight **lintas tab**:
  - Gunakan Web Locks API (`navigator.locks.request("sp-refresh", ...)`). Di dalam lock, baca ulang refresh token
    dari localStorage; jika sudah berbeda dari yang dipakai sebelumnya (tab lain sudah refresh), pakai hasil tab
    itu dan jangan refresh lagi.
  - Siarkan token baru ke tab lain lewat `BroadcastChannel("sp-auth")`; logout juga disiarkan.
- Refresh proaktif ~1 menit sebelum `expires_at`, dan reaktif saat 401 `Token is expired`.
- Refresh gagal → hapus sesi, arahkan ke `/login?expired=1` dengan pesan "Sesi berakhir, silakan login kembali",
  lalu kembalikan ke halaman semula setelah login.

### 5.3 Endpoint sesi lainnya

| Endpoint | Pakai untuk |
|---|---|
| `GET /api/v1/auth/me` | Profil saat aplikasi dimuat |
| `GET /api/v1/app-config` | Konfigurasi dan jam server saat aplikasi dimuat |
| `GET /api/v1/auth/me/face-photo` | Foto profil (gambar, lihat 4.5) |
| `POST /api/v1/auth/logout` | Logout perangkat ini |
| `POST /api/v1/auth/logout-all` | "Keluar dari semua perangkat" di Profil |
| `PUT /api/v1/auth/change-password` | `{"old_password","password","password_confirmation"}`; perangkat lain otomatis logout, sesi ini tetap |

## 6. Unit, hak akses & menu

**Pusat dan unit.** Role dari `user.role.code`, unit dari `user.unit` (`null` = user pusat).

- **User pusat** (`super_admin`, `security_manager`) melihat data **semua unit**. Tampilkan **pemilih unit** global di
  header ("Semua unit" + daftar dari `GET /api/v1/units?limit=100`) dan kirim sebagai `unit_id` ke endpoint daftar:
  users, patrol-points, patrol-shifts, patrol-groups, patrol-list-items, patrol-scans, export, settings, audit-logs.
  Tampilkan kolom **Unit** di tabel saat "Semua unit" dipilih. Dashboard pusat: ringkasan per unit (group berjalan
  setiap unit via `GET /api/v1/patrol-groups/current?unit_id=`), lalu drill-down ke satu unit.
- **User unit** (`security_head`, `security_admin`) hanya melihat unitnya. Sembunyikan pemilih unit; tampilkan
  nama unit di header. Server mengabaikan `unit_id` dari mereka.
- **Manager Keamanan hanya memantau**: semua halaman dalam mode baca.
- Kepala Keamanan dan Admin Keamanan saat ini punya hak yang sama, tetapi tetap role berbeda.

Hanya empat role berikut yang bisa login di web (bagian 5.1). Tampilkan menu dan tombol sesuai tabel ini; tetap
tangani 403 dari server.

| Fitur | super_admin | security_manager | security_head / security_admin |
|---|---|---|---|
| Dashboard, Monitoring, Titik per Shift, Riwayat Scan, Laporan, Export | semua unit | semua unit | unit sendiri |
| Unit: lihat | ✓ | ✓ | unit sendiri |
| Unit: tambah/ubah/nonaktifkan/hapus | ✓ | | |
| Titik Patroli: lihat | semua unit | semua unit | unit sendiri |
| Titik Patroli: tambah/ubah/hapus | | | ✓ |
| Pengaturan Shift: lihat | semua unit | semua unit | unit sendiri |
| Pengaturan Shift: kelola | | | ✓ |
| Pengguna: lihat | semua | semua | unit sendiri |
| Pengguna: tambah/ubah/hapus/reset password | ✓ (termasuk user pusat) | | ✓ (role unit, unit sendiri) |
| Role: lihat | ✓ | ✓ | ✓ |
| Role: tambah/ubah/hapus | ✓ | | |
| App Client | ✓ | | |
| Audit Log | ✓ | ✓ | |
| Pengaturan Sistem: lihat | global / per unit | global / per unit | unit sendiri |
| Pengaturan Sistem: ubah | nilai global | | nilai unit sendiri |
| Help Desk: baca termasuk draft | ✓ | ✓ | terbit saja |
| Help Desk: kelola & urutkan | ✓ | | |
| Profil & ganti password | ✓ | ✓ | ✓ |

Aturan tambahan dari server:

- Hanya Super-Admin yang bisa membuat/mengubah/menghapus user pusat (Super-Admin, Manager Keamanan) dan memilih role
  tersebut (sembunyikan opsinya untuk role lain; server menjawab 403 `you are not allowed to manage users with this role`).
- Data unit lain dijawab 404 untuk user unit.
- User tidak bisa menghapus, menonaktifkan, atau mengganti role akunnya sendiri (422) — nonaktifkan kontrolnya.
- App Client yang sedang dipakai web tidak bisa dihapus/dinonaktifkan (422) — bandingkan `app_id` dengan `VITE_APP_ID`.
- Role sistem (`is_system = true`) tidak bisa dihapus/dinonaktifkan; kode role tidak bisa diubah.

## 7. Konvensi data

- Semua waktu dari server berformat RFC3339. Tampilkan dalam zona `config.timezone` (Asia/Jakarta), format
  `DD MMM YYYY HH:mm`.
- Filter tanggal memakai `YYYY-MM-DD` (`date_from`, `date_to`, inklusif) dan berlaku pada **tanggal shift**
  (`shift_date`), yaitu tanggal shift dimulai.
- Semua `DELETE` adalah **soft delete**: tampilkan dialog konfirmasi "Data akan dihapus dan tidak lagi tampil."
  Email, kode NFC, dan kode role milik data yang dihapus bisa dipakai lagi.
- Kondisi scan: `normal` → "Normal", `abnormal` → "Tidak Normal" (warna merah).
- Kategori help desk: `rule` → "Aturan", `guide` → "Tata Cara", `faq` → "FAQ".

## 8. Dashboard

- `GET /api/v1/patrol-groups/current` → group berjalan unit user beserta `items`. User pusat wajib menambahkan
  `?unit_id=` (tanpa itu 422 `unit_id is required`). Contoh item:

  ```json
  {
    "id": 2, "patrol_point_id": 2, "name": "Parking Area", "location": "Basement 1",
    "nfc_code": "04:A2:1F:9C", "latitude": -6.2252431, "longitude": 106.8011502,
    "is_location_match_required": true, "is_face_validation_required": false,
    "is_scanned": true, "scan_count": 1, "last_scanned_at": "2026-09-29T15:29:58+07:00",
    "last_scanned_by": { "id": 6, "name": "Budi Santoso", "email": "budi@securepatrol.local" },
    "last_condition": "abnormal"
  }
  ```

  Group juga berisi `unit` (`id`, `code`, `name`), `shift`, `shift_date`, `start_at`, `end_at`, `status` (`upcoming`/`ongoing`/`finished`), dan
  `progress` (`total_points`, `scanned_points`, `unscanned_points`, `total_scans`, `abnormal_scans`).
- Tampilkan: kartu ringkasan (shift, sisa waktu, progres %, temuan tidak normal), tabel titik dengan status
  terakhir (Belum di-scan / Normal / Tidak Normal, waktu dan petugas), dan **peta** titik dengan warna status
  (abu-abu belum, hijau normal, merah tidak normal).
- Panel "Scan terbaru": `GET /api/v1/patrol-scans?limit=10`.
- Refresh otomatis tiap 30 detik (hanya saat tab terlihat) dan tombol refresh manual.
- 422 `there is no active patrol shift at the scan time` → "Tidak ada shift aktif saat ini".

## 9. Monitoring, Titik per Shift/Periode, Riwayat Scan, Laporan

**Monitoring Patroli** — `GET /api/v1/patrol-groups?shift_id=&date_from=&date_to=&page=&limit=` (terbaru dulu):
tabel group dengan progress bar dan jumlah temuan. Detail `GET /api/v1/patrol-groups/{id}`: checklist item dan tab
scan (`GET /api/v1/patrol-scans?group_id={id}`).

**Titik per Shift/Periode** — `GET /api/v1/patrol-list-items?group_id=&shift_id=&date_from=&date_to=&status=scanned|unscanned&search=`.
Setiap item membawa `group` (`shift_name`, `shift_date`, `start_at`, `end_at`). Sorot titik yang belum di-scan
pada group yang sudah selesai.

**Riwayat Scan** — `GET /api/v1/patrol-scans` dengan filter `group_id`, `shift_id`, `date_from`, `date_to`,
`scanned_by`, `patrol_point_id`, `condition` (`normal`/`abnormal`); terbaru dulu. Tanpa filter = seluruh history.
Kolom: waktu scan, shift, titik, kondisi, catatan, petugas, jarak, validasi lokasi/wajah, jumlah foto.

Detail `GET /api/v1/patrol-scans/{id}` dalam drawer:

- Galeri foto (`photos[].url`, via `<SecureImage>`) dengan lightbox.
- Peta: posisi titik (dari `patrol-points`), posisi scan (`latitude`, `longitude`), lingkaran radius
  `config.location_radius_meters`, dan `distance_meters`.
- `is_location_valid`, `is_face_verified`, `face_match_score`.
- `scanned_at` (waktu di HP) dan `received_at` (waktu diterima server); tandai "Dikirim offline" jika selisihnya
  lebih dari 5 menit.

Filter `scanned_by` memakai daftar petugas dari `GET /api/v1/users` (semua role web bisa membacanya; user unit
otomatis hanya mendapat petugas unitnya). Semua endpoint di bagian ini menerima `unit_id` untuk user pusat.

**Ekspor Excel** — tombol "Export Excel" membuka dialog filter: **Shift**, **Titik**, **Petugas**, **Tanggal shift
dari** dan **sampai** (isian awal mengikuti filter halaman). Unduh lewat
`GET /api/v1/patrol-scans/export?shift_id=&patrol_point_id=&scanned_by=&date_from=&date_to=&include_photos=` menggunakan `apiFetch`
(butuh signature dan token, jadi tidak bisa memakai link biasa): ambil sebagai blob, lalu simpan dengan nama dari
header `Content-Disposition`. User pusat juga bisa memilih **Unit** (`unit_id`). File `.xlsx` berisi sheet
"Riwayat Scan" (kolom Waktu scan, Diterima server, Dikirim offline, Tanggal shift, Unit, Shift, Area, Titik, Lokasi, Kondisi, Catatan, Petugas, Email petugas; plus Foto 1–3 jika dengan foto) dan sheet "Filter". Tampilkan
loading selama unduhan.

- **Tanggal shift dari/sampai wajib.** Rentang maksimal diambil dari `GET /api/v1/settings` (semua role web bisa
  membacanya): `export_max_range_days` (default 7) dan `export_photo_max_range_days` (default 1). Batasi pemilih
  tanggal sesuai nilai itu dan tampilkan keterangan "Maksimal N hari".
- Checkbox **"Sertakan foto"** (`include_photos=true`): menambah kolom Foto 1–3 berisi thumbnail. Saat dicentang,
  rentang otomatis dibatasi ke `export_photo_max_range_days` (default **1 hari**: tanggal dari = sampai), dengan
  keterangan "File lebih besar dan proses lebih lama". Tampilkan loading tanpa batas waktu singkat (bisa ±30 detik).
- Error 422: `date_from and date_to are required for an export`, `date_from must not be after date_to`,
  `an export can cover at most N days, ...`, `an export with photos can cover only 1 day, ...`,
  `export is limited to 50000 rows, ...`, `an export with photos is limited to 2000 rows, ...`. Tampilkan
  pesan dalam Bahasa Indonesia di dialog.

**Area di monitoring dan laporan** — item daftar patroli, scan (`patrol_point.area_name`), dan rekap per titik
membawa `area_id` + `area_name` (nama area saat shift berjalan). Kelompokkan daftar titik per area (dengan subtotal),
dan tambahkan filter **Area** (`area_id`) di Titik per Shift, Riwayat Scan, Export Excel (kolom **Area** setelah
Shift; sheet Filter berisi baris Area), dan rekap per titik.

**Total patroli per titik** — `GET /api/v1/patrol-point-summary?group_id=` (satu shift pada satu tanggal) atau
`?shift_id=&date_from=&date_to=` (satu shift di rentang tanggal; unit mengikuti shift), opsional `&area_id=`. Response: `unit`, `shift`,
`groups`, `totals` (`points`, `scanned_points`, `unscanned_points`, `total_scans`, `abnormal_scans`), dan `items`
per titik (`name`, `location`, `nfc_code`, `total_scans`, `normal_scans`, `abnormal_scans`, `officers`,
`scanned_groups`/`groups`, `first_scanned_at`, `last_scanned_at`), termasuk titik yang belum pernah di-scan.
Pakai di:
- detail group di Monitoring (tab **Rekap per titik**, `group_id`);
- halaman **Laporan** (pilih shift + rentang tanggal → tabel dan grafik batang total scan per titik, sorot titik
  dengan `total_scans = 0` atau `scanned_groups < groups`), dengan ekspor CSV.
Error 422 `group_id or shift_id is required`; shift/group unit lain 404.

**Laporan** — untuk rentang tanggal (maks. 31 hari), ambil `GET /api/v1/patrol-groups?date_from=&date_to=&limit=100`
(semua halaman). Tampilkan tabel per tanggal × shift: `scanned_points/total_points` (%), `total_scans`,
`abnormal_scans`; grafik penyelesaian harian; dan ekspor CSV.

## 10. Master data

### 10.0 Unit — `/api/v1/units`

- Daftar: `GET ?page=&limit=&search=&is_active=`. Kolom: kode, nama, status, `users_count`, `patrol_points_count`.
  User unit hanya mendapat unitnya sendiri.
- Form (Super-Admin, JSON): `code` (wajib, maks 20, disimpan huruf besar, unik), `name` (wajib, maks 150),
  `latitude`, `longitude` (wajib; pilih di peta), `is_active` (default true saat create, wajib saat update).
- Unit baru otomatis mendapat 3 shift default (Kepala/Admin unit mengubahnya).
- **Nonaktifkan** unit: dialog peringatan "Semua pengguna unit ini akan langsung keluar dan tidak bisa login".
- Hapus: hanya unit kosong; 409 `unit still has users or patrol points, move or delete them first` → sarankan
  menonaktifkan. 409 `unit code is already used by another unit` → error di field `code`.

### 10.0a Area — `/api/v1/patrol-areas`

Area membagi titik patroli satu unit ke area besar (gedung, lantai, parkir). Di API namanya `area` karena "group"
sudah dipakai untuk group patroli (shift per tanggal); label di UI bebas, misalnya "Area".

- Daftar: `GET ?page=&limit=&search=&unit_id=` → `id`, `name`, `description`, `patrol_points_count`.
- Tambah/ubah (Kepala/Admin Keamanan, JSON): `name` (wajib, maks 100, unik per unit tanpa membedakan huruf besar/kecil),
  `description` (maks 255). 409 `an area with this name already exists in the unit` → error di field `name`.
- Hapus: hanya area tanpa titik; 409 `area still has patrol points, move them to another area first` → tampilkan
  pesan dan tautan ke daftar titik area itu.
- Pusat (Super-Admin/Manager) hanya melihat.

### 10.1 Titik Patroli — `/api/v1/patrol-points`

- Daftar: `GET ?page=&limit=&search=&unit_id=&area_id=` (search nama, lokasi, kode NFC). Tambah/ubah/hapus hanya
  untuk Kepala/Admin Keamanan; titik otomatis masuk ke unit mereka. Pusat hanya melihat.
- Kolom/field **Area**: `area` (`{id, name}` atau null). Di form, select area dari `GET /api/v1/patrol-areas`
  (`area_id`, opsional; kosong = tanpa area). `PUT` tanpa `area_id` mengeluarkan titik dari area.
  422 `area not found in this unit` → error di field area. Filter daftar per area.
- Form (JSON):

  | Field | Aturan |
  |---|---|
  | `name` | wajib, maks 150 |
  | `location` | wajib, maks 255 (deskripsi lokasi, mis. "Gedung A Lantai 2") |
  | `nfc_code` | wajib, maks 100; UID tag heksadesimal huruf besar dipisah titik dua, mis. `04:A2:1F:9C`; server menyimpan dalam huruf besar |
  | `latitude` | wajib, −90..90 (0 valid) |
  | `longitude` | wajib, −180..180 (0 valid) |
  | `is_location_match_required` | boolean (default false saat create, wajib saat update) |
  | `is_face_validation_required` | boolean (default false saat create, wajib saat update) |

- **Pemilih lokasi**: peta dengan marker yang bisa digeser, input koordinat manual, dan lingkaran radius
  `config.location_radius_meters` sebagai pratinjau area scan yang diterima.
- 409 `nfc code is already used by another patrol point` → error pada field `nfc_code` (kode NFC unik di semua unit).
- Posisi awal peta untuk titik baru: koordinat unit (`user.unit.latitude/longitude`).
- `PUT` mengirim semua field; `DELETE` soft delete. Titik baru otomatis masuk ke daftar patroli shift yang
  sedang berjalan; perubahan tidak mengubah history shift yang sudah selesai.

### 10.2 Pengaturan Shift — `/api/v1/patrol-shifts`

- `GET ?unit_id=` mengembalikan array (tanpa paginasi), urut unit lalu jam mulai. Field: `unit_id`, `name`,
  `start_time`, `end_time`, `duration_minutes`, `crosses_midnight`, `is_active`.
- Setiap unit punya shift sendiri. Kelola hanya untuk Kepala/Admin Keamanan (shift dibuat di unitnya); pusat
  melihat saja (pilih unit untuk melihat timeline-nya).
- Form: `name` (wajib, maks 100), `start_time` dan `end_time` format `HH:MM` (`end_time` boleh `24:00`;
  jam akhir lebih kecil dari jam mulai berarti melewati tengah malam), `is_active`.
- Jelaskan di UI: **jam akhir adalah batas (cut-off)** — scan tepat di jam akhir masuk shift berikutnya; perubahan
  hanya berlaku untuk shift berikutnya.
- Tampilkan **timeline 24 jam** semua shift aktif agar celah dan tumpang tindih terlihat.
- 409 `patrol shift overlaps with another active shift: <nama> (<mulai>-<selesai>)` (dicek dalam satu unit); 422 untuk format jam.

### 10.3 Pengguna — `/api/v1/users`

- Daftar: `GET ?page=&limit=&search=&role_id=&is_active=&unit_id=&head_office=` (search nama/email;
  `head_office=true` = hanya user pusat, untuk user pusat). Kolom: foto, nama, email, role, unit, status aktif,
  terkunci (`is_locked`), login terakhir.
- **Unit di form**: untuk Super-Admin, tampilkan pilihan `unit_id` jika role yang dipilih adalah role unit (wajib,
  422 `unit_id is required for this role`); sembunyikan untuk role pusat. Untuk Kepala/Admin Keamanan, jangan tampilkan
  pilihan unit (user selalu dibuat di unitnya) dan sembunyikan role pusat dari pilihan role. Memindahkan user ke unit
  lain (hanya Super-Admin) mengakhiri semua sesinya.
- **Tambah** (`POST`, **multipart**): `name` (wajib, maks 150), `email` (wajib, unik), `password`,
  `password_confirmation` (8–72 karakter, minimal satu huruf dan satu angka), `role_id`, `face_photo`
  (**wajib**, JPEG/PNG, **maks 5 MB**; cek ukuran dan tipe sebelum upload; tampilkan pratinjau), `is_active` (opsional).
- **Ubah** (`PUT`, multipart): `name`, `email`, `role_id`, `is_active` (wajib), `face_photo` (opsional, mengganti foto).
  Menonaktifkan user atau mengganti role-nya langsung mengakhiri semua sesinya — tampilkan peringatan di dialog.
- **Reset password** (`PUT /users/{id}/reset-password`, JSON `{"password","password_confirmation"}`): juga
  membuka kunci akun dan mengakhiri semua sesinya.
- Foto: `GET /users/{id}/face-photo` via `<SecureImage>`.
- **Foto wajah diperiksa server** saat tambah dan saat ganti foto: harus tepat satu wajah, cukup besar, dan menghadap
  kamera. Tampilkan panduan di samping kolom upload ("Foto close-up, menghadap kamera, satu orang, pencahayaan
  terang") dan tampilkan pesan penolakan server di bawah kolom foto dalam Bahasa Indonesia:

  | Pesan server | Tampilkan |
  |---|---|
  | `no face detected in the photo, ...` | Wajah tidak terdeteksi. Gunakan foto wajah yang jelas dan terang. |
  | `the photo must contain exactly one face, N faces were found` | Foto berisi N wajah. Gunakan foto satu orang saja. |
  | `the face is too small, ...` | Wajah terlalu kecil. Gunakan foto yang lebih dekat. |
  | `the face must look straight at the camera with both eyes visible` | Wajah harus menghadap kamera dengan kedua mata terlihat. |
  | `face photo could not be read as an image` | File foto rusak atau tidak bisa dibaca. |

- Error: 409 `email is already registered`; 422 `face photo is required`, `role not found or inactive`,
  `unit_id is required for this role`, `unit not found`,
  `file size must not exceed 5 MB`, `file must be a JPEG or PNG image`,
  `you cannot delete, deactivate or change the role of your own account`; 403 untuk user Super-Admin.

### 10.4 Role — `/api/v1/roles`

- Role berlaku untuk semua unit. Semua role web bisa membaca (untuk form pengguna); hanya Super-Admin yang mengelola.
- Daftar berhalaman. Form tambah: `code` (wajib, 3–50 karakter, huruf kecil/angka/underscore, diawali huruf;
  tidak bisa diubah), `name` (wajib, maks 100), `description` (maks 255), `is_active`. Ubah: `name`,
  `description`, `is_active` (wajib).
- Role sistem: sembunyikan tombol hapus dan kunci toggle aktif.
- Error: 409 `role code is already used`; 400 `system role cannot be deleted or deactivated`,
  `role is still assigned to users`.

### 10.5 App Client — `/api/v1/app-clients` (Super-Admin)

- Kolom: nama, platform, `app_id` (dengan tombol salin), `key_hint` (mis. `****s0yM`), aktif, `expires_at`,
  `last_used_at`, `key_rotated_at`, `previous_key_expires_at`.
- **Buat** (`POST`): `name`, `platform` (`android`/`ios`/`web`/`server`), `description`, `expires_at` (opsional).
- **Rotasi key** (`POST /app-clients/{id}/rotate-key`, `{"grace_period_hours": 0..168}`): jelaskan bahwa selama
  masa tenggang key lama masih diterima; `0` langsung mematikan key lama (untuk key bocor).
- Response buat & rotasi berisi **`app_key` yang hanya ditampilkan sekali**: tampilkan di modal yang tidak bisa
  ditutup sebelum user mencentang "Saya sudah menyimpan App Key", dengan tombol salin. Jangan simpan App Key di
  state global, cache, atau log.
- Ubah: `name`, `description`, `is_active` (wajib), `expires_at`.

### 10.6 Help Desk — `/api/v1/help-desk-articles`

- Tab **Aturan / Tata Cara / FAQ** (`category=rule|guide|faq`), pencarian, dan filter terbit/draft
  (`is_published=true|false`, hanya untuk Super-Admin dan Manager Keamanan). Help desk berlaku untuk semua unit
  dan **hanya dikelola Super-Admin**; role lain membaca artikel terbit saja. Urutan dari server sudah final; jangan diurutkan ulang.
- Form: `category`, `title` (maks 200), `content` (Markdown, maks 20.000), `is_published`, `sort_order` (opsional;
  kosong = paling akhir). Editor dengan tab **Tulis / Pratinjau**; pratinjau memakai renderer yang sama dengan tampilan baca.
- Render Markdown dengan `react-markdown` + `remark-gfm` **tanpa** `rehype-raw` dan tanpa `dangerouslySetInnerHTML`.
- **Urutkan** dengan drag & drop per kategori, lalu simpan lewat
  `PUT /api/v1/help-desk-articles/reorder` dengan `{"category": "guide", "article_ids": [7, 4, 6, 5]}`.
  Daftar harus memuat **semua** artikel kategori itu, termasuk draft — ambil daftar lengkap (`limit=100`, tanpa
  filter pencarian/status) sebelum mengizinkan drag. Terapkan urutan secara optimistic dan kembalikan jika server
  menjawab 422 `article_ids must contain every article of the category exactly once`.
- Draft ditandai jelas ("Draft") dan hanya terlihat oleh Super-Admin dan Manager Keamanan.

### 10.7 Audit Log — `GET /api/v1/audit-logs` (Super-Admin, Manager Keamanan)

- Hanya-baca; server mencatat otomatis setiap create/update/delete yang berhasil (tidak termasuk login/logout).
- Filter: `user_id`, `unit_id` (perubahan oleh user unit tersebut), `action` (`create`/`update`/`delete`), `resource` (mis. `users`, `patrol-points`,
  `help-desk-articles`, `app-clients`, `patrol-scans`), `date_from`, `date_to`, `search` (path, IP, atau ID data).
- Kolom: waktu, pengguna (`user.name`, `user.email`, `user.role_code`; `null` untuk aksi dari CLI dengan
  `source = "cli"`), aksi (badge warna), `endpoint` (mis. `PUT /users/:id`), `resource_id`, `ip_address`,
  `app_platform`, `unit_id` (unit pelaku; `null` = pusat/CLI), dan `user_agent` di detail.

### 10.8 Pengaturan Sistem — `/api/v1/settings`

- `GET` mengembalikan array setting: `key`, `level` (`global`/`unit`), `group` (`patrol`/`face`/`security`),
  `type` (`integer`/`number`/`boolean`), `value`, `default_value`, `is_default`, `global_value`, `is_inherited`,
  `min`, `max`, `unit`, `description`, `updated_by`, `updated_at`.
- **Setting per unit.** Super-Admin mengubah **nilai global** (dipakai semua unit yang tidak punya nilai sendiri);
  pusat bisa melihat setting sebuah unit dengan `?unit_id=` (baca saja). Kepala/Admin Keamanan mengubah **nilai
  unitnya**; tampilkan badge "Mengikuti pusat" jika `is_inherited`, nilai pusat (`global_value`) sebagai
  pembanding, dan tombol **Ikuti nilai pusat** (kirim `null`).
- Tampilkan per grup dengan label Bahasa Indonesia, input sesuai `type` (switch untuk boolean) dan batas
  `min`/`max`, penanda "diubah dari default", dan tombol **Kembalikan ke default** per baris.
- Simpan dengan `PUT` `{"values": {"patrol_location_radius_meters": 150}}` (hanya yang berubah; `null` = default
  untuk nilai global, atau kembali mengikuti pusat untuk nilai unit).
  Semua atau tidak sama sekali: 422 berisi daftar pesan per setting.
- Manager Keamanan melihat halaman dalam mode baca.
- Setelah menyimpan `access_token_ttl_minutes` atau `refresh_token_ttl_days`, nilai baru berlaku untuk login berikutnya.

### 10.9 Tampilan Aplikasi (nama & logo) — `/api/v1/branding`

- `GET /api/v1/branding` **tidak butuh login** (cukup signature App Client): `app_name`, `logo_base64`,
  `logo_mime_type`, `logo_updated_at`, `updated_at`. `logo_base64 = null` → pakai logo bawaan web.
- Ambil saat aplikasi dibuka (sebelum halaman login) dan pakai di: halaman login, sidebar/header, `document.title`
  (mis. "Dashboard — {app_name}"), dan favicon (buat dari logo). Simpan di `localStorage` (dibungkus try/catch)
  agar tampil instan saat reload; perbarui di belakang layar.
- Halaman **Pengaturan → Tampilan Aplikasi** (menu hanya untuk Super-Admin): form multipart `PUT /api/v1/branding`
  dengan `app_name` (wajib, maks 100), `logo` (opsional, JPEG/PNG **maks 1 MB**, cek sebelum upload, tampilkan
  pratinjau; sarankan persegi dan PNG transparan), dan tombol **Pakai logo bawaan** (`remove_logo=true`).
  Tanpa file `logo`, logo lama tetap dipakai.
- Setelah berhasil, langsung perbarui nama/logo di seluruh aplikasi (state global dan cache).
- Error 422: `logo size must not exceed 1 MB`, `file must be a JPEG or PNG image`; 403 untuk selain Super-Admin.

### 10.10 License — `/api/v1/license` (Super-Admin)

Backend wajib memiliki license aktif. Status license ada di field `license` pada response login, refresh, dan
`GET /auth/me`: `status` (`missing`/`active`/`grace`/`expired`/`invalid`), `expires_at`, `grace_until`, `days_left`.

- **Saat aplikasi dibuka (sebelum login):** panggil `GET /api/v1/license/status` (tanpa token, cukup signature):
  `status`, `locked`, `message`, `expires_at`, `grace_until`, `days_left`.
  - `locked = false` → alur normal (halaman login), dengan banner jika `grace` / `days_left` ≤ 30.
  - `locked = true` → tampilkan **layar "License belum aktif"** (status: belum dipasang / berakhir / tidak valid)
    dengan tombol **"Login sebagai Super-Admin"**. Login role lain akan ditolak server (403), jadi tampilkan pesan
    jelas. Setelah Super-Admin login, langsung arahkan ke halaman License (hanya halaman itu yang bisa dipakai).
  - Panggil ulang endpoint ini setelah license dipasang, saat tab kembali aktif, dan setiap 5 menit selama sesi;
    jika berubah menjadi `locked`, arahkan ke layar terkunci.
- **Banner global** (semua role web):
  - `active` dengan `days_left` terisi (≤ 30 hari): kuning — "License berakhir dalam {days_left} hari."
  - `grace`: merah — "License sudah berakhir. Sistem akan terkunci dalam {days_left} hari. Hubungi vendor."
- **Terkunci** (`missing`/`expired`/`invalid`):
  - Login selain Super-Admin → 403 `license is not active, contact your administrator` → tampilkan
    "Sistem belum memiliki license aktif. Hubungi administrator."
  - Request lain yang dijawab 403 dengan pesan itu (termasuk saat sedang bekerja) → akhiri sesi (non Super-Admin)
    dengan pesan yang sama. Untuk Super-Admin, arahkan ke halaman License.
  - Super-Admin setelah login saat terkunci: hanya tampilkan halaman **License** (layout minimal, tanpa menu lain).
- **Halaman Pengaturan → License** (menu hanya Super-Admin): `GET /api/v1/license` →
  - status (badge), `reason` jika tidak aktif, **Install ID** (tombol salin; "kirim ke vendor untuk mendapatkan license"),
  - data license: `license_id`, `customer`, `issued_at`, `expires_at`, `grace_until`,
  - pemakaian: "Unit aktif {used}/{max}", "App Client aktif {used}/{max}" (progress bar; merah jika `over_limit`,
    dengan pesan "Melebihi license: unit/App Client terbaru tidak berfungsi"),
  - form **Pasang license**: textarea kode (`SPL1....`) → `PUT /api/v1/license` `{"code": "..."}`. Setelah berhasil,
    muat ulang sesi (`GET /auth/me`) agar banner dan status terbaru.
  - Error 422: `license code is malformed` / `license code signature is invalid` → "Kode license tidak valid";
    `this license was issued for another installation` → "License ini untuk instalasi lain (Install ID berbeda)";
    `this license has already expired` → "License ini sudah berakhir".
- **Batas di halaman lain**:
  - Unit (tambah/aktifkan): 403 `the license unit limit has been reached (N)` → "Batas unit license tercapai (N)".
  - App Client (tambah/aktifkan): 403 `the license app client limit has been reached (N)`.
  - Tampilkan sisa kuota di halaman Unit dan App Client ("3 dari 5 unit aktif") dari `GET /api/v1/license`
    (Super-Admin).
  - Login user unit: 403 `your unit exceeds the license limit, contact the head office` → tampilkan pesan itu
    ("Unit Anda melebihi batas license, hubungi pusat").
  - App Client web yang melebihi batas: semua request 403 `this app client exceeds the license limit` → tampilkan
    halaman "Aplikasi ini melebihi batas license. Hubungi administrator." (bukan masalah login).

## 11. Profil

Data diri (`/auth/me`), foto (`/auth/me/face-photo`), ganti password (validasi 8–72, huruf + angka,
konfirmasi sama, berbeda dari password lama; 422 `old password is incorrect`), dan tombol
**Keluar dari semua perangkat** (`POST /auth/logout-all`, lalu logout lokal).

## 12. Kualitas & keamanan

- TypeScript strict; tipe untuk setiap response API; tidak ada `any` tanpa alasan.
- Setiap halaman punya state loading (skeleton), kosong, dan error dengan tombol ulangi.
- Form: validasi zod yang cocok dengan aturan server; error server dipetakan ke field.
- Tabel: paginasi dari server, filter disimpan di query string URL agar bisa dibagikan dan bertahan saat reload.
- Aksesibilitas dasar: label form, fokus terlihat, bisa dioperasikan dengan keyboard.
- Layout desktop-first, tetap bisa dipakai di tablet.
- Content Security Policy di hosting (script hanya dari origin sendiri; `connect-src` ke API; `img-src 'self' blob: data:`
  dan domain tile peta).
- Jangan log token, App Key, password, atau foto.

## 13. Kriteria selesai

- [ ] Unit test signature lulus untuk ketiga test vector (4.2), termasuk test signing multipart.
- [ ] Login, reload halaman → tetap login; token kedaluwarsa → refresh otomatis tanpa keluar.
- [ ] Dua tab terbuka, token kedaluwarsa bersamaan → hanya satu refresh, kedua tab tetap login.
- [ ] Jam komputer dimajukan 10 menit → request tetap berhasil (offset jam bekerja).
- [ ] Menu dan tombol sesuai role pada tabel bagian 6 (uji dengan keempat role manajemen dari seed).
- [ ] Pemilih unit untuk user pusat memfilter semua halaman; user unit tidak melihat data unit lain.
- [ ] Unit dinonaktifkan → user unit tersebut langsung keluar dengan pesan unit nonaktif.
- [ ] Setting unit: ubah radius unit, lalu "Ikuti nilai pusat" mengembalikan ke nilai global.
- [ ] Login `tim@securepatrol.local` di web ditolak dengan pesan "Akun Anda hanya dapat digunakan melalui aplikasi mobile."
- [ ] Tambah **dan ubah** user dengan foto 5 MB berhasil di Chrome, Firefox, dan Safari; foto > 5 MB atau bukan
      gambar ditolak sebelum upload.
- [ ] Foto wajah dan foto scan tampil lewat `<SecureImage>`.
- [ ] Titik patroli dibuat dengan memilih lokasi di peta; kode NFC ganda menampilkan error pada field.
- [ ] Shift tumpang tindih ditolak dengan pesan dari server; timeline menampilkan shift aktif.
- [ ] Help desk: drag & drop urutan tersimpan dan tampil sama setelah reload; Markdown berisi `<script>` tidak dieksekusi.
- [ ] App Client: App Key hanya muncul sekali; App Client web yang sedang dipakai tidak bisa dihapus/dinonaktifkan.
- [ ] Riwayat scan: filter lengkap, detail dengan peta dan foto; Export Excel dengan filter shift, titik, petugas,
      dan rentang tanggal shift menghasilkan file yang terbuka di Excel dengan waktu dalam WIB.
- [ ] Audit log tampil untuk Super-Admin dan Manager Keamanan dengan filter aksi, resource, pengguna, dan tanggal.
- [ ] Laporan rentang tanggal menampilkan tabel, grafik, dan ekspor CSV.

## 14. Urutan pengerjaan

1. Setup project, konfigurasi env, `apiFetch` (signing, offset jam, envelope, error) + unit test test vector.
2. Login, sesi, refresh lintas tab, layout, menu berdasarkan role, halaman 403/404.
3. Dashboard (per unit dan ringkasan pusat), pemilih unit.
4. Unit, Titik Patroli (dengan peta), dan Pengaturan Shift.
5. Pengguna dan Role.
6. Monitoring, Titik per Shift/Periode, Riwayat Scan (detail, ekspor), Laporan.
7. Help Desk (editor, reorder) dan Profil.
8. App Client.
9. Uji semua kriteria di bagian 13.

## 15. Laporan akhir

Laporkan: stack dan library, struktur folder, cara menjalankan (dev/build) dan menguji, hasil setiap kriteria di
bagian 13, keputusan yang kamu ambil sendiri, serta **permintaan ke backend** (endpoint/field yang dibutuhkan,
alasannya, dan contoh request/response).

## Lampiran: data uji di environment development

Dengan `SEED_DUMMY_DATA=true`, backend membuat satu user per role (password dari `SEED_DEFAULT_PASSWORD`
atau dari log saat seeding):

| Email | Role |
|---|---|
| `superadmin@securepatrol.local` | Super-Admin (pusat) |
| `manager@securepatrol.local` | Manager Keamanan (pusat) |
| `kepala@securepatrol.local` | Kepala Keamanan, Unit Utama |
| `admin@securepatrol.local` | Admin Keamanan, Unit Utama |
| `tim@securepatrol.local` | Tim Keamanan, Unit Utama (**tidak bisa login di web** — untuk menguji penolakan) |

Juga tersedia unit **Unit Utama** (`UNIT-UTAMA`) dengan 3 shift default, 3 titik patroli contoh (`DUMMY-NFC-0001` s.d. `0003`), dan 10 artikel help desk.
Data scan hanya bisa dibuat dari aplikasi mobile (atau lewat App Client `android`/`ios` di halaman `/docs`).
