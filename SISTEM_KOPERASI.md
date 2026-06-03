# Dokumentasi Sistem Aplikasi Koperasi Simpan Pinjam

## 1. Ringkasan Sistem
Aplikasi ini adalah sistem manajemen koperasi simpan pinjam berbasis web yang dibangun dengan bahasa Go (`golang`) menggunakan framework Gin. Sistem mencakup:
- Autentikasi dan otorisasi pengguna
- Role-based dashboard untuk `anggota`, `bendahara`, `ketua`, dan `admin`
- Pengelolaan anggota, simpanan, pinjaman, angsuran dan transaksi
- Fitur pengajuan keluar anggota dan konfirmasi oleh ketua/bendahara
- Upload file, notifikasi, dan laporan neraca
- Data disimpan di PostgreSQL

## 2. Struktur Umum Aplikasi

### 2.1 File utama
- `main.go`: Titik masuk aplikasi.
  - Memanggil `config.InitDB()` untuk menginisialisasi koneksi database.
  - Memanggil `routes.SetupRouter()` untuk menyiapkan router Gin.
  - Menjalankan server pada port `:8081`.

### 2.2 Paket penting
- `config/`: konfigurasi dan migrasi database otomatis.
- `routes/`: definisi semua rute HTTP dan middleware.
- `controllers/`: logika handling request dan render template.
- `models/`: struktur data domain dan representasi tabel database.
- `repository/`: operasi database untuk setiap entitas.
- `middleware/`: middleware otentikasi dan keamanan.
- `templates/`: file HTML untuk antarmuka.
- `static/`: CSS, JS, gambar, dan aset frontend.

## 3. Alur Inisialisasi dan Koneksi Database

### 3.1 `config.InitDB()`
- Membaca variabel lingkungan `DATABASE_URL`.
- Jika tidak tersedia, menggunakan fallback:
  - `postgres://postgres:postgres@localhost:5432/koperasi?sslmode=disable`
- Melakukan `db.Ping()` untuk memastikan koneksi berhasil.
- Memanggil fungsi migrasi ringan:
  - `ensureAngsuranTable`
  - `ensureImportHistoryTable`
  - `ensureReferensiPendaftaranTable`
  - `ensureSimpananWajibTables`
  - `updateAnggotaStatusConstraint`
  - `ensureDetailMetodePembayaranColumn`
  - `updateTransactionStatusConstraints`

### 3.2 Migrasi Otomatis
- Aplikasi membuat tabel/tabel tambahan secara otomatis bila belum ada.
- Ini mencegah kesalahan runtime karena tabel tidak ditemukan.
- Contoh: tabel `angsuran`, `import_history`, `referensi_pendaftaran`, `konfigurasi_simpanan_wajib`, `log_pemotongan_simpanan`.

## 4. Routing dan Middleware

### 4.1 Setup Router
- `routes.SetupRouter()` membuat engine Gin.
- Menetapkan trusted proxy lokal.
- Menyimpan session menggunakan cookie store `koperasisession`.
- Menonaktifkan cache untuk file statis.
- Memetakan static file dan favicon.
- Menyiapkan template functions seperti:
  - `add`, `div`, `mul`
  - `formatRupiah`
  - `json`, `toJson`
  - `now`, `hasPrefix`, `iterate`, `title`

### 4.2 Template Loading
- Semua file HTML dari folder `templates/` di-parse.
- Termasuk subfolder `admin`, `anggota`, `bendahara`, `ketua`, `layouts`, `utama`.

### 4.3 Rute Publik
- `/`, `/login`, `/register`
- `/logout`
- `/tentang/:slug`, `/pelayanan/:slug`
- `/hubungi-kami`
- API publik seperti `/api/jenis-simpanan`, `/api/jenis-angsuran`, `/api/metode-angsuran`

### 4.4 Rute Role-based
- `anggotaRoutes := router.Group("/anggota")`
  - Dashboard, profil, pesan, ganti password
  - Pengajuan pinjaman, simpanan, pengambilan, angsuran
  - Halaman informasi seperti sejarah, visi misi, struktur
  - Fitur pengajuan keluar

- `adminRoutes := router.Group("/admin")`
  - Dashboard admin
  - Import data referensi dan anggota
  - Manajemen anggota
  - Upload file, logo, background, tanda tangan
  - Transaksi simpanan/pinjaman dan riwayat
  - Login history dan laporan neraca
  - Pengaturan WA gateway dan konfigurasi pengguna

- `bendaharaRoutes := router.Group("/bendahara")`
  - Dashboard bendahara
  - Konfirmasi anggota dan konfirmasi transaksi
  - Lihat detail simpanan, pinjaman, angsuran, pengambilan
  - Import file potong gaji
  - Setting dan proses pemotongan simpanan wajib
  - Manajemen anggota keluar
  - Import data anggota & riwayat login

- `ketuaRoutes := router.Group("/ketua")`
  - Dashboard ketua
  - Konfirmasi anggota dan persetujuan anggota keluar
  - Konfirmasi transaksi dan laporan
  - Lihat data anggota
  - Upload bukti transfer gaji
  - Pengaturan profil ketua

