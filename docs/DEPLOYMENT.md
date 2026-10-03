# Deploy Secure Patrol Backend dengan Docker Compose

Panduan dari server sampai API berjalan di HTTPS, lalu cara menjalankan ulang, update, rollback, dan backup.
Backend dan database PostgreSQL-nya berjalan dari satu `docker-compose.yml`; frontend adalah container terpisah.

```
Internet ──HTTPS──▶ nginx (server) ──▶ secure-backend (127.0.0.1:PORT) ──▶ secure-postgres
                                        └──────── docker compose project ────────┘
```

| Nama | Isi |
|---|---|
| `secure-backend` | Container API ini |
| `secure-postgres` | Container PostgreSQL 16 khusus Secure Patrol (port tidak dibuka ke luar) |
| `secure-postgres-data` | Volume data database (**wajib di-backup**) |
| `secure-backend-storage` | Volume foto wajah dan foto scan (**wajib di-backup**) |
| `secure-backend-logs` | Volume log aplikasi |
| `secure-redis` | Opsional (`--profile redis`), hanya jika backend dijalankan lebih dari satu container |

Ada dua cara instalasi:

| Cara | Untuk | Kode sumber di server |
|---|---|---|
| **A. Image privat** (`deploy/customer/`) | **Server pelanggan** (on-premise) | Tidak ada; hanya image ter-obfuscate dari registry privat |
| **B. Build dari source** (`docker-compose.yml` di root) | Server milik vendor (staging/development) | Ada (`git clone`) |

**Jangan pernah memberi pelanggan akses ke repository.** Dengan kode sumber, pemeriksaan license bisa dihapus.

Setiap instalasi wajib memasang **license** (bagian 5b). Tanpa license, hanya Super-Admin yang bisa login.

## 1. Siapkan server

Contoh untuk Ubuntu 22.04/24.04 dengan domain API (misalnya `api.securepatrol.id`) yang sudah diarahkan ke IP server.

```bash
# Docker + Docker Compose (skrip resmi), git, nginx, certbot
curl -fsSL https://get.docker.com | sudo sh
sudo usermod -aG docker $USER        # lalu logout-login agar bisa memakai docker tanpa sudo
sudo apt-get update && sudo apt-get install -y git nginx certbot python3-certbot-nginx

# Firewall: hanya SSH, HTTP, HTTPS yang terbuka ke internet
sudo ufw allow OpenSSH && sudo ufw allow 'Nginx Full' && sudo ufw enable

# Jam server harus akurat: request bertanda tangan ditolak jika selisih jam > 5 menit
timedatectl    # pastikan "System clock synchronized: yes"
```

## 2. Siapkan folder

**A. Server pelanggan (image privat).** Vendor memberikan dua file dari `deploy/customer/` (`docker-compose.yml`,
`.env_example`) dan satu **token baca** registry (lihat bagian 12).

```bash
sudo mkdir -p /opt/secure-patrol && sudo chown $USER /opt/secure-patrol && cd /opt/secure-patrol
# salin docker-compose.yml dan .env_example dari vendor ke folder ini
echo "<token dari vendor>" | docker login ghcr.io -u <username dari vendor> --password-stdin
```

**B. Server vendor (build dari source).**

```bash
git clone https://github.com/alifnrz27/secure-patrol-backend.git /opt/secure-patrol-backend
cd /opt/secure-patrol-backend
```

## 3. Buat file `.env`

```bash
cp .env_example .env
chmod 600 .env
openssl rand -base64 48                        # jalankan 2x: untuk JWT_SECRET dan APP_MASTER_SECRET
openssl rand -base64 24 | tr -d '/+='          # untuk DB_PASSWORD
nano .env
```

Isi minimal untuk production:

