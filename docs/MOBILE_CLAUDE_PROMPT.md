# Instruksi untuk Claude: Aplikasi Mobile Secure Patrol

Kamu akan membangun aplikasi mobile **Secure Patrol** untuk petugas keamanan di repository ini. Backend sudah
jadi dan sudah dites; tugasmu adalah membuat aplikasi yang memakainya dengan benar, termasuk saat tidak ada sinyal.

Baca seluruh dokumen ini sebelum menulis kode. Semua path, field, dan pesan error di bawah adalah perilaku
backend yang sebenarnya — jangan menebak field lain. Jika butuh detail tambahan, spesifikasi lengkap ada di
`/docs/openapi.yaml` pada backend (bisa dibuka di `http://<server>/docs` saat backend berjalan dengan
`APP_ENV=development`).

## 0. Cara kerja

1. **Pelajari repository dulu.** Tentukan stack yang dipakai (Flutter, Kotlin/Android, Swift/iOS, React Native,
   dll.), arsitektur, state management, library HTTP dan database lokal yang sudah ada. Ikuti pola yang sudah
   ada; jangan memperkenalkan framework baru tanpa alasan kuat.
2. Jika repository masih kosong atau stack-nya belum jelas, **berhenti dan tanyakan** stack yang diinginkan
   sebelum membuat project.
3. Kerjakan bertahap sesuai urutan di bagian 13, dan pastikan setiap tahap berjalan sebelum lanjut.
4. Teks UI dalam **Bahasa Indonesia**. Nama kode, variabel, dan komentar dalam bahasa Inggris.

## 1. Ruang lingkup

**Dibangun di aplikasi mobile:**

| Fitur | Keterangan |
|---|---|
| Login | Email + password; simpan sesi, foto wajah, dan konfigurasi |
| Dashboard | Titik NFC pada shift yang sedang berjalan beserta status terakhirnya |
| Scan patroli | Baca NFC → validasi lokasi (jika wajib) → validasi wajah (jika wajib) → kondisi, catatan, foto → kirim |
| Mode offline | Semua data GET di-cache; semua scan disimpan lokal dulu lalu dikirim saat online |
| Ganti password | |
| Help desk | Aturan, tata cara, FAQ (baca saja) |
| Logout | |

**Tidak dibangun** (hanya ada di web admin, dan backend menolaknya dari mobile dengan 403): manajemen unit,
user, role, app client, patrol point, shift, dan artikel help desk.

**Unit.** Setiap petugas (Tim, Kepala, Admin Keamanan) terikat ke **satu unit**. Server otomatis membatasi semua
data (group, titik patroli, shift, riwayat scan, setting) ke unit user tersebut; aplikasi tidak perlu mengirim
`unit_id`. Petugas hanya bisa scan titik milik unitnya. User pusat (Super-Admin, Manager Keamanan) tidak punya
unit dan **tidak bisa scan**; aplikasi mobile ditujukan untuk user unit.

## 2. Konfigurasi build

Buat konfigurasi per environment (dev/staging/production) yang **tidak di-commit**:

| Kunci | Contoh | Keterangan |
|---|---|---|
| `BASE_URL` | `https://api.example.com` | Tanpa trailing slash; semua path API diawali `/api/v1` |
| `APP_ID` | `sp_and_k3m9qx2ht5vbn4rw_w8xq2a` | Berbeda untuk Android dan iOS |
| `APP_KEY` | `spk_...` | Rahasia; dipakai untuk menandatangani request |

App ID/Key dibuat oleh tim backend (`go run . create-app-client -name "Secure Patrol Android" -platform android`,
dan `-platform ios` untuk iOS). **Platform harus `android` atau `ios`** — App Client `web` ditolak saat scan.

Simpan App Key lewat mekanisme build secret (misalnya `local.properties`/BuildConfig, `.xcconfig`,
`--dart-define`), aktifkan obfuscation (R8/ProGuard), dan jangan pernah menulisnya ke log.

## 3. Kontrak API umum

### 3.1 Format response

Semua response JSON berbentuk:

```json
{ "meta": { "message": "Login success", "code": 200, "status": "success" }, "data": { } }
```

Pada error, `meta.status` = `"Error"`, `meta.message` berisi pesan (bahasa Inggris), dan `data` bisa berupa
`null`, string detail, atau array pesan validasi. Response 413 (body terlalu besar) juga berformat ini.

### 3.2 Tanda tangan request (wajib untuk SEMUA request ke `/api/v1`)

Setiap request, termasuk login, wajib membawa 4 header:

| Header | Isi |
|---|---|
| `X-App-Id` | App ID |
| `X-Timestamp` | Unix time dalam **detik**, dari jam yang sudah dikoreksi (lihat 3.3) |
| `X-Nonce` | String acak 16–64 karakter, **baru untuk setiap percobaan kirim** (misalnya 16 byte acak → hex) |
| `X-Signature` | Hex lowercase dari HMAC-SHA256 di bawah |

