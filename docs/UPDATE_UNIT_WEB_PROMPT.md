# Instruksi untuk Claude: Update Web Admin — Fitur Multi Unit

Kamu melanjutkan web admin Secure Patrol (React) yang sudah dibangun dari `WEB_CLAUDE_PROMPT.md`. Backend
sekarang mendukung **multi unit**. Terapkan perubahan di bawah ini tanpa merusak fitur yang sudah ada. Pelajari
dulu kode yang ada (apiFetch, layout, menu per role, form, tabel) dan ikuti pola yang sudah dipakai.

Sumber kebenaran kontrak API: `/docs/openapi.yaml` di backend (environment development). Jika ada yang tidak
cocok dengan instruksi ini, ikuti OpenAPI dan laporkan perbedaannya.

## 1. Konsep

| Jenis user | Role | Unit | Data yang terlihat |
|---|---|---|---|
| Pusat | `super_admin`, `security_manager` | `user.unit = null` | Semua unit, bisa difilter `unit_id` |
| Unit | `security_head`, `security_admin` (+ `security_team`, hanya mobile) | `user.unit` terisi | Hanya unit sendiri |

- **Super-Admin** (admin pusat): kelola master unit, user (termasuk user pusat), role, app client, help desk,
  dan setting **global**. **Tidak** membuat/mengubah titik patroli dan shift (hanya melihat).
- **Manager Keamanan** (pusat): **hanya memantau** — semua halaman mode baca.
- **Kepala Keamanan & Admin Keamanan**: hak akses sama (role tetap dibedakan di tampilan). Kelola titik patroli,
  shift, pengguna (role unit), dan setting unitnya sendiri.
- Untuk user unit, server otomatis membatasi data ke unitnya; `unit_id` yang dikirim diabaikan. Data unit lain
  dijawab **404**.

## 2. Perubahan respons autentikasi

`POST /auth/login`, `POST /auth/refresh`, dan `GET /auth/me` sekarang berisi:

```json
"user": {
  "id": 3, "name": "Kepala Keamanan", "role": { "code": "security_head", ... },
  "unit_id": 1,
  "unit": { "id": 1, "code": "UNIT-UTAMA", "name": "Unit Utama",
            "latitude": -6.2250138, "longitude": 106.8008324, "is_active": true }
}
```

- Simpan `user.unit` di state sesi. `null` = user pusat.
- `settings` dan `config` sudah sesuai unit user (radius, lockout, TTL token).
- Error baru: **403 `your unit is inactive, contact the head office`** — bisa muncul saat login, refresh, **atau
  request apa pun** (unit dinonaktifkan saat user sedang bekerja). Tangani secara global di `apiFetch`: akhiri
  sesi dan arahkan ke login dengan pesan **"Unit Anda sedang dinonaktifkan, hubungi pusat."** Jangan
  perlakukan seperti 403 biasa.

## 3. Pemilih unit (user pusat)

- Tambahkan **pemilih unit global** di header untuk user pusat: opsi "Semua unit" + daftar dari
  `GET /api/v1/units?limit=100&is_active=` (tampilkan unit nonaktif dengan label "Nonaktif").
- Simpan pilihan di state global (dan `localStorage` sebagai kenyamanan, dibungkus try/catch).
- Kirim sebagai `unit_id` (kosongkan untuk "Semua unit") ke endpoint:
  `GET /users`, `/patrol-points`, `/patrol-shifts`, `/patrol-groups`, `/patrol-groups/current`,
  `/patrol-list-items`, `/patrol-scans`, `/patrol-scans/export`, `/settings`, `/audit-logs`.
- Saat "Semua unit", tampilkan kolom **Unit** di tabel (titik patroli, shift, pengguna, group, riwayat scan).
  Nama unit: dari `unit_name`/`unit` di respons atau peta `unit_id → nama` dari daftar unit.
- User unit: **sembunyikan** pemilih; tampilkan nama & kode unit di header.

## 4. Menu & hak akses (ganti tabel lama)

| Fitur | super_admin | security_manager | security_head / security_admin |
|---|---|---|---|
| Dashboard, Monitoring, Titik per Shift, Riwayat Scan, Laporan, Export | semua unit | semua unit | unit sendiri |
| **Unit**: lihat | ✓ | ✓ | (menu disembunyikan) |
| **Unit**: tambah/ubah/nonaktifkan/hapus | ✓ | | |
| Titik Patroli: lihat | ✓ | ✓ | ✓ |
| Titik Patroli: tambah/ubah/hapus | | | ✓ |
| Pengaturan Shift: lihat | ✓ | ✓ | ✓ |
| Pengaturan Shift: kelola | | | ✓ |
| Pengguna: lihat | semua | semua | unit sendiri |
| Pengguna: tambah/ubah/hapus/reset password | ✓ | | ✓ (role unit saja) |
| Role: lihat | ✓ | ✓ | ✓ |
| Role: kelola | ✓ | | |
| App Client | ✓ | | |
| Audit Log | ✓ | ✓ | |
| Pengaturan Sistem: lihat | global / per unit | global / per unit | unit sendiri |
| Pengaturan Sistem: ubah | nilai global | | nilai unit sendiri |
| Help Desk: baca draft | ✓ | ✓ | |
| Help Desk: tambah/ubah/hapus/urutkan | ✓ | | |