```ini
# Database: container secure-postgres dibuat otomatis dengan nama, user, dan password ini
DB_HOST = "secure-postgres"
DB_PORT = "5432"
DB_NAME = "secure_patrol"
DB_USERNAME = "secure_patrol"
DB_PASSWORD = "<hasil openssl untuk DB_PASSWORD>"
DB_SSLMODE = "disable"             # koneksi di dalam jaringan Docker
DB_TIMEZONE = "Asia/Jakarta"
DB_LOG_LEVEL = "1"

PORT = "9001"
APP_ENV = "production"             # "development" untuk mengaktifkan halaman /docs
APP_TIMEZONE = "Asia/Jakarta"

SEED_DUMMY_DATA = "false"          # jangan buat user dummy di production

JWT_SECRET = "<hasil openssl #1>"
APP_MASTER_SECRET = "<hasil openssl #2>"

STORAGE_PATH = "./storage"         # di container: /app/storage (volume secure-backend-storage)

# IP asli pengunjung dari nginx (untuk audit log). 172.16.0.0/12 = jaringan Docker.
PROXY_HEADER = "X-Forwarded-For"
TRUSTED_PROXIES = "172.16.0.0/12"

REDIS_HOST = ""                    # "secure-redis" jika memakai profile redis
```

- **`DB_NAME`, `DB_USERNAME`, `DB_PASSWORD` hanya dipakai saat database dibuat pertama kali.** Setelah volume
  `secure-postgres-data` ada, mengubah nilai ini di `.env` **tidak** mengubah user/password di database; ubah
  password lewat `ALTER USER` (bagian 11) lalu samakan `.env`.
- **`APP_MASTER_SECRET` tidak boleh diganti** setelah dipakai: semua App ID/App Key akan tidak valid, dan
  **Install ID berubah sehingga license berhenti berfungsi** (harus minta license baru ke vendor).
  Simpan salinan `.env` di tempat aman (password manager / vault).
- `MTSEL_*` dan `RABBITMQ_*` tidak dipakai; boleh dihapus dari `.env`.
- Aturan patroli, lockout login, masa berlaku token, dan validasi wajah **tidak** diatur di `.env`; nilainya ada di
  database dan diubah lewat menu Pengaturan Sistem (`PUT /api/v1/settings`), lihat `docs/AUTH.md` bagian 7.

## 4. Jalankan

```bash
docker compose pull && docker compose up -d     # A. image privat (server pelanggan)
docker compose up -d --build                    # B. build dari source (server vendor)
```

Compose menjalankan `secure-postgres` lebih dulu, menunggu database siap, baru menjalankan `secure-backend`.
Saat pertama kali jalan, database dan tabel dibuat otomatis (migrasi), begitu juga pengaturan sistem global dengan
nilai default. Shift dibuat per unit: setiap unit baru otomatis mendapat 3 shift default. Cek:

```bash
docker compose ps                           # keduanya "running (healthy)"
docker compose logs backend | grep -E "migration|settings"
curl -s http://127.0.0.1:9001/health        # OK
```

## 5. Data awal: Super-Admin dan App Client

```bash
# Akun Super-Admin pertama; juga membuat 5 role sistem. Password ditampilkan sekali, segera ganti setelah login.
docker compose exec backend /app/secure-patrol-backend create-super-admin -email admin@perusahaan.com

# App ID / App Key untuk setiap aplikasi (App Key ditampilkan sekali; simpan dengan aman)
docker compose exec backend /app/secure-patrol-backend create-app-client -name "Secure Patrol Web" -platform web
docker compose exec backend /app/secure-patrol-backend create-app-client -name "Secure Patrol Android" -platform android
docker compose exec backend /app/secure-patrol-backend create-app-client -name "Secure Patrol iOS" -platform ios
```

App ID/Key web dipakai frontend, Android/iOS dipakai aplikasi mobile.

Selanjutnya dari web admin:

1. Super-Admin membuat **unit** (menu Unit). Setiap unit baru otomatis mendapat 3 shift default.
2. Super-Admin membuat Kepala/Admin Keamanan untuk setiap unit (pilih unitnya), serta Manager Keamanan (pusat).
3. Kepala/Admin Keamanan mengatur titik patroli, shift, petugas, dan setting unitnya sendiri.

## 5b. License