```
payload   = METHOD + "\n" + REQUEST_URI + "\n" + X_TIMESTAMP + "\n" + X_NONCE + "\n" + hex(SHA256(BODY))
signature = hex( HMAC_SHA256(key = UTF8(APP_KEY), message = UTF8(payload)) )
```

- `METHOD` huruf besar.
- `REQUEST_URI` = path lengkap + query string **persis seperti yang dikirim**, contoh
  `/api/v1/help-desk-articles?category=faq&page=1`. Encode query terlebih dahulu, baru tanda tangani string yang sama.
- `BODY` = byte mentah yang benar-benar dikirim. Tanpa body → SHA256 dari string kosong
  (`e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855`).
- **Multipart:** susun seluruh body multipart (termasuk boundary) menjadi byte array lebih dulu, hitung
  signature dari byte tersebut, lalu kirim byte yang sama persis dengan `Content-Type: multipart/form-data; boundary=...`
  yang sama. Jangan biarkan library HTTP membangun ulang multipart setelah ditandatangani.
- Implementasikan sebagai satu interceptor/middleware HTTP, dan buat nonce + timestamp + signature **baru setiap
  kali request dikirim ulang** (retry atau sinkronisasi offline). Nonce yang sama ditolak.

**Test vector** (dihitung oleh kode backend; implementasimu wajib menghasilkan nilai yang sama — buat unit test):

App Key uji: `spk_TEST-ONLY-4pZt7Qm2Xv9Lr3Ks8Wd1Nf6Hy0Bc5Ja`

| # | Method & URI | Timestamp | Nonce | Body | Signature |
|---|---|---|---|---|---|
| 1 | `POST /api/v1/auth/login` | `1790668800` | `0123456789abcdef0123456789abcdef` | `{"email":"tim@securepatrol.local","password":"Password123"}` | `f5681cea7af562190e884bc0b1aff7a303367cf687ede9ec9c70d6e2c64da48c` |
| 2 | `GET /api/v1/patrol-groups/current` | `1790668800` | `fedcba9876543210fedcba9876543210` | (kosong) | `e4ac3c5d488438c9e829dfd0091a6eb540d3bd4fc4682f85195859e15b7f1f12` |
| 3 | `GET /api/v1/help-desk-articles?category=faq&page=1` | `1790668801` | `a1b2c3d4e5f60718293a4b5c6d7e8f90` | (kosong) | `ccd21a96183515fc2ae3ad4aed4de79b217933250e232f99128a6892c767c2c3` |

SHA256 body #1 = `511f90493776b18136558905986a0abb6bb1e4bdc68e5bcd1374196004f0fc24`.

### 3.3 Koreksi jam HP

Server menolak request jika `X-Timestamp` selisih lebih dari **300 detik** dari jam server. Jam HP petugas sering
tidak akurat, jadi:

- Simpan `clockOffset = serverTime − deviceTime`. Ambil `serverTime` dari header HTTP **`Date`** (ada di setiap
  response, termasuk `GET /health` tanpa autentikasi dan response 401) atau dari `config.server_time`.
- Gunakan `deviceNow + clockOffset` untuk `X-Timestamp` **dan** untuk `scanned_at`.
- Saat aplikasi dibuka dan online, panggil `GET {BASE_URL}/health` (tanpa header apa pun) untuk memperbarui offset.
- Jika menerima 401 `meta.message = "Unauthorized app"` dengan `data = "request timestamp is invalid or outside the allowed window"`,
  perbarui offset dari header `Date` response tersebut lalu kirim ulang **sekali**.

### 3.4 Penanganan status HTTP

| Kondisi | Arti | Tindakan |
|---|---|---|
| 401, `meta.message = "Unauthorized app"` | Masalah App ID/Key, signature, jam, atau nonce | Jam → koreksi & ulang sekali (3.3). Nonce dipakai → ulang dengan nonce baru. Lainnya → konfigurasi salah; jangan logout, catat error |
| 401, `meta.message = "Unauthorized"`, `data = "Token is expired"` | Access token habis | Refresh (bagian 4.3) lalu ulang request |
| 401, `meta.message = "Unauthorized"`, `data` lain | Token/sesi tidak valid (logout di perangkat lain, akun dinonaktifkan, password direset) | Coba refresh sekali; jika gagal → sesi berakhir, arahkan ke login (tanpa menghapus antrian scan) |
| 403, `data`/`meta.message` = `your unit is inactive, contact the head office` | Unit user dinonaktifkan pusat | Sesi berakhir: arahkan ke login dengan pesan "Unit Anda sedang dinonaktifkan, hubungi pusat". **Jangan hapus antrian scan**; kirim lagi setelah unit aktif dan user login ulang |
| 403 lainnya | Role atau platform tidak diizinkan | Tampilkan pesan; jangan retry |
| 404 | Data tidak ditemukan | Tampilkan pesan; jangan retry |
| 413 | Body > 20 MB | Kompres foto (bagian 7.6); jangan retry dengan body yang sama |
| 422 | Ditolak aturan bisnis / validasi | Tampilkan pesan; **jangan retry** |
| 429 | Akun terkunci (3 kali salah password, 5 menit) | Tampilkan pesan dengan hitung mundur dari `data.retry_after_seconds` (juga header `Retry-After`) |
| 5xx, timeout, tidak ada jaringan | Sementara | Retry dengan backoff (bagian 5.3) |

