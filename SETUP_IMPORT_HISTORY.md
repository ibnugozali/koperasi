# Setup Fitur Riwayat Import Anggota

## 📋 Langkah-langkah Setup

### 1. Buat Database Baru atau Update Database yang Sudah Ada

**Opsi A: Database Baru (Recommended)**
Jika Anda membuat database baru dari awal, cukup jalankan file `database/koperasi.sql` yang sudah include tabel `import_history`:

```bash
# Buat database baru
mysql -u root -p -e "CREATE DATABASE IF NOT EXISTS koperasi"

# Import semua tabel termasuk import_history
mysql -u root -p koperasi < database/koperasi.sql
```

**Opsi B: Database yang Sudah Ada**
Jika database sudah ada dan hanya ingin menambahkan tabel `import_history`:

```sql
-- Tabel riwayat import anggota
DROP TABLE IF EXISTS import_history CASCADE;
CREATE TABLE import_history (
    id_import VARCHAR(36) PRIMARY KEY,
    id_pengelola INT NOT NULL REFERENCES pengelola(id_pengelola) ON DELETE CASCADE,
    username VARCHAR(100) NOT NULL,
    file_name VARCHAR(255) NOT NULL,
    total_data INT NOT NULL DEFAULT 0,
    success_count INT NOT NULL DEFAULT 0,
    failed_count INT NOT NULL DEFAULT 0,
    imported_data TEXT,
    parse_errors TEXT,
    tanggal_import TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Index untuk performa
CREATE INDEX IF NOT EXISTS idx_import_history_pengelola ON import_history(id_pengelola);
CREATE INDEX IF NOT EXISTS idx_import_history_tanggal ON import_history(tanggal_import);
```

### 2. Cara Menjalankan SQL

**A. Menggunakan phpMyAdmin:**
1. Buka phpMyAdmin
2. Pilih database `koperasi`
3. Klik tab "SQL"
4. Copy paste SQL di atas
5. Klik "Go"

**B. Menggunakan MySQL Command Line:**
```bash
# Untuk database baru
mysql -u root -p koperasi < database/koperasi.sql

# Atau masuk ke MySQL dan copy paste SQL manual
mysql -u root -p koperasi
```

**C. Menggunakan HeidiSQL / DBeaver / MySQL Workbench:**
1. Buka aplikasi database client Anda
2. Connect ke database `koperasi`
3. Buka SQL editor
4. Copy paste dan jalankan SQL di atas

### 3. Restart Server

Setelah tabel berhasil dibuat, restart aplikasi Go:
```bash
go run main.go
```

## ✨ Fitur yang Ditambahkan

### 1. **Penyimpanan Riwayat Import**
- Setiap kali upload file import berhasil, data akan disimpan ke database
- Data yang disimpan meliputi:
  - Nama file
  - Total data
  - Jumlah berhasil/gagal
  - Data anggota yang berhasil diimport
  - Error yang terjadi
  - Timestamp import

### 2. **Tampilan Data Setelah Refresh**
- Saat buka halaman import anggota, akan menampilkan hasil import terakhir
- Data tetap tampil meskipun:
  - Refresh halaman (F5)
  - Logout dan login kembali
  - Tutup dan buka browser lagi

### 3. **Riwayat Personal**
- Setiap pengelola (bendahara) memiliki riwayat import sendiri
- Tidak akan melihat riwayat import pengelola lain

## 📊 Struktur Data yang Disimpan

```json
{
  "id_import": "uuid",
  "id_pengelola": 1,
  "username": "nama-user",
  "file_name": "template_anggota.xlsx",
  "total_data": 10,
  "success_count": 8,
  "failed_count": 2,
  "imported_data": "[{...data anggota...}]",
  "parse_errors": "[...error messages...]",
  "tanggal_import": "2025-12-02 10:30:00"
}
```

## 🔧 File yang Dimodifikasi/Dibuat

1. **Database**
   - ✅ `database/koperasi.sql` - Ditambahkan tabel `import_history`
   - ❌ `database/add_import_history_table.sql` - **DIHAPUS** (sudah digabung ke koperasi.sql)

2. **Models**
   - ✅ `models/import_history.go` (NEW)

3. **Repository**
   - ✅ `repository/import_history_repository.go` (NEW)

4. **Controllers**
   - ✅ `controllers/bendahara_controller.go`
     - `BendaharaImportAnggotaPage()` - Menampilkan riwayat
     - `BendaharaImportAnggota()` - Menyimpan riwayat

5. **Templates**
   - ✅ `templates/bendahara/bendahara_import_anggota.html`
     - Menambahkan script untuk load data dari server

## 🎯 Cara Menggunakan

1. Login sebagai bendahara
2. Buka menu "Import Data Anggota"
3. Upload file Excel dengan data anggota
4. Setelah upload berhasil, data akan muncul di halaman
5. **Refresh halaman** atau **logout dan login lagi** → Data tetap muncul!

## 🔍 Verifikasi

Untuk memastikan tabel berhasil dibuat, jalankan query berikut:

```sql
SHOW TABLES LIKE 'import_history';
```

Atau cek struktur tabel:

```sql
DESCRIBE import_history;
```

## ⚠️ Catatan Penting

- Pastikan tabel `pengelola` sudah ada sebelum membuat tabel `import_history` (karena ada foreign key)
- Data riwayat akan otomatis terhapus jika pengelola dihapus (CASCADE)
- Hanya menyimpan 1 riwayat terbaru per pengelola saat halaman dibuka (bisa dimodifikasi untuk menyimpan lebih banyak)
