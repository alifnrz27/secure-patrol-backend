package seeder

import (
	"log"
	"secure-patrol-backend/models"

	"gorm.io/gorm"
)

// helpDeskArticles are the initial rules, usage guides and FAQ. The numbers in
// the text (radius, photo limits, lockout, offline window) match the defaults
// of the application; update them here if the defaults change.
var helpDeskArticles = []models.HelpDeskArticle{
	{
		Category:  models.HelpDeskCategoryRule,
		SortOrder: 1,
		Title:     "Aturan Umum Petugas Patroli",
		Content: `Setiap petugas patroli wajib:

1. Hadir **15 menit sebelum shift dimulai** dan melakukan serah terima dengan petugas shift sebelumnya.
2. Memakai seragam lengkap dan tanda pengenal selama bertugas.
3. Memastikan HP dalam keadaan aktif, baterai cukup, serta **GPS dan NFC menyala**.
4. Menyelesaikan seluruh titik patroli pada daftar patroli shift yang sedang berjalan.
5. Tidak meninggalkan area tugas tanpa izin Kepala Keamanan.
6. Segera melaporkan kejadian darurat kepada Kepala Keamanan melalui telepon, lalu mencatatnya di aplikasi.`,
	},
	{
		Category:  models.HelpDeskCategoryRule,
		SortOrder: 2,
		Title:     "Aturan Scan Titik Patroli (NFC)",
		Content: `- Scan NFC **harus dilakukan langsung di lokasi titik patroli**. Titik yang mewajibkan validasi lokasi hanya menerima scan dalam radius **100 meter** dari titik.
- **Dilarang menitipkan scan** kepada petugas lain. Setiap scan tercatat atas nama akun yang login.
- Pilih kondisi **Normal** jika tidak ada temuan, atau **Tidak Normal** jika ada temuan. Kondisi Tidak Normal **wajib disertai catatan**.
- Lampirkan foto sebagai bukti, **maksimal 3 foto** dengan ukuran masing-masing **maksimal 5 MB** (JPEG/PNG).
- Satu titik boleh di-scan lebih dari satu kali dalam satu shift, misalnya saat patroli ulang.`,
	},
	{
		Category:  models.HelpDeskCategoryRule,
		SortOrder: 3,
		Title:     "Kebijakan Keamanan Akun",
		Content: `- **Jangan pernah membagikan password** kepada siapa pun, termasuk sesama petugas.
- Password minimal **8 karakter** dan harus mengandung **huruf dan angka**.
- Akun akan **terkunci selama 5 menit** setelah **3 kali salah password**.
- Jika HP hilang, segera laporkan ke Admin Keamanan agar semua sesi login pada akun Anda diakhiri.
- Gunakan menu **Logout dari semua perangkat** jika Anda curiga akun Anda dipakai orang lain.`,
	},
	{
		Category:  models.HelpDeskCategoryGuide,
		SortOrder: 1,
		Title:     "Cara Login ke Aplikasi",
		Content: `1. Buka aplikasi Secure Patrol.
2. Masukkan **email** dan **password** yang diberikan oleh Admin Keamanan.
3. Tekan tombol **Masuk**.
4. Setelah berhasil, halaman utama menampilkan shift yang sedang berjalan beserta daftar titik patroli.

Belum punya akun atau lupa password? Hubungi Admin Keamanan untuk dibuatkan akun atau di-reset password-nya.`,
	},
	{
		Category:  models.HelpDeskCategoryGuide,
		SortOrder: 2,
		Title:     "Cara Melakukan Patroli (Scan NFC)",
		Content: `1. Buka halaman **Patroli**. Aplikasi menampilkan shift yang sedang berjalan dan daftar titik yang harus dikunjungi.
2. Datangi titik patroli, lalu tempelkan HP ke **tag NFC** di lokasi tersebut.
3. Jika diminta, lakukan **validasi wajah** dengan menghadap kamera depan di tempat yang cukup terang.
4. Pilih kondisi **Normal** atau **Tidak Normal**, isi catatan bila perlu, dan ambil foto (maksimal 3).
5. Tekan **Kirim**. Titik yang sudah di-scan ditandai selesai pada daftar patroli.

Shift ditentukan otomatis dari waktu scan. Scan tepat pada jam akhir shift (misalnya pukul 16:00) masuk ke shift berikutnya.`,
	},
	{
		Category:  models.HelpDeskCategoryGuide,
		SortOrder: 3,
		Title:     "Cara Melaporkan Kondisi Tidak Normal",
		Content: `Gunakan kondisi **Tidak Normal** untuk temuan seperti pintu tidak terkunci, lampu mati, kerusakan fasilitas, atau orang mencurigakan.

1. Scan NFC di titik patroli seperti biasa.
2. Pilih kondisi **Tidak Normal**.
3. Tulis **catatan** yang jelas: apa yang ditemukan, di mana, dan tindakan yang sudah diambil. Catatan wajib diisi.
4. Ambil **foto** temuan (maksimal 3 foto).
5. Tekan **Kirim**, lalu laporkan juga ke Kepala Keamanan jika temuan membutuhkan tindakan segera.`,
	},
	{
		Category:  models.HelpDeskCategoryGuide,
		SortOrder: 4,
		Title:     "Menggunakan Aplikasi Saat Tidak Ada Sinyal",
		Content: `Scan tetap bisa dilakukan di area tanpa sinyal, misalnya basement atau gudang.

- Hasil scan **disimpan sementara di HP** dan dikirim otomatis saat sinyal kembali.
- Waktu yang tercatat adalah **waktu saat scan dilakukan**, bukan waktu saat data terkirim, sehingga scan tetap masuk ke shift yang benar.
- Pastikan data terkirim **paling lambat 24 jam** setelah scan. Scan yang lebih lama dari itu akan ditolak.
- **Jangan menghapus data aplikasi atau logout** sebelum semua scan terkirim.`,
	},
	{
		Category:  models.HelpDeskCategoryFAQ,
		SortOrder: 1,
		Title:     "Scan saya ditolak karena jarak terlalu jauh, apa yang harus dilakukan?",
		Content: `Titik patroli tersebut mewajibkan Anda berada dalam radius **100 meter** dari lokasinya.

- Pastikan Anda benar-benar berada di lokasi titik patroli.
- Aktifkan **GPS mode akurasi tinggi**, lalu tunggu beberapa detik sampai lokasi stabil.
- Jika berada di dalam gedung, coba mendekat ke jendela atau area terbuka, lalu scan ulang.

Jika masih ditolak padahal Anda sudah berada di lokasi, laporkan ke Admin Keamanan karena koordinat titik mungkin perlu diperbarui.`,
	},
	{
		Category:  models.HelpDeskCategoryFAQ,
		SortOrder: 2,
		Title:     "Validasi wajah gagal terus, bagaimana solusinya?",
		Content: `- Pastikan wajah terlihat jelas dan **pencahayaan cukup**. Hindari membelakangi lampu atau matahari.
- Lepaskan masker, topi, atau kacamata hitam saat validasi.
- Bersihkan lensa kamera depan.
- Foto wajah pembanding diambil dari data akun Anda. Jika penampilan Anda sudah banyak berubah, minta Admin Keamanan untuk **memperbarui foto wajah** Anda.`,
	},
	{
		Category:  models.HelpDeskCategoryFAQ,
		SortOrder: 3,
		Title:     "Akun saya terkunci, kapan bisa login lagi?",
		Content: `Akun terkunci otomatis setelah **3 kali salah memasukkan password**.

- Tunggu **5 menit**, lalu coba login kembali dengan password yang benar.
- Jika lupa password, minta Admin Keamanan untuk melakukan **reset password**. Reset password juga langsung membuka kunci akun.`,
	},
}

func seedHelpDeskArticles(db *gorm.DB) error {
	created := 0

	for _, article := range helpDeskArticles {
		article.IsPublished = true

		result := db.Where(models.HelpDeskArticle{Category: article.Category, Title: article.Title}).
			Attrs(article).
			FirstOrCreate(&article)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected > 0 {
			created++
		}
	}

	log.Printf("[seeder] seeded %d new help desk article(s)", created)
	return nil
}
