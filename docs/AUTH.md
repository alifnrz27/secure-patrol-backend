# Autentikasi Secure Patrol API

> Dokumentasi interaktif (Swagger UI) tersedia di `/docs` **hanya jika `APP_ENV=development`**.
> Isi App ID dan App Key lewat tombol *Authorize*; halaman itu menandatangani setiap request secara otomatis.

Setiap request ke `/api/v1/*` melewati dua lapis keamanan:

1. **App signature** — membuktikan request berasal dari aplikasi resmi (Android, iOS, web, server) yang terdaftar. Wajib untuk semua endpoint, termasuk login.
2. **Bearer token** — membuktikan siapa user yang login. Wajib untuk semua endpoint kecuali `/auth/login` dan `/auth/refresh`.

`GET /health` tidak memerlukan keduanya.

## 1. App ID & App Key

Setiap aplikasi mendapat satu pasang kredensial:

| | Format | Contoh |
|---|---|---|
| App ID | `sp_<platform>_<16 karakter acak>_<6 karakter checksum>` | `sp_and_k3m9qx2ht5vbn4rw_w8xq2a` |
| App Key | `spk_<43 karakter base64url>` | `spk_-e2NwI_EJysxpO6hWXyKRxB4CWHpSDxm18PngULs0yM` |

Kode platform: `and` (android), `ios`, `web`, `srv` (server).

- Checksum App ID dihitung dengan HMAC dari `APP_MASTER_SECRET`. App ID palsu atau salah ketik langsung ditolak tanpa query ke database.
- App Key **hanya ditampilkan sekali** saat dibuat atau di-rotate. Di database, key disimpan terenkripsi AES-256-GCM.

### Membuat App Client pertama

API hanya bisa dipanggil oleh app client yang sudah terdaftar, jadi app client pertama dibuat lewat terminal:

```bash
go run . create-app-client -name "Secure Patrol Android" -platform android
```

Selanjutnya Super-Admin bisa mengelola app client lewat API (`/app-clients`).

### Rotasi key

`POST /api/v1/app-clients/:id/rotate-key` dengan body `{"grace_period_hours": 72}`.

Selama masa grace period, key lama dan key baru sama-sama diterima, sehingga aplikasi versi lama tetap berjalan sampai user melakukan update. Dengan `0`, key lama langsung mati (pakai ini jika key bocor).

## 2. Menandatangani request

Setiap request wajib mengirim header berikut:

| Header | Isi |
|---|---|
| `X-App-Id` | App ID |
| `X-Timestamp` | Unix time dalam detik (UTC). Maksimal selisih ±5 menit dari jam server |
| `X-Nonce` | String acak 16–64 karakter, unik untuk setiap request (misalnya UUID tanpa `-`) |
| `X-Signature` | Hex lowercase dari HMAC-SHA256 (lihat di bawah) |

```
payload   = METHOD + "\n" + REQUEST_URI + "\n" + TIMESTAMP + "\n" + NONCE + "\n" + SHA256_HEX(BODY)
signature = HEX( HMAC_SHA256(key = APP_KEY, message = payload) )
```

- `METHOD`: huruf besar, misalnya `POST`.
- `REQUEST_URI`: path lengkap beserta query string, persis seperti yang dikirim. Contoh: `/api/v1/users?page=1&search=budi`.
- `BODY`: byte mentah body yang dikirim. Untuk request tanpa body (GET/DELETE) gunakan string kosong, sehingga hash-nya `e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855`.
- Untuk upload `multipart/form-data`, yang di-hash adalah **seluruh byte multipart** yang dikirim, termasuk boundary.

Nonce yang sudah dipakai akan ditolak (proteksi replay attack). Karena itu, jika request perlu di-retry, **buat nonce dan signature baru**.

### Contoh Android (Kotlin + OkHttp Interceptor)