Tanpa license aktif sistem **terkunci**: hanya Super-Admin yang bisa login, dan hanya untuk memasang license.

```bash
# 1. Ambil Install ID, kirim ke vendor
docker compose exec backend /app/secure-patrol-backend license-info

# 2. Pasang kode license dari vendor (kode panjang diawali SPL1.)
docker compose exec backend /app/secure-patrol-backend license-install -code "SPL1...."
```

`license-info` juga menampilkan status, tanggal berakhir, dan pemakaian (unit dan App Client aktif terhadap
batas license). Perpanjangan bisa dipasang dari web admin (**Pengaturan → License**, Super-Admin) atau dengan
perintah yang sama.

- License berisi batas **unit aktif** dan **App Client aktif**, tanggal berakhir, dan masa tenggang.
- Unit/App Client di atas batas (termasuk yang diaktifkan langsung di database) tidak berfungsi; yang paling lama
  tetap jalan. Pemeriksaan ulang berjalan setiap menit.
- Setelah masa tenggang habis, sistem terkunci sampai license baru dipasang. Data tidak dihapus.
- **Jam server harus akurat.** Jika jam server mundur lebih dari 1 hari dibanding data yang sudah tercatat, license
  dianggap tidak valid.
- Membuat pasangan kunci dan menerbitkan license (vendor): `docs/LICENSE_SPEC.md`.

## 6. nginx dan HTTPS

`/etc/nginx/sites-available/secure-backend`:

```nginx
server {
    listen 80;
    server_name api.securepatrol.id;

    # 3 foto scan @ 5 MB + data form; backend menolak body > 20 MB
    client_max_body_size 20m;

    location / {
        proxy_pass http://127.0.0.1:9001;          # sama dengan PORT di .env
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        # Timpa (jangan tambahkan) agar client tidak bisa memalsukan IP di audit log
        proxy_set_header X-Forwarded-For $remote_addr;
        proxy_set_header X-Forwarded-Proto $scheme;
        # Export Excel riwayat scan bisa memakan waktu
        proxy_read_timeout 120s;
    }
}
```

```bash
sudo ln -s /etc/nginx/sites-available/secure-backend /etc/nginx/sites-enabled/
sudo nginx -t && sudo systemctl reload nginx
sudo certbot --nginx -d api.securepatrol.id      # HTTPS + perpanjangan otomatis
curl -s https://api.securepatrol.id/health      # OK
```

**Frontend** berjalan di container/compose sendiri dengan server block/domain sendiri (misalnya
`admin.securepatrol.id`). Arahkan base URL API frontend ke `https://api.securepatrol.id` dan isi App ID/Key web dari
langkah 5.

**Server staging tanpa domain.** Untuk membuka API langsung di `http://IP-SERVER:9001`, ubah port backend di
`docker-compose.yml` dari `"127.0.0.1:${PORT:-3000}:${PORT:-3000}"` menjadi `"${PORT:-3000}:${PORT:-3000}"`, lalu
`docker compose up -d --force-recreate`, dan izinkan TCP 9001 di firewall panel VPS. Port yang dibuka Docker **tidak**
diblokir `ufw`, jadi jangan lakukan ini di production.

## 7. Menjalankan ulang dan update

Semua perintah dijalankan di `/opt/secure-patrol-backend`.

| Kebutuhan | Perintah |
|---|---|
| Update (A. image privat) | ubah `IMAGE_TAG` di `.env` (atau biarkan `latest`), lalu `docker compose pull && docker compose up -d` |
| Update (B. build dari source) | `git pull && docker compose up -d --build` |
| Terapkan perubahan `.env` | `docker compose up -d --force-recreate` |
| Restart backend | `docker compose restart backend` |
| Hentikan semua | `docker compose down` |
| Status / log | `docker compose ps` · `docker compose logs -f backend` |

- `docker compose down` dan build ulang **tidak** menghapus data (database dan foto ada di volume).
  **Jangan** memakai `docker compose down -v`: itu menghapus database, foto, dan log.