Catatan: server bisa memutus koneksi saat body > 20 MB sebelum sempat mengirim 413, sehingga library HTTP
melaporkan error koneksi. Cegah dengan memvalidasi ukuran foto sebelum mengirim.

## 4. Autentikasi

### 4.1 Login — `POST /api/v1/auth/login`

Request: `{"email": "budi@securepatrol.local", "password": "Rahasia123"}`. Trim spasi pada email.

Response 200 (dipersingkat):

```json
{
  "data": {
    "token_type": "Bearer",
    "access_token": "eyJhbGciOiJIUzI1NiIsInR5...",
    "expires_in": 3599,
    "expires_at": "2026-09-29T16:29:58+07:00",
    "refresh_token": "a62b65153ab847349663...",
    "refresh_expires_at": "2026-10-29T15:29:58+07:00",
    "user": {
      "id": 6, "name": "Budi Santoso", "email": "budi@securepatrol.local",
      "role": { "id": 5, "code": "security_team", "name": "Tim Keamanan" },
      "unit_id": 1,
      "unit": { "id": 1, "code": "UNIT-UTAMA", "name": "Unit Utama",
                "latitude": -6.2250138, "longitude": 106.8008324, "is_active": true },
      "is_active": true, "is_locked": false,
      "face_photo_url": "/api/v1/users/6/face-photo",
      "face_photo_updated_at": "2026-09-29T15:29:58+07:00",
      "face_photo_base64": "iVBORw0KGgoAAAAN...",
      "face_photo_mime_type": "image/png"
    },
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
  }
}
```

Setelah login berhasil:

1. Simpan `access_token`, `refresh_token`, dan waktu kedaluwarsanya di **secure storage** (Android Keystore /
   EncryptedSharedPreferences, iOS Keychain).
2. Decode `face_photo_base64` (base64 standar) → simpan sebagai file di direktori privat aplikasi (terenkripsi
   jika platform mendukung), beserta `face_photo_updated_at`. Foto ini adalah **acuan pencocokan wajah** (bagian 7.4).
   Server sudah menjamin foto acuan berisi tepat satu wajah yang cukup besar dan menghadap kamera (orientasi EXIF
   sudah diperhitungkan saat divalidasi, tetapi file yang dikirim adalah file asli — terapkan orientasi EXIF saat
   men-decode foto acuan sebelum menghitung embedding).
   Jika `null`, user belum punya foto: titik yang mewajibkan validasi wajah tidak bisa di-scan — tampilkan
   pesan "Foto wajah belum terdaftar, hubungi Admin Keamanan".
3. Simpan `config` dan `settings` (dipakai offline) dan perbarui `clockOffset` dari `config.server_time`.
   `settings` berisi `patrol_location_radius_meters`, `patrol_max_offline_hours`, `face_mobile_accuracy`,
   `login_max_failed_attempts`, `login_lock_minutes`, `access_token_ttl_minutes`, dan `refresh_token_ttl_days`;
   nilainya diatur admin dan bisa berubah, jadi selalu pakai nilai terbaru dari login, refresh, atau `GET /auth/me`.
   Nilainya **sudah sesuai unit user** (unit bisa punya radius dan aturan sendiri), jadi pakai apa adanya.
   Simpan juga `user.unit` (kode dan nama unit) untuk ditampilkan di header Dashboard dan Profil.
4. Jika user yang login **berbeda** dari user sebelumnya di perangkat ini, hapus semua cache milik user
   sebelumnya — tetapi lihat 4.5 soal antrian scan.
5. Lakukan sinkronisasi awal (bagian 5.2), lalu buka Dashboard.

Error login: 401 `email or password is incorrect`, 403 `account is inactive`,
403 `your unit is inactive, contact the head office` ("Unit Anda sedang dinonaktifkan, hubungi pusat"), 429 `account is temporarily locked...`,
422 validasi. Login **tidak** bisa dilakukan offline.

Semua role boleh login dari App Client `android` dan `ios`. Jika muncul 403 `your role is not allowed to sign in on this platform`,
aplikasi memakai App ID/Key platform yang salah (misalnya milik web) — periksa konfigurasi build.

### 4.2 Membuka aplikasi dengan sesi tersimpan

1. Jika ada sesi tersimpan → langsung buka Dashboard dari cache (jangan menunggu jaringan).
2. Di background (jika online): `GET /health` untuk offset jam, `GET /app-config` untuk konfigurasi terbaru,
   `GET /auth/me` untuk profil. Jika `face_photo_updated_at` berbeda dari yang tersimpan, unduh ulang foto wajah
   lewat `GET /auth/me/face-photo` (response berupa file gambar, bukan JSON).