Perubahan dari versi sebelumnya yang perlu diperhatikan:
- Manager Keamanan **kehilangan** semua aksi tulis (pengguna, titik, shift, help desk).
- Super-Admin **kehilangan** tombol tambah/ubah/hapus titik patroli dan shift.
- Kepala Keamanan **mendapat** akses Pengguna dan Role (lihat).
- Kepala/Admin Keamanan **kehilangan** akses tulis help desk dan tidak lagi melihat draft.

## 5. Halaman baru: Unit — `/api/v1/units`

- Daftar: `GET ?page=&limit=&search=&is_active=`. Kolom: kode, nama, status (badge), `users_count`,
  `patrol_points_count`, diperbarui.
- Form (Super-Admin), JSON:

  | Field | Aturan |
  |---|---|
  | `code` | wajib, maks 20; server menyimpan huruf besar; unik |
  | `name` | wajib, maks 150 |
  | `latitude`, `longitude` | wajib; pilih di peta (komponen yang sama dengan Titik Patroli) |
  | `is_active` | default `true` saat create, **wajib** saat update |

- Info di form tambah: "Unit baru otomatis mendapat 3 shift default (08:00–16:00, 16:00–24:00, 00:00–08:00)."
- Menonaktifkan (`is_active: false`): dialog konfirmasi **"Semua pengguna unit ini akan langsung keluar dan
  tidak bisa login sampai unit diaktifkan kembali."**
- Hapus: hanya unit kosong. 409 `unit still has users or patrol points, move or delete them first` → tampilkan
  pesan dan sarankan menonaktifkan unit. 409 `unit code is already used by another unit` → error field `code`.

## 6. Dashboard

- User unit: seperti sebelumnya (`GET /patrol-groups/current`), tambah nama unit di kartu ringkasan.
- User pusat: `GET /patrol-groups/current` **wajib** `?unit_id=` (tanpa itu 422 `unit_id is required`).
  - "Semua unit": tampilkan **ringkasan per unit** — untuk setiap unit aktif panggil
    `current?unit_id={id}` (paralel, batasi ±5 sekaligus) dan tampilkan kartu: nama unit, shift, progres,
    temuan tidak normal. 422 `there is no active patrol shift...` → "Tidak ada shift aktif". Klik kartu → pilih
    unit tersebut di pemilih unit.
  - Satu unit dipilih: dashboard detail seperti biasa.
- Group sekarang berisi `unit: { id, code, name }`.

## 7. Monitoring, Titik per Shift, Riwayat Scan, Laporan, Export

- Semua daftar menerima `unit_id` (dari pemilih unit). Tampilkan kolom Unit saat "Semua unit".
- `group` pada scan dan item berisi `unit_id` dan `unit_name`.
- Filter petugas (`scanned_by`) sekarang tersedia untuk **semua** role web, termasuk Kepala Keamanan
  (`GET /users` otomatis hanya berisi petugas unitnya). Hapus pengecualian lama untuk Kepala Keamanan.
- Filter shift & titik: ambil opsi sesuai unit terpilih (`/patrol-shifts?unit_id=`, `/patrol-points?unit_id=`).
  Saat "Semua unit", tampilkan nama unit di opsi (mis. "Shift 1 — Unit Utama").
- **Export Excel**: dialog filter mendapat field **Unit** (hanya user pusat; isian awal dari pemilih unit).
  Kolom file sekarang: Waktu scan, Diterima server, Dikirim offline, Tanggal shift, **Unit**, Shift, Titik,
  Lokasi, Kondisi, Catatan, Petugas, Email petugas. Sheet "Filter" juga berisi baris Unit.
- Laporan: kelompokkan per unit saat "Semua unit".

## 8. Titik Patroli & Pengaturan Shift

- Tombol tambah/ubah/hapus hanya untuk Kepala/Admin Keamanan. Data dibuat otomatis di unit mereka (tidak ada
  field unit di form).
- Super-Admin/Manager: mode baca. Jika server menjawab 403 `patrol points are managed by each unit` /
  `shifts are managed by each unit`, tampilkan pesan itu.
