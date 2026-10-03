# Instruksi untuk Claude: Web Admin License Manager Secure Patrol (FE + BE satu repo)

Kamu membangun aplikasi web **internal vendor** untuk mengelola pelanggan Secure Patrol dan **menerbitkan kode
license**. Aplikasi ini terpisah dari backend Secure Patrol dan **tidak pernah** dipasang di server pelanggan.
Frontend dan backend berada di **satu repository**.

Backend Secure Patrol (di server pelanggan) hanya **memverifikasi** kode license dengan public key. Aplikasi ini
satu-satunya yang **membuat** kode, dengan private key. Format kode harus **persis** seperti spesifikasi di bawah;
kode yang salah satu karakter saja akan ditolak backend.

## 0. Cara kerja

- Repository kosong: buat project dari awal sesuai stack di bagian 2. Jika sudah ada isinya, pelajari dulu dan
  ikuti strukturnya.
- Kerjakan bertahap sesuai bagian 11; jalankan test setelah setiap tahap.
- Jangan pernah menulis private key ke file yang di-commit, log, response API, atau kode frontend.
- Jika ada yang ambigu, pilih yang paling aman dan catat di laporan akhir.

## 1. Fitur

| Menu | Isi |
|---|---|
| Login | Email + password + **2FA TOTP** (wajib), hanya akun vendor |
| Dashboard | Ringkasan: jumlah pelanggan, license aktif, **akan berakhir ≤ 30 hari**, sudah berakhir, terbit bulan ini |
| Pelanggan | CRUD pelanggan (nama perusahaan, kontak, email, telepon, catatan) |
| Instalasi | Per pelanggan: daftar server/instalasi dengan **Install ID** (mis. "Server Jakarta", `SP-VQKH-ZZRU-5ED5-HQBX`) |
| License | Daftar semua license + filter; **terbitkan**, **perpanjang**, **upgrade**; detail; salin/unduh kode |
| Cek kode | Tempel kode `SPL1....` → verifikasi tanda tangan dan tampilkan isinya (untuk support) |
| Audit log | Siapa menerbitkan/mengubah apa dan kapan (hanya baca) |
| Akun | Kelola akun vendor (hanya Owner), ganti password, atur 2FA |

## 2. Stack

- **Next.js 15 (App Router) + TypeScript**, satu repo untuk UI dan API (Route Handlers / Server Actions).
- **PostgreSQL** + **Prisma** (migrasi tersimpan di repo).
- UI: **Tailwind CSS + shadcn/ui**, tabel dengan paginasi/filter/sort di server.
- Auth: session cookie (httpOnly, secure, sameSite=lax), password **bcrypt/argon2**, 2FA **TOTP** (`otplib`) dengan QR.
- Validasi: **zod** (dipakai bersama oleh form dan API).
- Test: **vitest** (unit, termasuk test vector bagian 4.5) + **Playwright** untuk alur utama.
- Docker: `Dockerfile` + `docker-compose.yml` (app + postgres) untuk dijalankan di server vendor.
- Node.js ≥ 20, hanya modul `crypto` bawaan untuk tanda tangan (tanpa library kripto pihak ketiga).

## 3. Konfigurasi (`.env`, contoh di `.env.example` tanpa nilai rahasia)

| Variabel | Isi |
|---|---|
| `DATABASE_URL` | Koneksi PostgreSQL |
| `SESSION_SECRET` | ≥ 32 karakter acak |
| `LICENSE_PRIVATE_KEY` | **Private key vendor**: 64 karakter hex (seed Ed25519 32 byte). Nilainya diisi pemilik dari file `~/.secure-patrol/license_private.key`; **jangan** membuat key baru |
| `LICENSE_PUBLIC_KEY` | `a37b9a34ae55b5b6f1db443071152ee747eab11f493c6d5735f07b1a80c6bd2d` (boleh juga hardcode sebagai default) |
| `APP_URL` | URL aplikasi |

Saat start, cek `LICENSE_PRIVATE_KEY` cocok dengan public key: turunkan public key dari seed dan bandingkan dengan
`LICENSE_PUBLIC_KEY`. Jika tidak cocok, **gagal start** dengan pesan jelas. Ini mencegah salah key, karena kode dari
key yang salah akan ditolak semua server pelanggan.

## 4. Logic license (WAJIB persis)

### 4.1 Format

```
SPL1.<payload>.<signature>
```

- `payload` = base64url **tanpa padding** dari JSON UTF-8.
- `signature` = base64url tanpa padding dari tanda tangan **Ed25519** atas teks ASCII `SPL1.<payload>`.
- base64url: `+` → `-`, `/` → `_`, hapus `=`.

### 4.2 Isi payload (urutan key seperti ini)