3. Jalankan sinkronisasi (bagian 5).

### 4.3 Refresh token — `POST /api/v1/auth/refresh`

Request: `{"refresh_token": "..."}` → response berformat sama dengan login (dengan `config`; `face_photo_base64` selalu `null`).

- **Refresh token hanya bisa dipakai sekali.** Setiap refresh menghasilkan refresh token baru; simpan yang baru
  secara atomik sebelum memakai access token baru.
- Jika refresh token lama dipakai lagi, server menganggapnya dicuri dan **mematikan seluruh sesi**. Karena itu
  refresh harus **single-flight**: hanya satu refresh berjalan pada satu waktu; request lain yang mendapat 401
  menunggu hasil refresh tersebut lalu mengulang dengan token baru. Terapkan dengan mutex/lock global
  (termasuk untuk worker sinkronisasi di background).
- Refresh juga boleh dilakukan proaktif ~1 menit sebelum `expires_at`.
- Jika refresh gagal dengan 401 → sesi berakhir: arahkan ke layar login dengan pesan "Sesi berakhir, silakan login kembali".

### 4.4 Logout — `POST /api/v1/auth/logout`

- Jika antrian scan (bagian 5.3) **tidak kosong**, jangan izinkan logout: tampilkan "Masih ada X scan yang belum
  terkirim. Hubungkan ke internet untuk mengirimnya terlebih dahulu."
- Jika kosong: panggil endpoint (abaikan kegagalan jaringan), lalu hapus token, foto wajah, cache, dan konfigurasi.

### 4.5 Sesi berakhir saat masih ada antrian

Jika sesi berakhir (refresh gagal) sementara antrian scan belum kosong, **jangan hapus antrian**. Setelah user
login lagi **dengan akun yang sama**, lanjutkan sinkronisasi. Jika yang login akun lain, tampilkan peringatan
bahwa scan milik akun sebelumnya tidak bisa dikirim sampai akun tersebut login kembali, dan simpan antriannya
terpisah per user id.

## 5. Penyimpanan lokal & sinkronisasi (offline-first)

Aplikasi harus tetap bisa dipakai tanpa sinyal. Prinsip: **tampilkan data lokal dulu, perbarui dari jaringan
di belakang layar, dan semua perubahan disimpan lokal sebelum dikirim.**

### 5.1 Database lokal

Gunakan database lokal terenkripsi yang sesuai stack (misalnya Room + SQLCipher, Drift + sqlcipher, GRDB +
SQLCipher, WatermelonDB/SQLite + SQLCipher). Minimal tabel:

- `cache_entries(key, owner_user_id, json, fetched_at)` — hasil GET.
- `pending_scans` — antrian scan (bagian 5.3).
- File foto scan disimpan di direktori privat aplikasi dan direferensikan oleh `pending_scans`.

Semua data terikat pada `owner_user_id`.

### 5.2 Cache data GET

| Data | Endpoint | Kapan diperbarui |
|---|---|---|
| Konfigurasi | `GET /api/v1/app-config` (juga ada di login/refresh) | Buka aplikasi, setiap sinkronisasi |
| Profil | `GET /api/v1/auth/me` | Buka aplikasi |
| Foto wajah | `GET /api/v1/auth/me/face-photo` | Saat `face_photo_updated_at` berubah |
| Shift | `GET /api/v1/patrol-shifts` | Buka aplikasi, setiap sinkronisasi |
| Group & daftar patroli aktif | `GET /api/v1/patrol-groups/current` | Buka Dashboard, pull-to-refresh, setelah scan terkirim, saat shift berganti |
| Semua titik patroli | `GET /api/v1/patrol-points?page=N&limit=100` (ulangi sampai semua halaman) | Buka aplikasi, setiap sinkronisasi |
| Help desk | `GET /api/v1/help-desk-articles?limit=100` | Buka menu Help Desk, setiap sinkronisasi |

Aturan: tampilkan cache segera (stale-while-revalidate), tampilkan "Terakhir diperbarui: <waktu>" dan banner
"Mode offline" jika gagal memperbarui. Kegagalan jaringan saat GET tidak boleh menghapus cache.

### 5.3 Antrian scan (outbox)

Kolom minimal `pending_scans`:

| Kolom | Keterangan |
|---|---|
| `client_scan_id` | UUID v4 tanpa tanda hubung, dibuat **saat NFC dipindai** — kunci idempotensi |
| `owner_user_id` | |
| `patrol_point_id`, `nfc_code`, `point_name` | Untuk tampilan |
| `condition`, `note`, `latitude`, `longitude` | |
| `scanned_at` | RFC3339 dengan offset, dari jam yang sudah dikoreksi (contoh `2026-09-29T08:15:00+07:00`) |
| `face_verified`, `face_match_score` | |
| `photo_paths` | 0–3 path file lokal |
| `status` | `pending` / `sending` / `sent` / `rejected` / `expired` |
| `attempts`, `next_attempt_at`, `last_error` | |