- Posisi awal peta untuk titik baru: `user.unit.latitude/longitude`.
- Kode NFC tetap unik **di semua unit**: 409 bisa berarti kode dipakai unit lain — pesan field:
  "Kode NFC sudah dipakai (bisa di unit lain)."
- Shift: overlap hanya dicek dalam satu unit. Timeline 24 jam ditampilkan per unit (pusat: pilih unit dulu;
  saat "Semua unit" tampilkan satu timeline per unit).
- `PatrolPoint` dan `PatrolShift` sekarang punya field `unit_id`.

## 9. Pengguna — `/api/v1/users`

- Daftar: tambahan filter `unit_id` (pemilih unit) dan `head_office=true` (checkbox "Hanya user pusat", khusus
  user pusat). Kolom baru **Unit** (`user.unit.name`, "Pusat" jika `null`).
- Form tambah/ubah (multipart) mendapat field **`unit_id`**:
  - **Super-Admin**: tampilkan select Unit hanya jika role yang dipilih adalah role unit (bukan `super_admin` /
    `security_manager`); wajib. 422 `unit_id is required for this role` → error field unit;
    422 `unit not found` → error field unit. Untuk role pusat, sembunyikan dan jangan kirim `unit_id`.
  - **Kepala/Admin Keamanan**: jangan tampilkan select unit (user selalu dibuat di unitnya). Sembunyikan role
    `super_admin` dan `security_manager` dari pilihan role.
  - Ubah: `unit_id` opsional; kosong = tetap di unit sekarang. Jika Super-Admin memindahkan unit, tampilkan
    peringatan "Pengguna akan keluar dari semua perangkat."
- 403 `you are not allowed to manage users with this role` sekarang berlaku untuk semua user pusat (bukan hanya
  Super-Admin) jika pelakunya bukan Super-Admin.
- User tidak bisa memindahkan unit akunnya sendiri (422, pesan sama dengan menonaktifkan diri sendiri).

## 10. Pengaturan Sistem — `/api/v1/settings`

Setiap item sekarang punya field tambahan: `level` (`global`/`unit`), `global_value`, `is_inherited`.

- **Super-Admin**: halaman mengedit **nilai global** ("Berlaku untuk semua unit yang tidak mengatur nilai
  sendiri"). `null` = kembali ke default. Dengan pemilih unit terisi, Super-Admin melihat setting unit itu
  (`GET ?unit_id=`) dalam **mode baca** — Super-Admin tidak mengubah nilai unit.
- **Kepala/Admin Keamanan**: halaman mengedit **nilai unit**. Per baris tampilkan:
  - badge **"Mengikuti pusat"** jika `is_inherited = true`;
  - "Nilai pusat: {global_value}" sebagai pembanding;
  - tombol **"Ikuti nilai pusat"** → `PUT {"values": {"<key>": null}}`.
- **Manager Keamanan**: mode baca (global atau per unit via pemilih).
- 404 pada `GET ?unit_id=` → unit tidak ada.

## 11. Help Desk & Audit Log

- Help Desk: tombol kelola dan filter draft hanya untuk Super-Admin (draft juga terlihat oleh Manager).
- Audit Log: filter baru `unit_id` (pemilih unit) dan kolom Unit dari `unit_id` (`null` = "Pusat"/CLI).

## 12. Kriteria selesai

- [ ] Login sebagai `kepala@securepatrol.local`: header menampilkan "Unit Utama", tidak ada pemilih unit, tidak ada
      menu Unit, App Client, Audit Log; bisa kelola titik, shift, pengguna.
- [ ] Login sebagai `superadmin@securepatrol.local`: ada pemilih unit dan menu Unit; tidak ada tombol tambah
      titik/shift; buat unit baru → 3 shift otomatis terlihat saat unit dipilih.
- [ ] Login sebagai `manager@securepatrol.local`: semua halaman mode baca.
- [ ] Super-Admin membuat Kepala Keamanan untuk unit baru (select unit muncul hanya untuk role unit).
- [ ] Nonaktifkan unit saat Kepala unit itu sedang login di tab lain → tab tersebut keluar dengan pesan unit nonaktif.
- [ ] Hapus unit berisi data → pesan 409 dan saran menonaktifkan.
- [ ] Dashboard pusat "Semua unit" menampilkan kartu per unit.
- [ ] Export oleh Kepala Keamanan hanya berisi unitnya; export pusat dengan filter unit bekerja; kolom Unit ada.
- [ ] Setting unit: ubah radius unit → badge "Mengikuti pusat" hilang; "Ikuti nilai pusat" mengembalikannya.

## 13. Laporan akhir

Laporkan file yang diubah, cara menguji, hasil setiap kriteria di bagian 12, keputusan yang kamu ambil sendiri,
dan permintaan ke backend jika ada.
