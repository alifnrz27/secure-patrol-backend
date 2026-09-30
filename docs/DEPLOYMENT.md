# Deploy Secure Patrol Backend dengan Docker Compose

Panduan dari server sampai API berjalan di HTTPS, lalu cara menjalankan ulang, update, rollback, dan backup.
PostgreSQL memakai server database yang **sudah ada**; frontend adalah container terpisah.

```
Internet ──HTTPS──▶ nginx (server) ──▶ secure-backend (container, 127.0.0.1:3000) ──▶ PostgreSQL yang sudah ada
```

| Nama | Isi |
|---|---|
| `secure-backend` | Container API ini |
| `secure-backend-storage` | Volume foto wajah dan foto scan (**wajib di-backup**) |
| `secure-backend-logs` | Volume log aplikasi |
| `secure-redis` | Opsional, hanya jika backend dijalankan lebih dari satu container |

Menjalankan atau menjalankan ulang cukup satu perintah di folder project:

```bash
docker compose up -d --build
```

## 1. Siapkan server

Contoh untuk Ubuntu 22.04/24.04 dengan domain API (misalnya `api.securepatrol.id`) yang sudah diarahkan ke IP server.

```bash
# Docker + Docker Compose (skrip resmi), git, nginx, certbot
curl -fsSL https://get.docker.com | sudo sh
sudo usermod -aG docker $USER        # lalu logout-login agar bisa memakai docker tanpa sudo
sudo apt-get update && sudo apt-get install -y git nginx certbot python3-certbot-nginx

# Firewall: hanya SSH, HTTP, HTTPS yang terbuka ke internet. Port 3000 tidak perlu dibuka.
sudo ufw allow OpenSSH && sudo ufw allow 'Nginx Full' && sudo ufw enable

# Jam server harus akurat: request bertanda tangan ditolak jika selisih jam > 5 menit
timedatectl    # pastikan "System clock synchronized: yes"
```

## 2. Ambil kode

```bash
sudo mkdir -p /opt/secure-patrol && sudo chown $USER /opt/secure-patrol
git clone https://github.com/alifnrz27/secure-patrol-backend.git /opt/secure-patrol/backend
cd /opt/secure-patrol/backend
```

## 3. Siapkan database di PostgreSQL yang sudah ada

Buat user dan database khusus (sekali saja). Tabelnya dibuat otomatis oleh backend saat start.

```bash
DB_PASSWORD=$(openssl rand -base64 24 | tr -d '/+=')
echo "Simpan password database ini: $DB_PASSWORD"

sudo -u postgres psql <<SQL
CREATE USER secure_patrol WITH PASSWORD '$DB_PASSWORD';
CREATE DATABASE secure_patrol OWNER secure_patrol;
SQL
```

Lalu pastikan container bisa menjangkau PostgreSQL. Pilih sesuai letak PostgreSQL-mu:

**A. PostgreSQL terpasang langsung di server ini** (paling umum). Container menghubunginya lewat
`host.docker.internal`, yang datang dari jaringan Docker (`172.16.0.0/12`), sehingga perlu diizinkan di tiga tempat:

```bash
# 1) PostgreSQL menerima koneksi selain dari localhost
#    /etc/postgresql/<versi>/main/postgresql.conf
listen_addresses = '*'
#    Aman selama langkah 2 dan 3 membatasi siapa yang boleh masuk. Jangan pakai alamat interface Docker
#    (172.17.0.1): saat reboot PostgreSQL bisa start sebelum Docker dan tidak mendengarkan di sana.

# 2) Izinkan login hanya dari jaringan Docker
#    /etc/postgresql/<versi>/main/pg_hba.conf  (tambahkan baris ini)
host    secure_patrol    secure_patrol    172.16.0.0/12    scram-sha-256

# 3) Firewall: buka 5432 hanya untuk jaringan Docker; untuk internet tetap tertutup
sudo ufw allow from 172.16.0.0/12 to any port 5432 proto tcp

sudo systemctl restart postgresql
```

