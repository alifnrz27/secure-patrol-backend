# Instruksi untuk Claude: Update Aplikasi Mobile — Fitur Multi Unit

Kamu melanjutkan aplikasi mobile Secure Patrol yang sudah dibangun dari `MOBILE_CLAUDE_PROMPT.md`. Backend
sekarang mendukung **multi unit**. Perubahan di sisi mobile **kecil**: server otomatis membatasi semua data ke
unit user, jadi aplikasi **tidak perlu mengirim `unit_id`** di request mana pun. Pelajari dulu kode yang ada
(klien API, penanganan error, antrian scan, cache) dan ikuti pola yang sudah dipakai.

Sumber kebenaran kontrak API: `/docs/openapi.yaml` di backend (environment development).

## 1. Konsep

- Setiap petugas (Tim, Kepala, Admin Keamanan) terikat ke **satu unit** (lokasi/cabang). Titik patroli, shift,
  group patroli, riwayat scan, dan setting semuanya milik unit.
- Petugas **hanya bisa scan titik milik unitnya**. Kode NFC tetap unik di semua unit.
- User pusat (Super-Admin, Manager Keamanan) tidak punya unit dan **tidak bisa scan**.
- Pusat bisa **menonaktifkan unit**; seluruh user unit itu langsung tidak bisa memakai aplikasi.

## 2. Data unit dari login

`POST /auth/login`, `POST /auth/refresh`, dan `GET /auth/me` sekarang berisi:

```json
"user": {
  "id": 5, "name": "Tim Keamanan",
  "role": { "id": 5, "code": "security_team", "name": "Tim Keamanan" },
  "unit_id": 1,
  "unit": { "id": 1, "code": "UNIT-UTAMA", "name": "Unit Utama",
            "latitude": -6.2250138, "longitude": 106.8008324, "is_active": true }
}
```

- Simpan `user.unit` (tambahkan kolom/field di penyimpanan sesi lokal; buat migrasi DB lokal jika perlu).
- Tampilkan **nama unit** di header Dashboard dan di halaman Profil.
- `settings` dan `config` di respons **sudah sesuai unit** (setiap unit bisa punya radius scan, batas offline,
  akurasi wajah, lockout, dan masa token sendiri). Tidak perlu logika tambahan — tetap pakai nilai terbaru dari
  login/refresh/`/auth/me`/`/app-config`, dan **jangan hardcode** nilai default.
- Jika `user.unit` bernilai `null` (akun pusat login di mobile): tampilkan pesan "Akun pusat tidak dapat melakukan
  patroli" di Dashboard dan nonaktifkan tombol Scan. Fitur lain (help desk, ganti password, logout) tetap bisa.

## 3. Unit nonaktif — error baru (penting)

**403 dengan pesan `your unit is inactive, contact the head office`** bisa muncul saat login, saat refresh token,
**dan pada request apa pun** (termasuk saat sinkronisasi antrian scan), karena unit bisa dinonaktifkan kapan saja.
Pada request dengan token, pesan ada di `data`; pada login/refresh ada di `meta.message` — periksa keduanya.

Tangani secara global di klien API:

1. Akhiri sesi (hapus token) dan arahkan ke layar login dengan pesan
   **"Unit Anda sedang dinonaktifkan. Hubungi pusat."**
2. **Jangan hapus antrian scan** dan jangan tandai scan sebagai gagal permanen. Hentikan sinkronisasi; scan akan
   dikirim lagi setelah unit aktif kembali dan user login ulang (ikuti aturan yang sama dengan bagian 4.5 prompt
   awal: antrian milik user tersebut).
3. Di layar login, 403 dengan pesan ini → tampilkan pesan yang sama (bukan "email atau password salah").

Bedakan dari 403 lain (role/platform), yang tetap ditangani seperti sebelumnya.

## 4. Scan — error baru

`POST /api/v1/patrol-scans` bisa menjawab **403** berikut. Keduanya **permanen**: jangan retry, tandai scan di
antrian sebagai gagal dengan pesan ini, dan tampilkan di daftar scan pending/gagal:

| `meta.message` | Tampilkan |
|---|---|
| `this patrol point belongs to another unit` | Titik patroli ini milik unit lain. |
| `only unit users can scan patrol points` | Akun pusat tidak dapat melakukan patroli. |

Ini berbeda dengan 403 unit nonaktif (bagian 3) yang **tidak** permanen.

Cegah sebelum scan tersimpan: saat mencari titik dari NFC (bagian 7.2 prompt awal), cache titik dan group
hanya berisi titik unit user, dan `GET /patrol-points/by-nfc` menjawab **404** untuk titik unit lain. Jadi tag
unit lain akan tampil sebagai "Tag NFC ini tidak terdaftar di unit Anda." — ubah teks pesan 404 tersebut.

## 5. Perubahan respons lain (tidak wajib dipakai)

- Group (`/patrol-groups/current`, `/patrol-groups/{id}`) berisi `unit: { id, code, name }`.
- `group` di scan dan item berisi `unit_id` dan `unit_name`.
- Titik patroli dan shift berisi `unit_id`.
- Endpoint baru `GET /api/v1/units` (user unit hanya mendapat unitnya) — tidak perlu dipakai; data unit sudah
  ada di `user.unit`.

Pastikan parser JSON **mengabaikan field yang tidak dikenal**, sehingga field baru tidak membuat parsing gagal.

## 6. Ganti user / ganti unit

- Jika user yang login berbeda dari sebelumnya, atau **`user.unit.id` berbeda** dari yang tersimpan (user
  dipindah unit oleh pusat), hapus cache data GET (group, titik, shift, riwayat) lalu sinkronisasi ulang.
  Antrian scan diperlakukan sama seperti aturan ganti user di prompt awal.
- Pemindahan unit oleh pusat membuat semua sesi user tersebut berakhir (401) — alur refresh/login yang ada sudah
  menanganinya.

## 7. Kriteria selesai

- [ ] Login `tim@securepatrol.local`: Dashboard menampilkan "Unit Utama"; profil menampilkan unit.
- [ ] Setting radius unit diubah dari web (Kepala Keamanan) → setelah refresh/login ulang, validasi lokasi di HP
      memakai radius baru.
- [ ] Unit dinonaktifkan dari web saat aplikasi terbuka → request berikutnya membawa user ke login dengan pesan
      unit nonaktif; scan pending tetap ada di antrian dan terkirim setelah unit aktif dan login ulang.
- [ ] Login saat unit nonaktif → pesan unit nonaktif (bukan salah password).
- [ ] Scan tag milik unit lain → "Tag NFC ini tidak terdaftar di unit Anda." (online) atau, jika terlanjur
      tersimpan, gagal permanen dengan pesan "Titik patroli ini milik unit lain."
- [ ] Login dengan akun pusat di mobile → tombol Scan nonaktif dengan pesan akun pusat.
- [ ] Respons dengan field baru tidak menyebabkan error parsing.

## 8. Laporan akhir

Laporkan file yang diubah, migrasi penyimpanan lokal (jika ada), cara menguji, hasil setiap kriteria di bagian
7, keputusan yang kamu ambil sendiri, dan permintaan ke backend jika ada.

## Lampiran: data uji (development)

Dengan `SEED_DUMMY_DATA=true`: `tim@`, `kepala@`, `admin@securepatrol.local` ada di **Unit Utama**
(`UNIT-UTAMA`); `manager@` dan `superadmin@securepatrol.local` adalah akun pusat. Untuk menguji tag unit lain,
buat unit kedua dan titik patrolinya dari web (Super-Admin membuat unit dan Kepala Keamanan-nya, lalu Kepala
tersebut membuat titik).
