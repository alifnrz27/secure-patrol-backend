# Spesifikasi License Secure Patrol (untuk repo generator license)

Dokumen ini untuk developer/Claude yang membangun **web admin generator license** (repo terpisah). Backend
Secure Patrol hanya **memverifikasi** license; generator yang **membuat** license. Keduanya memakai format di
bawah ini. Implementasi referensi Node.js dan PHP di dokumen ini sudah diuji: kode yang dihasilkan diterima
backend.

## 1. Kunci

| Kunci | Di mana | Fungsi |
|---|---|---|
| **Private key** (seed Ed25519 32 byte, 64 karakter hex) | **Hanya** di server backend generator, sebagai secret/env `LICENSE_PRIVATE_KEY` | Membuat (menandatangani) license |
| **Public key** `a37b9a34ae55b5b6f1db443071152ee747eab11f493c6d5735f07b1a80c6bd2d` | Tertanam di backend Secure Patrol (`pkg/license/key.go`) | Memeriksa license; **tidak bisa** membuat license |

Aturan private key:

- **Jangan** di-commit ke repo mana pun, **jangan** di frontend/browser, **jangan** di server pelanggan, jangan di log.
- Tanda tangan dibuat di **server** generator (API), bukan di JavaScript browser.
- Simpan cadangan di password manager. Jika hilang, license baru tidak bisa dibuat untuk instalasi yang ada.
- Jika bocor: buat pasangan kunci baru (`go run ./tools/license keygen -out <file>`), ganti public key di backend,
  rilis image baru, dan terbitkan ulang license semua pelanggan.

## 2. Format kode license

```
SPL1.<payload>.<signature>
```

- `payload` = base64url **tanpa padding** dari JSON (UTF-8) isi license.
- `signature` = base64url tanpa padding dari tanda tangan **Ed25519** atas teks ASCII `SPL1.<payload>`
  (prefix, titik, dan payload persis seperti di kode).
- base64url: `+` → `-`, `/` → `_`, buang `=`.

Karena yang ditandatangani adalah teks base64 payload, urutan key dan spasi JSON bebas; backend membaca JSON
setelah tanda tangan cocok. Field tambahan diabaikan.

## 3. Isi payload

| Field | Tipe | Aturan |
|---|---|---|
| `lid` | string | Wajib. ID license unik, mis. `LIC-20261001-8F3A2C` |
| `customer` | string | Nama perusahaan pelanggan |
| `install_id` | string | Wajib. Dari pelanggan (`license-info` / halaman License), format `SP-XXXX-XXXX-XXXX-XXXX` (tidak membedakan huruf besar/kecil) |
| `max_units` | integer | Wajib, ≥ 1. Jumlah **unit aktif** maksimal |
| `max_app_clients` | integer | Wajib, ≥ 1. Jumlah **App Client (App ID/Key) aktif** maksimal (web + android + ios + server) |
| `issued_at` | string RFC3339 | Wajib. Waktu terbit, mis. `2026-10-01T09:00:00+07:00` |
| `expires_at` | string RFC3339 | Wajib, setelah `issued_at`. License berakhir pada waktu ini |
| `grace_days` | integer | 0–365. Masa tenggang setelah `expires_at`; sistem tetap jalan dengan peringatan |

Contoh:

```json
{
  "lid": "LIC-20261001-8F3A2C",
  "customer": "PT Contoh Aman",
  "install_id": "SP-Y3HW-ZDCK-C4CO-BY6F",
  "max_units": 5,
  "max_app_clients": 3,
  "issued_at": "2026-10-01T09:00:00+07:00",
  "expires_at": "2027-10-01T00:00:00+07:00",
  "grace_days": 14
}
```

Perilaku di backend:

- Sebelum `expires_at`: aktif (peringatan mulai 30 hari sebelum berakhir).
- Antara `expires_at` dan `expires_at + grace_days`: masa tenggang, tetap jalan dengan peringatan.
- Sesudahnya, atau jika tanda tangan/`install_id` tidak cocok, atau jam server dimundurkan: **terkunci total**
  (hanya Super-Admin bisa login untuk memasang license baru).
- License yang sudah lewat masa tenggang ditolak saat dipasang.
- License yang dipasang terakhir yang berlaku. Perpanjangan/upgrade = terbitkan license baru dengan `install_id` sama.

## 4. Implementasi referensi

### Node.js (≥ 16, tanpa dependency)