```kotlin
class AppSignatureInterceptor(
    private val appId: String,
    private val appKey: String,
) : Interceptor {
    override fun intercept(chain: Interceptor.Chain): Response {
        val request = chain.request()

        val bodyBytes = Buffer().also { request.body?.writeTo(it) }.readByteArray()
        val timestamp = (System.currentTimeMillis() / 1000).toString()
        val nonce = UUID.randomUUID().toString().replace("-", "")
        val requestUri = request.url.encodedPath +
            (request.url.encodedQuery?.let { "?$it" } ?: "")

        val payload = listOf(
            request.method.uppercase(),
            requestUri,
            timestamp,
            nonce,
            sha256Hex(bodyBytes),
        ).joinToString("\n")

        val signed = request.newBuilder()
            .header("X-App-Id", appId)
            .header("X-Timestamp", timestamp)
            .header("X-Nonce", nonce)
            .header("X-Signature", hmacSha256Hex(appKey, payload))
            .build()

        return chain.proceed(signed)
    }

    private fun sha256Hex(data: ByteArray) =
        MessageDigest.getInstance("SHA-256").digest(data).joinToString("") { "%02x".format(it) }

    private fun hmacSha256Hex(key: String, message: String): String {
        val mac = Mac.getInstance("HmacSHA256")
        mac.init(SecretKeySpec(key.toByteArray(), "HmacSHA256"))
        return mac.doFinal(message.toByteArray()).joinToString("") { "%02x".format(it) }
    }
}
```

Simpan App Key di `local.properties` / BuildConfig (jangan di-commit) dan aktifkan obfuscation (R8).

### Contoh JavaScript / Node

```js
import crypto from 'node:crypto'

function signHeaders(method, requestUri, body, appId, appKey) {
  const timestamp = Math.floor(Date.now() / 1000).toString()
  const nonce = crypto.randomBytes(16).toString('hex')
  const bodyHash = crypto.createHash('sha256').update(body ?? '').digest('hex')
  const payload = [method.toUpperCase(), requestUri, timestamp, nonce, bodyHash].join('\n')

  return {
    'X-App-Id': appId,
    'X-Timestamp': timestamp,
    'X-Nonce': nonce,
    'X-Signature': crypto.createHmac('sha256', appKey).update(payload).digest('hex'),
  }
}
```

## 3. Login & token

| Endpoint | Keterangan |
|---|---|
| `POST /auth/login` | Body `{"email", "password"}`, menghasilkan `access_token` dan `refresh_token` |
| `POST /auth/refresh` | Body `{"refresh_token"}`, menghasilkan pasangan token baru |
| `GET /auth/me` | Profil user yang sedang login |
| `GET /auth/me/face-photo` | Foto wajah user yang sedang login |
| `PUT /auth/change-password` | Body `{"old_password", "password", "password_confirmation"}`. Semua perangkat lain otomatis logout |
| `POST /auth/logout` | Logout dari perangkat ini |
| `POST /auth/logout-all` | Logout dari semua perangkat |

- Response login dan refresh berisi `config` (radius lokasi, batas foto, batas offline, `server_time`), juga
  tersedia di `GET /app-config`. Instruksi lengkap untuk aplikasi mobile ada di `docs/MOBILE_CLAUDE_PROMPT.md`.
- Access token dikirim di header `Authorization: Bearer <access_token>`. Berlaku sesuai setting
  `access_token_ttl_minutes` (default 60 menit).
- Refresh token berlaku sesuai setting `refresh_token_ttl_days` (default 30 hari) dan diperpanjang setiap kali dipakai.
- Response login, refresh, dan `GET /auth/me` berisi `settings`: `patrol_location_radius_meters`,
  `patrol_max_offline_hours`, `face_mobile_accuracy`, `login_max_failed_attempts`, `login_lock_minutes`,
  `access_token_ttl_minutes`, dan `refresh_token_ttl_days`.
- **Refresh token hanya bisa dipakai sekali.** Setiap refresh menghasilkan refresh token baru; simpan yang baru. Jika refresh token lama dipakai lagi, server menganggap token dicuri dan seluruh session dimatikan, sehingga user harus login ulang.
- Token terikat ke App ID yang dipakai saat login. Token dari aplikasi iOS tidak bisa dipakai di aplikasi Android.
- Setelah 3 kali salah password berturut-turut, akun dikunci selama 5 menit (HTTP 429, dengan header `Retry-After`,
  `data.locked_until`, dan `data.retry_after_seconds`). Login berhasil mengembalikan hitungan ke 0.
- Menonaktifkan user, mengganti role-nya, atau me-reset password-nya langsung mematikan semua session user tersebut.

## 4. Hak akses role & platform