| Key | Tipe | Aturan |
|---|---|---|
| `lid` | string | Wajib, unik. Format `LIC-YYYYMMDD-XXXXXX` (6 hex huruf besar acak) |
| `customer` | string | Nama perusahaan pelanggan (maks 150) |
| `install_id` | string | Wajib. Regex `^SP-[A-Z2-7]{4}-[A-Z2-7]{4}-[A-Z2-7]{4}-[A-Z2-7]{4}$` (simpan huruf besar, trim) |
| `max_units` | integer | ≥ 1. Jumlah **unit aktif** maksimal |
| `max_app_clients` | integer | ≥ 1. Jumlah **App ID/App Key aktif** maksimal (web + android + ios + server) |
| `issued_at` | string | RFC3339 dengan offset, tanpa milidetik, zona Asia/Jakarta: `2026-10-01T09:00:00+07:00` |
| `expires_at` | string | RFC3339, **harus setelah** `issued_at`. Dari tanggal di form: `YYYY-MM-DDT00:00:00+07:00` |
| `grace_days` | integer | 0–365 (default 14) |

Perilaku di server pelanggan (untuk ditampilkan di UI sebagai keterangan):

- Sebelum `expires_at`: aktif. Peringatan muncul ≤ 30 hari sebelum berakhir.
- Sampai `expires_at + grace_days`: masa tenggang, sistem tetap berjalan dengan peringatan.
- Sesudahnya: **sistem pelanggan terkunci total**.
- License yang dipasang terakhir yang berlaku. Perpanjangan/upgrade = terbitkan license baru dengan `install_id` sama.
- License **tidak bisa dicabut dari jarak jauh** (server pelanggan offline). Status "dicabut" di aplikasi ini hanya
  catatan internal; tampilkan keterangan ini di UI.

### 4.3 Implementasi (letakkan di `src/server/license/` dan `import "server-only"`)

```ts
import "server-only";
import crypto from "node:crypto";

const PREFIX = "SPL1";
const b64url = (buf: Buffer) => buf.toString("base64").replace(/\+/g, "-").replace(/\//g, "_").replace(/=+$/, "");
const fromB64url = (s: string) => Buffer.from(s.replace(/-/g, "+").replace(/_/g, "/"), "base64");

export type LicensePayload = {
  lid: string; customer: string; install_id: string;
  max_units: number; max_app_clients: number;
  issued_at: string; expires_at: string; grace_days: number;
};

function privateKey(seedHex: string) {
  const seed = Buffer.from(seedHex.trim(), "hex");
  if (seed.length !== 32) throw new Error("LICENSE_PRIVATE_KEY must be 64 hex characters");
  const der = Buffer.concat([Buffer.from("302e020100300506032b657004220420", "hex"), seed]); // PKCS#8
  return crypto.createPrivateKey({ key: der, format: "der", type: "pkcs8" });
}

function publicKey(hex: string) {
  const der = Buffer.concat([Buffer.from("302a300506032b6570032100", "hex"), Buffer.from(hex, "hex")]); // SPKI
  return crypto.createPublicKey({ key: der, format: "der", type: "spki" });
}

// Urutan key menentukan isi payload; selalu bangun objek dengan urutan bagian 4.2.
export function encodeLicense(p: LicensePayload, seedHex: string): string {
  const ordered = {
    lid: p.lid, customer: p.customer, install_id: p.install_id,
    max_units: p.max_units, max_app_clients: p.max_app_clients,
    issued_at: p.issued_at, expires_at: p.expires_at, grace_days: p.grace_days,
  };
  const data = `${PREFIX}.${b64url(Buffer.from(JSON.stringify(ordered), "utf8"))}`;
  const signature = crypto.sign(null, Buffer.from(data, "ascii"), privateKey(seedHex));
  return `${data}.${b64url(signature)}`;
}

export function decodeLicense(code: string): LicensePayload {
  const parts = code.trim().split(".");
  if (parts.length !== 3 || parts[0] !== PREFIX) throw new Error("not a license code");
  return JSON.parse(fromB64url(parts[1]).toString("utf8"));
}

export function verifyLicense(code: string, publicKeyHex: string): boolean {
  const parts = code.trim().split(".");
  if (parts.length !== 3 || parts[0] !== PREFIX) return false;
  return crypto.verify(null, Buffer.from(`${parts[0]}.${parts[1]}`, "ascii"), publicKey(publicKeyHex), fromB64url(parts[2]));
}

export function publicKeyFromSeed(seedHex: string): string {
  const pub = crypto.createPublicKey(privateKey(seedHex)).export({ format: "der", type: "spki" });
  return Buffer.from(pub).subarray(-32).toString("hex");
}
```

### 4.4 Aturan menerbitkan