Aturan sinkronisasi:

1. **Setiap scan selalu disimpan ke antrian lebih dulu**, baru dicoba kirim — bahkan saat online. Jangan
   pernah hanya menyimpan scan di memori.
2. Pemicu: segera setelah scan disimpan, saat koneksi kembali, saat aplikasi dibuka/kembali ke foreground,
   pull-to-refresh, dan background job berkala (Android WorkManager minimal 15 menit dengan constraint
   network; iOS BGTaskScheduler).
3. Kirim **satu per satu**, urut berdasarkan `scanned_at`. Hanya satu proses sinkronisasi pada satu waktu.
4. Setiap percobaan membangun request baru (nonce, timestamp, signature baru) tetapi dengan `client_scan_id` yang sama.
5. Hasil:
   - **201** atau **200** (`"Patrol scan was already recorded"`) → `sent`; hapus file foto lokal.
   - **422 / 404 / 403** → `rejected` dengan `meta.message`; jangan retry; tampilkan ke user di daftar
     "Scan ditolak server" dan sebagai notifikasi.
   - **401** → tangani sesuai 3.4, lalu coba lagi.
   - **5xx / jaringan / timeout** → tetap `pending`, backoff eksponensial (5 d, 15 d, 1 m, 5 m, maks 15 m).
6. Scan dengan `scanned_at` lebih tua dari `settings.patrol_max_offline_hours` akan ditolak server. Tampilkan peringatan
   ketika ada scan pending yang mendekati batas (misalnya sisa < 2 jam), dan tandai `expired` setelah lewat.
7. Tampilkan jumlah scan pending di header Dashboard ("3 scan menunggu dikirim").

### 5.4 Hal yang tidak di-antrikan

**Ganti password** dan **login** harus online. Tampilkan pesan "Butuh koneksi internet" jika offline.

## 6. Dashboard

Sumber data: `GET /api/v1/patrol-groups/current` (response berisi group + `items`):

```json
{
  "id": 1,
  "unit": { "id": 1, "code": "UNIT-UTAMA", "name": "Unit Utama" },
  "shift": { "id": 1, "name": "Shift 1" },
  "shift_date": "2026-09-29",
  "start_at": "2026-09-29T08:00:00+07:00",
  "end_at": "2026-09-29T16:00:00+07:00",
  "status": "ongoing",
  "progress": { "total_points": 3, "scanned_points": 1, "unscanned_points": 2, "total_scans": 1, "abnormal_scans": 1 },
  "items": [
    {
      "id": 2, "patrol_point_id": 2, "name": "Parking Area", "location": "Basement 1",
      "nfc_code": "DUMMY-NFC-0002", "latitude": -6.2252431, "longitude": 106.8011502,
      "is_location_match_required": true, "is_face_validation_required": false,
      "is_scanned": true, "scan_count": 1,
      "last_scanned_at": "2026-09-29T15:29:58+07:00",
      "last_scanned_by": { "id": 6, "name": "Budi Santoso", "email": "budi@securepatrol.local" },
      "last_condition": "abnormal"
    }
  ]
}
```

Tampilkan:

- Header: nama unit, nama shift, tanggal, jam mulai–selesai, sisa waktu shift, progres (`scanned_points / total_points`),
  jumlah temuan tidak normal, jumlah scan pending, status online/offline.
- Daftar titik: nama, lokasi, ikon kewajiban (📍 lokasi, 🙂 wajah), dan status terakhir:
  - **Belum di-scan** (`is_scanned = false`)
  - **Normal** / **Tidak Normal** dari `last_condition`, dengan `last_scanned_at` dan `last_scanned_by.name`
  - **Menunggu sinkron** jika ada scan pending lokal untuk titik tersebut (tampilkan kondisi dari scan lokal
    dan waktunya; ini menimpa status server sampai terkirim)
- Filter: Semua / Belum / Sudah / Tidak Normal. Pull-to-refresh.
- Setelah scan terkirim, perbarui Dashboard dari server.
- Jika server menjawab 422 `there is no active patrol shift at the scan time`: tampilkan "Tidak ada shift aktif saat ini".
- **Pergantian shift saat offline:** jika waktu sekarang ≥ `end_at` group yang di-cache, tampilkan banner
  "Data shift sudah berganti, hubungkan ke internet untuk memperbarui". Scan tetap boleh dilakukan (server
  menentukan shift dari `scanned_at`), dan tentukan nama shift sementara dari cache `patrol-shifts`
  (jam akhir adalah batas: scan tepat di jam akhir masuk shift berikutnya; `end_time` bisa `24:00`, dan shift
  yang jam akhirnya lebih kecil dari jam mulai melewati tengah malam).

## 7. Alur scan patroli