| Fitur | Platform | Super-Admin | Manager Keamanan | Admin Keamanan | Kepala Keamanan | Tim Keamanan |
|---|---|---|---|---|---|---|
| Login dari **android/ios** | android/ios | ✓ | ✓ | ✓ | ✓ | ✓ |
| Login dari **web** (dan server) | web | ✓ | ✓ | ✓ | ✓ | **dilarang** |
| Profil sendiri, ganti password | semua | ✓ | ✓ | ✓ | ✓ | ✓ (android/ios) |
| Lihat patrol point, cari berdasarkan NFC | semua | ✓ | ✓ | ✓ | ✓ | ✓ |
| Tambah/ubah/hapus patrol point | **web** | ✓ | ✓ | ✓ | ✓ | |
| Lihat shift, group, daftar patroli | semua | ✓ | ✓ | ✓ | ✓ | ✓ |
| Tambah/ubah/hapus shift | **web** | ✓ | ✓ | ✓ | ✓ | |
| Scan NFC patroli | **android/ios** | ✓ | ✓ | ✓ | ✓ | ✓ |
| History scan | semua | semua scan | semua scan | semua scan | semua scan | scan sendiri |
| Baca help desk (aturan, tata cara, FAQ) | semua | ✓ + draft | ✓ + draft | ✓ + draft | ✓ + draft | ✓ (terbit saja) |
| Tambah/ubah/hapus artikel help desk | **web** | ✓ | ✓ | ✓ | ✓ | |
| Export Excel riwayat scan | **web** | ✓ | ✓ | ✓ | ✓ | |
| Audit log (riwayat create/update/delete) | **web** | ✓ | ✓ | | | |
| Lihat pengaturan sistem | **web** | ✓ | ✓ | ✓ | ✓ | |
| Ubah pengaturan sistem | **web** | ✓ | | | | |
| Lihat role | **web** | ✓ | ✓ | ✓ | | |
| Tambah/ubah/hapus role | **web** | ✓ | | | | |
| Kelola user | **web** | ✓ | ✓ | ✓ | | |
| Kelola App ID / App Key | **web** | ✓ | | | | |

- Role yang tidak boleh memakai web (Tim Keamanan dan role kustom) ditolak dengan HTTP 403
  `your role is not allowed to sign in on this platform` saat login, refresh token, dan setiap request, sehingga
  token lama pun tidak bisa dipakai jika role user berubah.
- Platform ditentukan oleh App Client yang menandatangani request (`X-App-Id`), bukan oleh user. Fitur bertanda
  **web** menolak request dari App Client android/ios/server dengan HTTP 403
  `This feature is only available on the web platform`, walaupun user-nya Super-Admin.
- Hanya Super-Admin yang bisa membuat atau mengubah user dengan role Super-Admin.

### Foto wajah

- Upload foto saat membuat user wajib, JPEG/PNG, **maksimal 5 MB**. Ukuran lebih dari itu ditolak dengan 422.
- Server memeriksa foto wajah (saat membuat user dan saat mengganti foto) sebelum disimpan: **tepat satu wajah**,
  minimal 20% sisi pendek foto, dan menghadap kamera dengan kedua mata terlihat. Foto yang tidak memenuhi syarat
  ditolak 422 dan tidak disimpan, sehingga foto acuan untuk pencocokan wajah di aplikasi mobile selalu layak pakai.
- Body request di atas 20 MB ditolak server dengan 413 sebelum body dibaca, lalu koneksi ditutup. Beberapa
  HTTP client (termasuk OkHttp) melaporkannya sebagai error koneksi, jadi cek ukuran foto di aplikasi sebelum upload.
- Response `POST /auth/login` menyertakan foto user di `user.face_photo_base64` (base64 standar) dan
  `user.face_photo_mime_type`. Nilainya `null` jika user belum punya foto, dan selalu `null` pada `/auth/refresh`.
  Foto 5 MB menjadi sekitar 6,7 MB dalam base64, jadi sebaiknya aplikasi mengompres foto sebelum upload.

## 5. Patroli

**Shift & group.** Admin mengatur shift (default: 08:00–16:00, 16:00–24:00, 00:00–08:00). Jam akhir adalah
batas (cut-off): scan tepat di jam akhir atau sesudahnya masuk ke shift berikutnya. Server otomatis membuat
satu *group* per shift per tanggal, lalu menyalin semua titik patroli ke daftar patroli group tersebut.
Layar utama aplikasi cukup memanggil `GET /patrol-groups/current`.

**Scan (`POST /patrol-scans`, multipart).**

| Field | Keterangan |
|---|---|
| `client_scan_id` | Wajib. Buat UUID baru **saat NFC dipindai** dan simpan bersama data scan di HP |
| `nfc_code`, `condition` (`normal`/`abnormal`), `note` | `note` wajib jika `abnormal` |
| `latitude`, `longitude` | Wajib, posisi HP saat scan |
| `scanned_at` | Waktu scan di HP (RFC3339, contoh `2026-09-29T08:15:00+07:00`) |
| `face_verified`, `face_match_score` | Hasil pencocokan wajah di HP (bandingkan dengan foto dari response login) |
| `photos` | Maksimal 3 file, masing-masing JPEG/PNG maksimal 5 MB |