1. Validasi input dengan zod (aturan bagian 4.2). `expires_at` minimal besok; tampilkan peringatan jika > 3 tahun.
2. Buat `lid` unik, `issued_at` = sekarang (Asia/Jakarta, tanpa milidetik).
3. `encodeLicense` lalu langsung `verifyLicense` hasilnya (harus `true`) sebelum disimpan.
4. Simpan baris license (bagian 5) termasuk **kode lengkap**, dalam satu transaksi dengan audit log.
5. License sebelumnya untuk instalasi yang sama ditandai `superseded` (tetap ada sebagai riwayat).
6. Tanda tangan **hanya** di server (Server Action / Route Handler). Private key tidak boleh sampai ke browser.

**Perpanjang** = form terisi dari license terakhir instalasi itu, `expires_at` + 1 tahun (bisa diubah).
**Upgrade** = sama, tetapi biasanya mengubah `max_units` / `max_app_clients`. Keduanya menerbitkan license baru.

### 4.5 Test vector (WAJIB lulus)

Data:

```json
{"lid":"LIC-TEST-VECTOR","customer":"PT Test Vector","install_id":"SP-TEST-AAAA-BBBB-CCCC","max_units":3,"max_app_clients":4,"issued_at":"2026-10-01T09:00:00+07:00","expires_at":"2027-10-01T00:00:00+07:00","grace_days":14}
```

Kode yang benar (dibuat backend Secure Patrol):

```
SPL1.eyJsaWQiOiJMSUMtVEVTVC1WRUNUT1IiLCJjdXN0b21lciI6IlBUIFRlc3QgVmVjdG9yIiwiaW5zdGFsbF9pZCI6IlNQLVRFU1QtQUFBQS1CQkJCLUNDQ0MiLCJtYXhfdW5pdHMiOjMsIm1heF9hcHBfY2xpZW50cyI6NCwiaXNzdWVkX2F0IjoiMjAyNi0xMC0wMVQwOTowMDowMCswNzowMCIsImV4cGlyZXNfYXQiOiIyMDI3LTEwLTAxVDAwOjAwOjAwKzA3OjAwIiwiZ3JhY2VfZGF5cyI6MTR9.R-TRaDPAmG1L55x5ryyU5tPxT0ULLK6Zv1V6kGCXpW885j3LwVEivw-91KDgpNv_UUaMJOn7nZLdsTv2ideRBA
```

Test:

- `verifyLicense(kode, PUBLIC_KEY)` → `true`; `decodeLicense(kode)` sama dengan data di atas.
- Mengubah satu karakter payload atau signature → `false`.
- Jika `LICENSE_PRIVATE_KEY` tersedia di environment test: `encodeLicense(data, key)` harus menghasilkan kode
  **persis sama** (Ed25519 deterministik). Lewati test ini jika variabel tidak ada (mis. di CI tanpa secret).
- `publicKeyFromSeed(LICENSE_PRIVATE_KEY)` = public key di bagian 3 (jika variabel tersedia).
- Build produksi tidak memuat `LICENSE_PRIVATE_KEY` atau modul `src/server/license` di bundle client
  (cek `.next/static` dengan grep).

## 5. Data (Prisma)

- `User`: id, name, email (unik), passwordHash, role (`owner` | `staff`), totpSecret (terenkripsi), isActive,
  lastLoginAt, failedLogins, lockedUntil, timestamps.
- `Customer`: id, name, contactName, email, phone, address, notes, timestamps, deletedAt (soft delete).
- `Installation`: id, customerId, name, installId (unik, format bagian 4.2), notes, timestamps, deletedAt.
- `License`: id, lid (unik), customerId, installationId, customerName (salinan saat terbit), installId (salinan),
  maxUnits, maxAppClients, issuedAt, expiresAt, graceDays, code (text), status (`current` | `superseded` |
  `revoked`), note, issuedById, timestamps.
  Status tampilan dihitung dari tanggal: **Aktif**, **Berakhir ≤ 30 hari**, **Masa tenggang**, **Berakhir**.
- `AuditLog`: id, userId, action (`login`, `customer.create`, `license.issue`, ...), entity, entityId, detail
  (JSON tanpa rahasia), ip, userAgent, createdAt.

## 6. API (Route Handlers, JSON, semua butuh login kecuali login)