### 7.1 Membaca NFC

- Gunakan **UID tag**, diformat sebagai heksadesimal **huruf besar dipisah titik dua**, contoh `04:A2:1F:9C:3B:7E:80`.
  Server membandingkan kode tanpa membedakan huruf besar/kecil, tetapi pemisah dan urutan byte harus sama
  dengan yang didaftarkan admin di web.
- Android: foreground dispatch / reader mode saat layar Scan terbuka. iOS: Core NFC (`NFCTagReaderSession`,
  butuh entitlement dan `NFCReaderUsageDescription`).
- Tangani: NFC mati (arahkan ke pengaturan), perangkat tanpa NFC, tag tidak terbaca.

### 7.2 Mencari titik patroli

1. Cari `nfc_code` (case-insensitive) di cache `items` group aktif.
2. Jika tidak ada, cari di cache semua titik patroli (`/patrol-points`).
3. Jika tidak ada dan online: `GET /api/v1/patrol-points/by-nfc?code=<UID ter-encode>` (404 = tidak terdaftar
   **di unit user**; titik milik unit lain juga dijawab 404).
4. Jika tetap tidak ada: "Tag NFC ini tidak terdaftar sebagai titik patroli." Jangan buat scan.

Dari titik patroli, ambil `latitude`, `longitude`, `is_location_match_required`, `is_face_validation_required`.

### 7.3 Validasi lokasi

Selalu ambil lokasi (koordinat wajib dikirim), tetapi **hanya blokir** jika `is_location_match_required = true`.

- Minta posisi baru (bukan cache lebih dari 30 detik) dengan akurasi tinggi, timeout ~20 detik, dan tampilkan akurasinya.
- Tolak lokasi palsu: Android `Location.isMock()` / `isFromMockProvider()`, iOS 15+
  `CLLocation.sourceInformation?.isSimulatedBySoftware`.
- Hitung jarak dengan rumus Haversine yang sama dengan server:

  ```
  R = 6371000
  dLat = rad(lat2 - lat1); dLon = rad(lon2 - lon1)
  a = sin(dLat/2)^2 + cos(rad(lat1)) * cos(rad(lat2)) * sin(dLon/2)^2
  distance = R * 2 * atan2(sqrt(a), sqrt(1 - a))
  ```

- Jika `distance > settings.patrol_location_radius_meters`: tampilkan "Anda berjarak {jarak} m dari titik patroli.
  Maksimal {radius} m." dan tombol "Coba lagi". Scan tidak disimpan.
- Server memeriksa ulang aturan ini; pemeriksaan di HP agar petugas mendapat umpan balik langsung, terutama saat offline.

### 7.4 Validasi wajah

**Hanya jika `is_face_validation_required = true`.** Jika tidak wajib, lewati langkah ini dan kirim
`face_verified=false` tanpa `face_match_score`.

Backend tidak menerima foto selfie; pencocokan sepenuhnya di HP, dan server hanya memeriksa hasilnya.

1. Buka kamera depan, deteksi wajah (tepat satu wajah, cukup besar, menghadap depan).
2. Lakukan liveness sederhana (misalnya kedip atau menoleh) agar tidak bisa memakai foto.
3. Hitung *face embedding* dari frame kamera dan dari foto acuan login (bagian 4.1). Simpan embedding foto acuan
   agar tidak dihitung ulang setiap scan; hitung ulang jika `face_photo_updated_at` berubah.
4. `face_match_score` = kemiripan dalam rentang 0–1 (misalnya cosine similarity yang dipetakan ke 0–1),
   dibulatkan 4 desimal.
5. Ambang batas = `settings.face_mobile_accuracy` (default 0.75, diatur admin). Jika `config.face_match_min_score`
   tidak null dan lebih besar, gunakan nilai itu, karena server akan menolak skor di bawahnya.
6. `face_verified = score >= ambang`. Beri maksimal 3 percobaan. Jika gagal: "Wajah tidak cocok dengan foto
   terdaftar" dan scan tidak disimpan.
7. Frame kamera hanya di memori; jangan disimpan atau dikirim.

Rekomendasi library on-device (pilih sesuai stack): deteksi wajah Google ML Kit (Android/Flutter) atau Vision
(iOS); embedding dengan model TFLite seperti MobileFaceNet (`tflite_flutter`, TensorFlow Lite Android,
atau Core ML hasil konversi). Pastikan model berjalan offline.

### 7.5 Form kondisi

- Kondisi: **Normal** (`normal`) atau **Tidak Normal** (`abnormal`).
- Catatan: maksimal 1000 karakter; **wajib jika Tidak Normal**.
- Foto: 0 sampai `config.max_scan_photos` (3), dari kamera.

### 7.6 Foto

- Kompres sebelum disimpan: sisi terpanjang ≤ 1920 px, JPEG kualitas ~80. Setiap file wajib
  ≤ `config.max_photo_size_bytes` (5 MB); kompres ulang jika masih lebih besar.