Di `.env`: `DB_HOST = "host.docker.internal"`.

**B. PostgreSQL berjalan sebagai container Docker lain.** Gabungkan backend ke network container tersebut dengan
membuat `docker-compose.override.yml` di folder project (dibaca otomatis oleh `docker compose`):

```yaml
services:
  backend:
    networks: [default, database]
networks:
  database:
    external: true
    name: <nama-network-postgres>   # lihat dengan: docker inspect <container-postgres> --format '{{json .NetworkSettings.Networks}}'
```

Di `.env`: `DB_HOST = "<nama-container-postgres>"`.

**C. Database terkelola / server lain.** Di `.env`: `DB_HOST = "<host database>"` dan `DB_SSLMODE = "require"`.

## 4. Buat file `.env`

```bash
cp .env_example .env
chmod 600 .env
openssl rand -base64 48   # jalankan dua kali: untuk JWT_SECRET dan APP_MASTER_SECRET
nano .env
```

Isi minimal untuk production:

```ini
DB_HOST = "host.docker.internal"   # sesuai langkah 3
DB_PORT = "5432"
DB_NAME = "secure_patrol"
DB_USERNAME = "secure_patrol"
DB_PASSWORD = "<password dari langkah 3>"
DB_SSLMODE = "disable"             # "require" untuk database di server lain
DB_TIMEZONE = "Asia/Jakarta"
DB_LOG_LEVEL = "1"

PORT = "3000"
APP_ENV = "production"             # halaman /docs otomatis nonaktif di production
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

- **`APP_MASTER_SECRET` tidak boleh diganti** setelah dipakai: semua App ID/App Key akan tidak valid.
  Simpan salinan `.env` di tempat aman (password manager / vault).
- `MTSEL_*` dan `RABBITMQ_*` tidak dipakai; boleh dihapus dari `.env`.
- Aturan patroli, lockout login, masa berlaku token, dan validasi wajah **tidak** diatur di `.env`; nilainya ada di
  database dan diubah lewat menu Pengaturan Sistem (`PUT /api/v1/settings`), lihat `docs/AUTH.md` bagian 7.

## 5. Jalankan

```bash
docker compose up -d --build
```

Saat pertama kali jalan, tabel dibuat otomatis (migrasi), begitu juga 3 shift default dan pengaturan sistem dengan
nilai default. Cek:

```bash
docker compose ps                           # secure-backend harus "running (healthy)"
docker compose logs backend | grep -E "migration|settings"
curl -s http://127.0.0.1:3000/health        # OK
```

## 6. Data awal: Super-Admin dan App Client

```bash
# Akun Super-Admin pertama; juga membuat 5 role sistem. Password ditampilkan sekali, segera ganti setelah login.
docker compose exec backend /app/secure-patrol-backend create-super-admin -email admin@perusahaan.com