| Method | Path | Fungsi |
|---|---|---|
| POST | `/api/auth/login` | Email + password → minta TOTP |
| POST | `/api/auth/verify-totp` | Kode TOTP → session |
| POST | `/api/auth/logout` | |
| GET/POST | `/api/customers` | Daftar (search, paginasi) / tambah |
| GET/PUT/DELETE | `/api/customers/:id` | Detail (+ instalasi + license) / ubah / soft delete |
| GET/POST | `/api/customers/:id/installations` | Daftar / tambah instalasi |
| PUT/DELETE | `/api/installations/:id` | Ubah / soft delete (ditolak jika masih punya license `current` yang aktif) |
| GET | `/api/licenses` | Daftar: filter customer, installation, status tampilan, rentang `expires_at`, search `lid`/install ID |
| POST | `/api/licenses` | **Terbitkan** (body: installationId, maxUnits, maxAppClients, expiresAt, graceDays, note) |
| GET | `/api/licenses/:id` | Detail + kode lengkap + payload ter-decode |
| POST | `/api/licenses/:id/revoke` | Tandai dicabut (hanya catatan, dengan alasan) |
| POST | `/api/licenses/verify` | Body `{code}` → `{valid, payload, statusTampilan}` |
| GET | `/api/dashboard` | Angka ringkasan + daftar license yang berakhir ≤ 30 hari |
| GET | `/api/audit-logs` | Daftar audit log (filter user, action, tanggal) |
| CRUD | `/api/users` | Hanya `owner` |

Error dalam format `{ "error": { "message", "fields"? } }` dengan status 400/401/403/404/409/422.

## 7. Halaman

- **Daftar license:** kolom LID, pelanggan, instalasi, Install ID, unit, App Client, terbit, berakhir, sisa hari,
  status (badge warna), aksi (detail, perpanjang, upgrade). Ekspor CSV.
- **Form terbitkan:** pilih pelanggan → pilih instalasi (atau tambah baru dengan input Install ID). Field unit,
  App Client, tanggal berakhir (preset +1 tahun / +2 tahun), masa tenggang (default 14), catatan.
  - Pratinjau ringkas sebelum submit.
  - Setelah terbit, tampilkan **kode lengkap** dengan tombol **Salin**, **Unduh .txt** (`<lid>.txt`), dan
    instruksi pemasangan untuk pelanggan (bagian 8).
- **Detail license:** semua field, kode lengkap (salin/unduh), payload ter-decode, riwayat license instalasi yang
  sama, audit log terkait.
- **Cek kode:** textarea → hasil valid/tidak, payload, status tampilan, dan apakah Install ID terdaftar di aplikasi
  ini (pelanggan mana).
- **Dashboard:** kartu angka + tabel "Akan berakhir ≤ 30 hari" dengan tombol Perpanjang.

## 8. Instruksi untuk pelanggan (ditampilkan setelah terbit)

```
1. Di server: docker compose exec backend /app/secure-patrol-backend license-info  → kirim Install ID ke vendor
2. Pasang: docker compose exec backend /app/secure-patrol-backend license-install -code "<KODE>"
   atau login Super-Admin di web admin Secure Patrol → menu License → tempel kode.
```

## 9. Keamanan

- Private key hanya dibaca di server dari `LICENSE_PRIVATE_KEY`; modul tanda tangan memakai `import "server-only"`.
- Tidak ada endpoint yang mengembalikan private key atau menerima key dari client.
- Login: rate limit + kunci akun 15 menit setelah 5 kali gagal; 2FA wajib untuk semua akun.
- Semua aksi tulis dan penerbitan license dicatat di audit log.
- CSRF: Server Actions/same-site cookie; header keamanan (CSP, X-Frame-Options DENY).
- Aplikasi dijalankan di server vendor, sebaiknya hanya dari VPN/IP kantor.
- Seed: perintah `npm run create-owner -- --email ... ` untuk akun owner pertama (password ditampilkan sekali).

## 10. Kriteria selesai

- [ ] Test vector bagian 4.5 lulus (verify; encode identik jika key tersedia).
- [ ] Kode yang diterbitkan dipasang di backend Secure Patrol development (`license-install`) → status `active`
      dengan unit/App Client sesuai.
- [ ] Start gagal jika `LICENSE_PRIVATE_KEY` tidak cocok dengan public key.
- [ ] Private key tidak ada di bundle client, log, atau response.
- [ ] Perpanjang/upgrade membuat license baru dan license lama `superseded`.
- [ ] Filter daftar license dan dashboard "berakhir ≤ 30 hari" benar.
- [ ] Login dengan 2FA; akun terkunci setelah 5 kali gagal; audit log tercatat.
- [ ] `docker compose up -d --build` menjalankan aplikasi + database dan migrasi otomatis.

## 11. Urutan pengerjaan

1. Setup project, Docker, Prisma, `.env.example`.
2. Modul `src/server/license` + test vector (bagian 4) — **lulus dulu sebelum lanjut**.
3. Auth (login, TOTP, session, owner pertama).
4. Pelanggan & instalasi.
5. Terbitkan / perpanjang / upgrade / detail / cek kode.
6. Dashboard, audit log, akun.
7. Test Playwright alur: login → tambah pelanggan → instalasi → terbitkan → salin kode.

## 12. Laporan akhir

Laporkan: struktur folder, cara menjalankan (dev/Docker), cara membuat akun owner, hasil setiap kriteria bagian
10, dan keputusan yang kamu ambil sendiri.
