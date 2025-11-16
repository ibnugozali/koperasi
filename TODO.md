# TODO: Perbaikan Sistem Koperasi Simpan Pinjam

## ✅ Selesai

### 1. Perbaikan Database Schema (database/koperasi.sql)
- [x] Ubah `id_anggota` dari SERIAL menjadi VARCHAR(13)
- [x] Tambahkan kolom `unit_kerja` VARCHAR(2)
- [x] Tambahkan kolom `fakultas_code` VARCHAR(2)
- [x] Tambahkan kolom `tahun` VARCHAR(4)
- [x] Tambahkan kolom `nomor_urut` VARCHAR(4)
- [x] Update foreign key references di tabel detail, pinjaman, angsuran, pesan

### 2. Perbaikan Models (models/anggota.go)
- [x] Ubah `IDAnggota` dari int menjadi string
- [x] Tambahkan field `UnitKerja`, `FakultasCode`, `Tahun`, `NomorUrut`

### 3. Perbaikan Repository (repository/anggota_repository.go)
- [x] Update `CreateAnggota()` untuk insert 14 parameter termasuk id_anggota
- [x] Update `GetAnggotaByID()` untuk handle NULL values dengan COALESCE
- [x] Update semua fungsi untuk menggunakan string ID
- [x] Update `GetPendingAnggota()` untuk include unit_kerja dan fakultas_code

### 4. Perbaikan Controllers
- [x] Update semua controller untuk menggunakan string ID
- [x] Fix type mismatches di admin_controller.go, bendahara_controller.go, halaman_controller.go, ketua_controller.go
- [x] Update `ConfirmMembership()` di admin_controller.go untuk generate ID dengan format yang benar

### 5. Perbaikan Templates (templates/utama/register.html)
- [x] Ubah field address dari "Provinsi" menjadi "Alamat"

### 6. Perbaikan Auth Controller (controllers/auth_controller.go)
- [x] Update `Register()` untuk generate temporary ID dengan format "TEMP" + username

## 📋 Format ID Anggota
ID anggota dihasilkan dengan format: `unit_kerja + fakultas_code + tahun + nomor_urut`

Contoh: `010120250001`
- `01`: unit_kerja (01=dosen, 02=karyawan/staff, 03=mahasiswa)
- `01`: fakultas_code (01=FAI, 02=FE, 03=FH, 04=FISIP, 05=FKIP, 06=FKM, 07=FAPERTA, 08=FT, 09=Rektorat)
- `2025`: tahun konfirmasi
- `0001`: nomor urut dalam unit_kerja + fakultas_code + tahun tersebut

## 🔄 Mekanisme Generate ID
1. Saat registrasi: ID sementara "TEMP" + username
2. Saat konfirmasi admin: Generate ID final berdasarkan unit_kerja, fakultas_code, tahun, dan nomor urut
3. Nomor urut dihitung berdasarkan jumlah anggota aktif dengan kombinasi unit_kerja + fakultas_code + tahun yang sama

## 🧪 Testing Status
- [x] Aplikasi berhasil dijalankan (go run main.go)
- [x] Database connection berhasil
- [x] Server berjalan di port 8081
- [x] Error handling untuk NULL values sudah diperbaiki
- [ ] Perlu testing konfirmasi anggota untuk memastikan ID generate dengan benar
- [ ] Perlu testing registrasi anggota baru
- [ ] Perlu testing login dan dashboard

## 📝 Catatan Penting
- Semua ID sekarang menggunakan string, bukan integer
- Foreign key references sudah diupdate untuk menggunakan VARCHAR
- Error handling sudah diperbaiki untuk NULL values
- Sistem siap untuk generate ID anggota dengan format yang diminta