```js
const crypto = require('crypto');

const b64url = (buf) => Buffer.from(buf).toString('base64')
  .replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/, '');

// seedHex: LICENSE_PRIVATE_KEY (64 hex chars) dari env/secret server.
function privateKeyFromSeed(seedHex) {
  const seed = Buffer.from(seedHex.trim(), 'hex');
  if (seed.length !== 32) throw new Error('LICENSE_PRIVATE_KEY must be 64 hex characters');
  const der = Buffer.concat([Buffer.from('302e020100300506032b657004220420', 'hex'), seed]); // PKCS#8
  return crypto.createPrivateKey({ key: der, format: 'der', type: 'pkcs8' });
}

function encodeLicense(payload, seedHex) {
  const data = 'SPL1.' + b64url(JSON.stringify(payload));
  const signature = crypto.sign(null, Buffer.from(data, 'ascii'), privateKeyFromSeed(seedHex));
  return data + '.' + b64url(signature);
}

// Hanya membaca isi (tanpa cek tanda tangan) — untuk tampilan di generator.
function decodeLicense(code) {
  const [prefix, data] = code.trim().split('.');
  if (prefix !== 'SPL1' || !data) throw new Error('not a license code');
  return JSON.parse(Buffer.from(data.replace(/-/g, '+').replace(/_/g, '/'), 'base64').toString('utf8'));
}

// Memeriksa kode dengan public key (mis. untuk tes di generator).
const PUBLIC_KEY_HEX = 'a37b9a34ae55b5b6f1db443071152ee747eab11f493c6d5735f07b1a80c6bd2d';
function verifyLicense(code) {
  const [prefix, data, sig] = code.trim().split('.');
  const der = Buffer.concat([Buffer.from('302a300506032b6570032100', 'hex'), Buffer.from(PUBLIC_KEY_HEX, 'hex')]);
  const key = crypto.createPublicKey({ key: der, format: 'der', type: 'spki' });
  const signature = Buffer.from(sig.replace(/-/g, '+').replace(/_/g, '/'), 'base64');
  return prefix === 'SPL1' && crypto.verify(null, Buffer.from(prefix + '.' + data, 'ascii'), key, signature);
}
```

### PHP (≥ 7.2, ext-sodium)

```php
function b64url(string $bin): string {
    return rtrim(strtr(base64_encode($bin), '+/', '-_'), '=');
}

// $seedHex: LICENSE_PRIVATE_KEY (64 hex chars) dari env/secret server.
function encode_license(array $payload, string $seedHex): string {
    $seed = hex2bin(trim($seedHex));
    if ($seed === false || strlen($seed) !== SODIUM_CRYPTO_SIGN_SEEDBYTES) {
        throw new InvalidArgumentException('LICENSE_PRIVATE_KEY must be 64 hex characters');
    }
    $secretKey = sodium_crypto_sign_secretkey(sodium_crypto_sign_seed_keypair($seed));
    $data = 'SPL1.' . b64url(json_encode($payload, JSON_UNESCAPED_SLASHES | JSON_UNESCAPED_UNICODE));
    return $data . '.' . b64url(sodium_crypto_sign_detached($data, $secretKey));
}

function decode_license(string $code): array {
    $parts = explode('.', trim($code));
    if (count($parts) !== 3 || $parts[0] !== 'SPL1') {
        throw new InvalidArgumentException('not a license code');
    }
    return json_decode(base64_decode(strtr($parts[1], '-_', '+/')), true);
}

function verify_license(string $code): bool {
    $parts = explode('.', trim($code));
    if (count($parts) !== 3 || $parts[0] !== 'SPL1') return false;
    $pub = hex2bin('a37b9a34ae55b5b6f1db443071152ee747eab11f493c6d5735f07b1a80c6bd2d');
    $sig = base64_decode(strtr($parts[2], '-_', '+/'));
    return sodium_crypto_sign_verify_detached($sig, $parts[0] . '.' . $parts[1], $pub);
}
```

### Go / CLI (repo backend, di laptop vendor)

```bash
go run ./tools/license issue -key ~/.secure-patrol/license_private.key \
  -install-id SP-Y3HW-ZDCK-C4CO-BY6F -customer "PT Contoh Aman" \
  -units 5 -app-clients 3 -expires 2027-10-01 -grace 14
go run ./tools/license decode SPL1....
```

## 5. Kebutuhan web admin generator

- Login khusus vendor (bukan untuk pelanggan), sebaiknya dengan 2FA.
- Data pelanggan: nama, kontak, daftar instalasi (Install ID).
- Form terbitkan license: pelanggan, Install ID, `max_units`, `max_app_clients`, tanggal berakhir, `grace_days`
  (default 14). `lid` dibuat otomatis; `issued_at` = sekarang.
- Validasi sebelum menandatangani: aturan di bagian 3 (Install ID format `SP-XXXX-XXXX-XXXX-XXXX`).
- Riwayat license yang diterbitkan (siapa, kapan, isi) — simpan kode lengkapnya agar bisa dikirim ulang.
- Tombol salin kode; kirim ke pelanggan lewat kanal aman.
- Private key hanya dibaca dari env server; tidak pernah ditampilkan, dikirim ke browser, atau disimpan di database.
- Uji: kode buatan generator dipasang di backend development (`PUT /api/v1/license` atau
  `secure-patrol-backend license-install -code ...`) harus berstatus `active`.

## 6. Alur pelanggan

1. Pelanggan memasang backend, lalu menjalankan
   `docker compose exec backend /app/secure-patrol-backend license-info` dan mengirim **Install ID** ke vendor.
2. Vendor menerbitkan license di generator dan mengirim kodenya.
3. Pelanggan memasang: `docker compose exec backend /app/secure-patrol-backend license-install -code "SPL1...."`
   (instalasi baru) atau lewat web admin **Pengaturan → License** (Super-Admin).
4. Perpanjangan/upgrade: ulangi langkah 2–3 dengan Install ID yang sama.

Install ID diturunkan dari `APP_MASTER_SECRET` di `.env` pelanggan. Nilai itu tidak boleh berubah; jika berubah,
semua App Client dan license lama berhenti berfungsi dan license harus diterbitkan ulang.