- Hanya JPEG atau PNG (server memeriksa isi file).
- Simpan di direktori privat aplikasi; hapus setelah scan terkirim.

### 7.7 Menyimpan dan mengirim scan

1. Buat record antrian (bagian 5.3): `client_scan_id` baru, `scanned_at` = waktu NFC dipindai (jam terkoreksi,
   format RFC3339 dengan offset), koordinat, hasil wajah, kondisi, catatan, path foto.
2. Simpan dalam satu transaksi, lalu tampilkan "Scan tersimpan" dan kembali ke Dashboard (status titik
   "Menunggu sinkron").
3. Picu sinkronisasi.

Request: `POST /api/v1/patrol-scans`, `multipart/form-data`:

| Field | Wajib | Contoh |
|---|---|---|
| `client_scan_id` | ya | `59079acaed3a4eb3a3f89b10caa4bc3f` |
| `nfc_code` | ya | `04:A2:1F:9C` |
| `condition` | ya | `normal` / `abnormal` |
| `note` | jika `abnormal` | `Lampu parkir mati` |
| `latitude`, `longitude` | ya | `-6.2252431`, `106.8011502` |
| `scanned_at` | ya (selalu kirim) | `2026-09-29T15:29:58+07:00` |
| `face_verified` | jika wajib wajah | `true` / `false` |
| `face_match_score` | jika wajib wajah | `0.9312` |
| `photos` | tidak | 0–3 file, field `photos` diulang untuk setiap file |

Contoh response 201:

```json
{
  "meta": { "message": "Patrol scan success", "code": 201, "status": "success" },
  "data": {
    "id": 1,
    "client_scan_id": "59079acaed3a4eb3a3f89b10caa4bc3f",
    "group": { "id": 1, "unit_id": 1, "unit_name": "Unit Utama", "shift_id": 1, "shift_name": "Shift 1", "shift_date": "2026-09-29",
               "start_at": "2026-09-29T08:00:00+07:00", "end_at": "2026-09-29T16:00:00+07:00" },
    "patrol_point": { "patrol_list_item_id": 2, "patrol_point_id": 2, "name": "Parking Area",
                      "location": "Basement 1", "nfc_code": "DUMMY-NFC-0002" },
    "condition": "abnormal", "note": "Lampu parkir mati",
    "latitude": -6.2252431, "longitude": 106.8011502, "distance_meters": 0,
    "is_location_valid": true, "is_face_verified": false, "face_match_score": null,
    "scanned_at": "2026-09-29T15:29:58+07:00", "received_at": "2026-09-29T15:29:58+07:00",
    "scanned_by": { "id": 6, "name": "Budi Santoso", "email": "budi@securepatrol.local" },
    "photos": [ { "id": 1, "url": "/api/v1/patrol-scans/1/photos/1" } ]
  }
}
```

Pesan 422 yang mungkin muncul (tampilkan versi Bahasa Indonesia):

| `meta.message` | Tampilkan |
|---|---|
| `you are {X} m away from the patrol point, the maximum allowed distance is {Y} m` | Anda berjarak {X} m dari titik patroli (maks {Y} m) |
| `face validation is required for this patrol point and was not verified` | Validasi wajah wajib untuk titik ini |
| `face_match_score is required for this patrol point` / `face match score is below the minimum required` | Wajah tidak cocok dengan foto terdaftar |
| `note is required when the condition is abnormal` | Catatan wajib diisi untuk kondisi Tidak Normal |
| `a scan can have at most 3 photos` / `file size must not exceed 5 MB` / `file must be a JPEG or PNG image` | Foto tidak valid |
| `there is no active patrol shift at the scan time` | Tidak ada shift aktif pada waktu scan |
| `scan is too old to be submitted` | Scan sudah melewati batas waktu pengiriman |
| `scanned_at cannot be in the future` | Jam perangkat tidak sesuai — koreksi offset jam |
| lainnya | tampilkan `meta.message` apa adanya |

404 `nfc tag is not registered as a patrol point` → "Tag NFC tidak terdaftar".

403 (jangan retry, tandai scan di antrian sebagai **gagal permanen** dengan pesan ini):

| `meta.message` | Tampilkan |
|---|---|
| `this patrol point belongs to another unit` | Titik patroli ini milik unit lain |
| `only unit users can scan patrol points` | Akun pusat tidak bisa melakukan patroli |

Riwayat scan milik sendiri bisa ditampilkan dari `GET /api/v1/patrol-scans?page=1&limit=20` (Tim Keamanan
otomatis hanya melihat scan miliknya) — opsional, jika waktu memungkinkan.

## 8. Ganti password — `PUT /api/v1/auth/change-password`

Request: `{"old_password": "...", "password": "...", "password_confirmation": "..."}`