# App ID / App Key untuk setiap aplikasi (App Key ditampilkan sekali; simpan dengan aman)
docker compose exec backend /app/secure-patrol-backend create-app-client -name "Secure Patrol Web" -platform web
docker compose exec backend /app/secure-patrol-backend create-app-client -name "Secure Patrol Android" -platform android
docker compose exec backend /app/secure-patrol-backend create-app-client -name "Secure Patrol iOS" -platform ios
```

App ID/Key web dipakai frontend, Android/iOS dipakai aplikasi mobile. Setelah itu user, titik patroli, shift, dan
pengaturan dikelola dari web admin.

## 7. nginx dan HTTPS

`/etc/nginx/sites-available/secure-backend`:

```nginx
server {
    listen 80;
    server_name api.securepatrol.id;

    # 3 foto scan @ 5 MB + data form; backend menolak body > 20 MB
    client_max_body_size 20m;

    location / {
        proxy_pass http://127.0.0.1:3000;
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
langkah 6.

## 8. Menjalankan ulang dan update

Semua perintah dijalankan di `/opt/secure-patrol/backend`.

| Kebutuhan | Perintah |
|---|---|
| Update ke kode terbaru | `git pull && docker compose up -d --build` |
| Terapkan perubahan `.env` | `docker compose up -d --force-recreate` |
| Restart | `docker compose restart backend` |
| Hentikan | `docker compose down` |
| Status / log | `docker compose ps` · `docker compose logs -f backend` |

- `docker compose down` dan build ulang **tidak** menghapus foto (volume `secure-backend-storage`), dan database ada
  di PostgreSQL terpisah. **Jangan** memakai `docker compose down -v`, karena itu menghapus volume foto dan log.
- Migrasi database berjalan otomatis saat container start. **Backup database sebelum update** (bagian 10).
- Compose ini hanya mengurus `secure-backend` (dan `secure-redis` jika dipakai). Frontend di-re-run dari
  compose-nya sendiri.

## 9. Rollback

Simpan image yang sedang jalan sebelum update, supaya bisa kembali jika versi baru bermasalah:

```bash
docker tag secure-backend:latest secure-backend:previous    # sebelum git pull
git pull && docker compose up -d --build

# Jika perlu kembali:
docker tag secure-backend:previous secure-backend:latest
docker compose up -d --force-recreate
```

## 10. Log dan backup

```bash
docker compose logs -f backend                                        # startup, migrasi, error fatal
docker compose exec backend tail -f /app/logs/debug/debug.log         # log aplikasi
docker compose exec backend tail -f /app/logs/access/access.log       # log request
```

Backup (jalankan terjadwal, misalnya lewat cron setiap malam, dan simpan di luar server):

```bash
mkdir -p ~/backups && cd ~/backups
# Database (dari PostgreSQL yang sudah ada)
pg_dump -h localhost -U secure_patrol -Fc secure_patrol > db-$(date +%F).dump
# Foto wajah dan foto scan
docker run --rm -v secure-backend-storage:/data -v "$PWD":/backup alpine \
  tar czf /backup/storage-$(date +%F).tgz -C /data .
```

Restore:

```bash
pg_restore -h localhost -U secure_patrol -d secure_patrol --clean --if-exists db-2026-09-30.dump
docker run --rm -v secure-backend-storage:/data -v "$PWD":/backup alpine \
  tar xzf /backup/storage-2026-09-30.tgz -C /data
docker compose restart backend
```

Backup juga file `.env` (terutama `APP_MASTER_SECRET` dan `JWT_SECRET`) di tempat aman.

## 11. Troubleshooting

| Gejala | Penyebab / solusi |
|---|---|
| Container berhenti, log: `JWT_SECRET must be set and at least 32 characters long` | Isi `JWT_SECRET` dan `APP_MASTER_SECRET` (≥ 32 karakter) di `.env`, lalu `docker compose up -d --force-recreate` |
| Log: `connect: connection refused` ke `host.docker.internal:5432` | PostgreSQL hanya mendengarkan di localhost; set `listen_addresses = '*'` — langkah 3A |
| Log: `connect: connection timed out` ke `host.docker.internal:5432` | Firewall memblokir; `sudo ufw allow from 172.16.0.0/12 to any port 5432 proto tcp` |
| Log: `no pg_hba.conf entry for host "172.x.x.x"` | Tambahkan baris `pg_hba.conf` di langkah 3A lalu restart PostgreSQL |
| Log: `password authentication failed` | `DB_USERNAME`/`DB_PASSWORD` di `.env` tidak sama dengan user di langkah 3 |
| Upload foto gagal dengan 413 dari nginx | `client_max_body_size 20m;` belum dipasang di nginx |
| Semua request 401 `request timestamp is invalid or outside the allowed window` | Jam server atau jam HP/komputer pengguna meleset > 5 menit; aktifkan NTP (`sudo timedatectl set-ntp true`) |
| IP di audit log selalu `172.x.x.x` | `PROXY_HEADER` / `TRUSTED_PROXIES` belum diisi, atau nginx tidak mengirim `X-Forwarded-For` |
| Container `unhealthy` | `docker compose logs backend`; pastikan `PORT` di `.env` sama dengan port di nginx |