- Migrasi database berjalan otomatis saat backend start. **Backup database sebelum update** (bagian 9).
- **Update ke versi multi unit:** data lama (titik patroli, shift, group patroli, dan user Kepala/Admin/Tim
  Keamanan) otomatis dimasukkan ke unit pertama, atau ke unit baru **"Unit Utama"** (`UNIT-UTAMA`) jika belum
  ada unit. Koordinat unit diambil dari titik patroli pertama. Setelah update, cek dan ubah nama, kode, dan lokasi
  unit tersebut dari menu Unit. Pengaturan sistem yang sudah diubah tetap dipakai sebagai nilai global. Cek log:
  `docker compose logs backend | grep -E "unit|assigned"`.
- Compose ini hanya mengurus Secure Patrol. Frontend dan aplikasi lain di server ini (termasuk database mereka)
  tidak tersentuh.

## 8. Rollback

**A. Image privat:** isi `IMAGE_TAG` di `.env` dengan versi sebelumnya (mis. `1.0.0`), lalu
`docker compose pull && docker compose up -d`.

**B. Build dari source:**

Simpan image yang sedang jalan sebelum update, supaya bisa kembali jika versi baru bermasalah:

```bash
docker tag secure-backend:latest secure-backend:previous    # sebelum git pull
git pull && docker compose up -d --build

# Jika perlu kembali:
docker tag secure-backend:previous secure-backend:latest
docker compose up -d --force-recreate
```

## 9. Log dan backup

```bash
docker compose logs -f backend                                        # startup, migrasi, error fatal
docker compose logs -f postgres                                       # log database
docker compose exec backend tail -f /app/logs/debug/debug.log         # log aplikasi
docker compose exec backend tail -f /app/logs/access/access.log       # log request
```

Backup (jalankan terjadwal, misalnya lewat cron setiap malam, dan simpan di luar server):

```bash
cd /opt/secure-patrol-backend && mkdir -p ~/backups
# Database
docker compose exec -T postgres sh -c 'pg_dump -U "$POSTGRES_USER" -Fc "$POSTGRES_DB"' > ~/backups/db-$(date +%F).dump
# Foto wajah dan foto scan
docker run --rm -v secure-backend-storage:/data -v ~/backups:/backup alpine \
  tar czf /backup/storage-$(date +%F).tgz -C /data .
```

Restore:

```bash
cd /opt/secure-patrol-backend
docker compose exec -T postgres sh -c 'pg_restore -U "$POSTGRES_USER" -d "$POSTGRES_DB" --clean --if-exists' < ~/backups/db-2026-09-30.dump
docker run --rm -v secure-backend-storage:/data -v ~/backups:/backup alpine \
  tar xzf /backup/storage-2026-09-30.tgz -C /data
docker compose restart backend
```

Backup juga file `.env` (terutama `APP_MASTER_SECRET`, `JWT_SECRET`, dan `DB_PASSWORD`) di tempat aman.

## 10. Membuka database dengan tool (opsional)

Database sengaja tidak membuka port. Untuk memakai `psql` langsung:

```bash
docker compose exec postgres sh -c 'psql -U "$POSTGRES_USER" -d "$POSTGRES_DB"'
```

Untuk tool di laptop (DBeaver, TablePlus): buka komentar `ports: - "127.0.0.1:5433:5432"` pada service `postgres` di
`docker-compose.yml`, jalankan `docker compose up -d`, lalu dari laptop buat SSH tunnel
`ssh -L 5433:127.0.0.1:5433 USER@IP-SERVER` dan hubungkan tool ke `127.0.0.1:5433`.

## 11. Troubleshooting