- Validasi di HP: 8–72 karakter, minimal satu huruf dan satu angka, konfirmasi sama, berbeda dari password lama.
- 200 → "Password berhasil diganti". Sesi di HP ini tetap aktif; semua perangkat lain otomatis logout.
- 422 `old password is incorrect` → "Password lama salah"; `new password must be different from the old password`;
  `password must be 8-72 characters and contain at least one letter and one digit`.
- Hanya online.

## 9. Help desk — `GET /api/v1/help-desk-articles`

- Parameter: `category` (`rule` | `guide` | `faq`), `search`, `page`, `limit` (maks 100).
- Urutan dari server sudah benar (aturan → tata cara → FAQ, lalu `sort_order`); jangan diurutkan ulang.
- Tampilkan tab **Aturan**, **Tata Cara**, **FAQ**, pencarian (pada data cache, agar bekerja offline), dan
  halaman detail. Field `content` berformat **Markdown** — render sebagai Markdown.
- Detail: `GET /api/v1/help-desk-articles/{id}`.
- Cache penuh untuk dibaca offline.

## 10. Keamanan

- Token, App Key, dan foto wajah tidak boleh muncul di log, crash report, atau analytics.
- Token di secure storage; database lokal dan foto terenkripsi / di direktori privat; kecualikan dari backup
  cloud (Android `allowBackup=false` atau aturan backup; iOS file protection + exclude from backup).
- Nonaktifkan screenshot di layar validasi wajah (Android `FLAG_SECURE`).
- Gunakan HTTPS untuk staging/production.

## 11. Kebutuhan tambahan untuk backend

Jika selama implementasi kamu membutuhkan sesuatu dari backend yang tidak ada di dokumen ini, **jangan
mengakalinya di aplikasi**. Catat di bagian "Permintaan ke backend" pada laporan akhirmu (endpoint/field, alasannya,
dan contoh request/response yang diharapkan).

## 12. Kriteria selesai

- [ ] Unit test signature lulus untuk ketiga test vector (bagian 3.2).
- [ ] Unit test Haversine: `(-6.2, 106.8)` ke `(-6.2009, 106.8)` ≈ 100 m.
- [ ] Login, buka ulang aplikasi tanpa jaringan → Dashboard dan Help Desk tampil dari cache.
- [ ] Jam HP dimajukan 10 menit → aplikasi tetap berhasil memanggil API (offset jam bekerja).
- [ ] Mode pesawat: scan 3 titik → tersimpan dengan status "Menunggu sinkron" → koneksi kembali → terkirim otomatis,
      Dashboard diperbarui, dan tidak ada data ganda di server.
- [ ] Koneksi terputus saat upload → dikirim ulang → server menjawab 200 "already recorded" dan antrian bersih.
- [ ] Titik wajib lokasi di-scan dari jarak > radius → diblokir di HP dengan pesan jarak.
- [ ] Titik wajib wajah: wajah orang lain → ditolak; wajah pemilik akun → lolos; titik tanpa kewajiban wajah
      tidak membuka kamera.
- [ ] Kondisi Tidak Normal tanpa catatan tidak bisa disimpan; foto > 5 MB dikompres otomatis.
- [ ] Dua request mendapat 401 "Token is expired" bersamaan → hanya satu refresh yang terjadi.
- [ ] Logout diblokir saat ada scan pending.
- [ ] Ganti password berhasil dan sesi tetap aktif.

## 13. Urutan pengerjaan

1. Konfigurasi build + HTTP client dengan signing interceptor + test vector + koreksi jam.
2. Login, penyimpanan sesi, refresh single-flight, logout.
3. Database lokal + cache GET + layar Dashboard.
4. Scan: NFC → pencarian titik → lokasi → form → antrian → sinkronisasi.
5. Validasi wajah.
6. Help Desk, Ganti Password, profil.
7. Uji semua kriteria di bagian 12.

## 14. Laporan akhir

Setelah selesai, laporkan: stack dan library yang dipakai, struktur folder baru, cara menjalankan dan menguji,
hasil setiap kriteria di bagian 12, keputusan yang kamu ambil sendiri (misalnya ambang wajah dan model yang dipakai),
serta **permintaan ke backend** jika ada.

## Lampiran: data untuk uji di environment development

Dengan `SEED_DUMMY_DATA=true`, backend membuat user berikut (password dari `SEED_DEFAULT_PASSWORD` atau dari log saat seeding):

- `tim@securepatrol.local` — Tim Keamanan (akun utama untuk uji mobile)
- `kepala@securepatrol.local`, `admin@securepatrol.local` — unit yang sama (**Unit Utama**, kode `UNIT-UTAMA`)
- `manager@securepatrol.local`, `superadmin@securepatrol.local` — user pusat, tidak punya unit dan tidak bisa scan

Titik patroli contoh memakai kode `DUMMY-NFC-0001` s.d. `0003`, yang tidak ada pada tag fisik. Untuk uji
dengan tag NFC sungguhan, minta admin mendaftarkan titik baru lewat web dengan `nfc_code` berisi UID tag dalam
format bagian 7.1.