## 5. Fitur Utama dan Alur Bisnis

### 5.1 Autentikasi dan Register
- Pengguna dapat mendaftar dan login.
- Role ditentukan saat registrasi atau oleh admin/bendahara.
- Pengguna non-authenticated diarahkan ke login.

### 5.2 Manajemen Anggota
- Dashboard anggota menampilkan ringkasan rekening simpanan, pinjaman, dan notifikasi.
- Admin/bendahara dapat menambahkan, melihat, mengedit, dan menghapus anggota.
- Fitur import excel untuk anggota dan referensi pendaftaran.

### 5.3 Simpanan dan Pinjaman
- Anggota dapat mengajukan simpanan dan pengambilan simpanan.
- Permohonan pinjaman dan angsuran dikelola oleh bendahara atau ketua.
- Transaksi direkam, kemudian dikonfirmasi oleh pihak berwenang.

### 5.4 Riwayat Transaksi
- Sistem menyediakan halaman riwayat untuk anggota, bendahara, ketua, dan admin.
- Meliputi riwayat login, riwayat transaksi, dan riwayat status anggota.

### 5.5 Fitur Pengajuan Keluar Anggota
- Anggota dapat mengajukan keluar dari koperasi melalui halaman profil.
- Data pengajuan dikirimkan ke ketua/bendahara untuk disetujui.
- Status anggota akan diperbarui menjadi `pending_keluar` lalu `keluar` jika disetujui.
- Data pengajuan dapat disimpan sebagai JSON di dalam tabel anggota.

### 5.6 Simpanan Wajib Otomatis
- Sistem menyediakan konfigurasi pemotongan otomatis untuk simpanan wajib.
- Bendahara dapat menyimpan konfigurasi dan memproses pemotongan secara manual atau otomatis.
- Riwayat pemotongan tercatat di tabel terpisah.

### 5.7 Laporan Neraca
- Admin dan ketua dapat mengakses laporan neraca.
- Laporan dapat diunduh dan ditampilkan melalui API internal.

## 6. Komponen Utama

### 6.1 `controllers/`
- Berisi handler untuk setiap halaman dan aksi.
- Contoh controller:
  - `anggota_controller.go`
  - `admin_controller.go`
  - `bendahara_controller.go`
  - `ketua_controller.go`
  - `auth_controller.go`
  - `halaman_controller.go`
- Controller bertanggung jawab mengumpulkan data dari repository, memvalidasi input, dan merender template.

### 6.2 `models/`
- Berisi definisi data dan struktur domain.
- Contoh model:
  - `anggota.go`
  - `transaksi.go`
  - `neraca.go`
  - `login_history.go`
  - `import_history.go`
  - `bukti_transfer_gaji.go`

### 6.3 `repository/`
- Berisi logika akses data ke PostgreSQL.
- Repository digunakan oleh controller untuk query dan update database.
- Memisahkan lapisan database dari logika presentasi.

### 6.4 `middleware/`
- `auth_middleware.go`: menangani autentikasi route dan hak akses.
- Melindungi rute role-based dan memastikan user login.

## 7. Setup dan Menjalankan Aplikasi

### 7.1 Persyaratan
- Go
- PostgreSQL
- `DATABASE_URL` (opsional)

### 7.2 Menjalankan Aplikasi
1. Pastikan database `koperasi` tersedia.
2. Set `DATABASE_URL` jika diperlukan, misalnya:
```bash
echo "DATABASE_URL=postgres://postgres:postgres@localhost:5432/koperasi?sslmode=disable" > .env
```
3. Jalankan aplikasi:
```bash
go run main.go
```
4. Buka browser ke:
```bash
http://localhost:8081
```

### 7.3 Catatan Database
- Jika `DATABASE_URL` tidak diset, aplikasi otomatis menggunakan koneksi lokal default.
- Tabel migrasi ringan dibuat otomatis saat startup.

## 8. Struktur Folder Template dan Aset

### 8.1 `templates/`
- `templates/admin/`: halaman admin.
- `templates/anggota/`: halaman anggota.
- `templates/bendahara/`: halaman bendahara.
- `templates/ketua/`: halaman ketua.
- `templates/layouts/`: template layout umum.
- `templates/utama/`: halaman publik.
- `templates/*.html`: halaman error, sukses, dan utama.

### 8.2 `static/`
- `static/css/`: stylesheet khusus.
- `static/js/`: file JavaScript.
- `static/images/`: aset gambar.
- `static/datatables/`: konfigurasi plugin DataTables.

## 9. Penutup
Sistem ini dirancang untuk memenuhi kebutuhan operasional koperasi simpan pinjam dengan otorisasi berbasis peran, otomatisasi pemotongan simpanan wajib, serta dokumentasi dan laporan transaksi. Alur aplikasi mengikuti prinsip MVC sederhana: `main -> config -> routes -> controllers -> repository -> models -> templates`.

> Gunakan dokumentasi ini sebagai panduan awal untuk memahami struktur, fitur, dan alur kerja aplikasi dari awal sampai akhir.