| Gejala | Penyebab / solusi |
|---|---|
| `docker compose up` gagal: `required variable DB_PASSWORD is missing a value` | Isi `DB_NAME`, `DB_USERNAME`, `DB_PASSWORD` di `.env` |
| Backend `Restarting`, log: `JWT_SECRET must be set and at least 32 characters long` | Isi `JWT_SECRET` dan `APP_MASTER_SECRET` (≥ 32 karakter), lalu `docker compose up -d --force-recreate` |
| Log: `password authentication failed for user` setelah `DB_PASSWORD` diubah | Password di `.env` hanya dipakai saat database pertama dibuat. Samakan di database (lihat di bawah tabel), lalu `docker compose up -d --force-recreate` |
| Log: `no such host` untuk database | `DB_HOST` harus `"secure-postgres"` |
| Upload foto gagal dengan 413 dari nginx | `client_max_body_size 20m;` belum dipasang di nginx |
| Semua request 401 `request timestamp is invalid or outside the allowed window` | Jam server atau jam HP/komputer pengguna meleset > 5 menit; aktifkan NTP (`sudo timedatectl set-ntp true`) |
| IP di audit log selalu `172.x.x.x` | `PROXY_HEADER` / `TRUSTED_PROXIES` belum diisi, atau nginx tidak mengirim `X-Forwarded-For` |
| Container `unhealthy` | `docker compose logs backend`; pastikan `PORT` di `.env` sama dengan port di nginx |
| Semua login selain Super-Admin `403 license is not active, contact your administrator` | License belum dipasang, berakhir, atau tidak valid. Cek `license-info` (bagian 5b) |
| `license-info`: `the server clock (...) is behind data already recorded` | Jam server mundur. Aktifkan NTP (`sudo timedatectl set-ntp true`); status pulih dalam 1 menit |
| `this license was issued for another installation` | Install ID berbeda (license milik instalasi lain, atau `APP_MASTER_SECRET` diganti). Minta license untuk Install ID dari `license-info` |
| User unit: `403 your unit exceeds the license limit` / request: `this app client exceeds the license limit` | Unit/App Client aktif melebihi license. Nonaktifkan yang tidak dipakai atau minta upgrade license |
| `docker compose pull`: `denied` / `unauthorized` | Token registry salah/dicabut; ulangi `docker login ghcr.io` dengan token dari vendor |

**Mengganti password database** (setelah database sudah dibuat):

```bash
docker compose exec postgres sh -c 'psql -U "$POSTGRES_USER" -d "$POSTGRES_DB"'
```

Di dalam psql (ganti `secure_patrol` jika `DB_USERNAME` berbeda):

```sql
ALTER USER secure_patrol WITH PASSWORD 'PASSWORD_BARU';
\q
```

Lalu isi `DB_PASSWORD = "PASSWORD_BARU"` di `.env` dan jalankan `docker compose up -d --force-recreate`.

## 12. Rilis image (vendor)

Image untuk pelanggan dibangun oleh GitHub Actions (`.github/workflows/release-image.yml`) dan disimpan di GitHub
Container Registry **privat**. Kode license di dalam binary di-obfuscate dengan `garble` (hanya paket license, agar
nama tabel database tidak berubah).

```bash
git tag v1.0.0 && git push origin v1.0.0     # -> ghcr.io/alifnrz27/secure-patrol-backend:1.0.0 dan :latest
```

Sekali saja:

1. Setelah workflow pertama selesai, buka GitHub → profil → **Packages** → `secure-patrol-backend` →
   **Package settings**: pastikan **Visibility: Private**.
2. Token untuk pelanggan: buat akun GitHub khusus (mis. `securepatrol-deploy`), undang ke package dengan akses
   **Read**, lalu buat **Personal access token (classic)** di akun itu dengan scope **`read:packages`** saja. Beri
   pelanggan username + token tersebut. Sebaiknya satu token per pelanggan agar bisa dicabut sendiri-sendiri
   (hentikan update untuk pelanggan yang tidak memperpanjang).

**Menerbitkan license** memakai private key yang hanya ada di laptop/generator vendor — lihat `docs/LICENSE_SPEC.md`.
Private key tidak pernah masuk ke repository, image, atau server pelanggan.

**Server vendor yang sudah berjalan** (build dari source, mis. staging): setelah update ke versi dengan license,
sistem terkunci sampai license dipasang. Jalankan `license-info`, terbitkan license untuk Install ID tersebut, lalu
`license-install`.