**Mode offline.** Simpan scan di HP, lalu kirim saat ada sinyal:
- Kirim `scanned_at` waktu scan sebenarnya. Server menentukan shift dari waktu ini, bukan dari waktu kirim.
  Scan yang lebih lama dari setting `patrol_max_offline_hours` (default 24 jam) ditolak.
- Kirim ulang dengan `client_scan_id` yang sama jika gagal atau timeout. Jika scan itu sudah tersimpan,
  server menjawab `200` "Patrol scan was already recorded"; hapus dari antrian lokal saat menerima `200` atau `201`.
- Setiap percobaan kirim harus membuat `X-Nonce`, `X-Timestamp`, dan `X-Signature` **baru**.
- Jika token sudah kedaluwarsa, panggil `/auth/refresh` dulu sebelum sinkronisasi.

**Scan ditolak (422)** jika jarak ke titik lebih dari setting `patrol_location_radius_meters` (default 100 m) pada titik yang
mewajibkan lokasi, atau `face_verified` bukan `true` pada titik yang mewajibkan wajah. Pesan error menyebutkan
jaraknya, misalnya `you are 240 m away from the patrol point, the maximum allowed distance is 100 m`, sehingga bisa
ditampilkan langsung ke petugas. Scan yang ditolak tidak perlu dikirim ulang.

## 6. Kode error app signature

Semua kegagalan app signature mengembalikan HTTP 401 dengan salah satu pesan berikut di field `data`:

| Pesan | Penyebab |
|---|---|
| `app credential headers are missing` | Salah satu dari 4 header tidak dikirim |
| `app credential is invalid` | App ID salah format, checksum tidak cocok, atau tidak terdaftar |
| `app client is disabled or expired` | App client dinonaktifkan atau sudah lewat `expires_at` |
| `request timestamp is invalid or outside the allowed window` | Jam perangkat selisih lebih dari 5 menit |
| `request nonce is invalid` | Panjang nonce bukan 16–64 karakter |
| `request nonce has already been used` | Request di-replay atau di-retry dengan nonce yang sama |
| `request signature is invalid` | App Key salah, atau payload yang ditandatangani berbeda dengan yang dikirim |

## 7. Pengaturan sistem

Aturan yang bisa berubah disimpan di database (tabel `system_settings`), bukan di `.env`. Saat server start,
setting yang belum ada dibuat dengan nilai default; nilai yang sudah diubah tidak ditimpa. Nilai tidak valid di
database diabaikan dan diganti default. Super-Admin mengubahnya lewat `PUT /api/v1/settings` (langsung berlaku,
tercatat di audit log; `null` = kembali ke default).

| Key | Default | Keterangan | Dikirim ke aplikasi |
|---|---|---|---|
| `patrol_location_radius_meters` | 100 | Jarak maksimal HP ke titik patroli saat scan (1–10.000 m) | ✓ |
| `patrol_max_offline_hours` | 24 | Batas umur scan offline yang masih diterima (1–720 jam) | ✓ |
| `face_mobile_accuracy` | 0.75 | Skor kecocokan wajah yang dibutuhkan HP (0–1) | ✓ |
| `login_max_failed_attempts` | 3 | Salah password sebelum akun dikunci (1–20) | ✓ |
| `login_lock_minutes` | 5 | Lama akun dikunci (1–1.440 menit) | ✓ |
| `access_token_ttl_minutes` | 60 | Masa berlaku access token (5–1.440 menit) | ✓ |
| `refresh_token_ttl_days` | 30 | Lama tetap login tanpa password (1–365 hari) | ✓ |
| `face_match_min_score` | 0 | Skor wajah minimal yang dicek server saat scan; 0 = nonaktif | |
| `face_photo_validation` | true | Cek foto wajah acuan saat user dibuat / foto diganti | |
| `face_min_size_ratio` | 0.2 | Ukuran wajah minimal di foto acuan (0,05–0,9) | |
| `face_max_tilt_degrees` | 20 | Kemiringan kepala maksimal di foto acuan (1–45°) | |
| `face_max_turn_ratio` | 0.12 | Batas kepala menoleh di foto acuan (0,01–0,5) | |
