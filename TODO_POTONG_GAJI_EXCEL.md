# TODO: Tambah Massal via Excel untuk Potong Gaji di Bendahara Konfirmasi Transaksi

## Status: ✅ SELESAI

## Ringkasan
Fitur "Tambah Massal via Excel" telah ditambahkan ke halaman `/bendahara/konfirmasi-transaksi` untuk memproses transaksi potong gaji secara massal melalui file Excel.

## Perubahan yang Dilakukan

### 1. UI Template ✅
- **File**: `templates/bendahara/bendahara_konfirmasi_transaksi.html`
- Menambahkan card "Tambah Massal Potong Gaji via Excel" di bawah daftar pending transaksi
- Form upload dengan `id="potongGajiImportForm"` dan tombol `id="potongGajiImportBtn"`
- Area notifikasi hasil `id="potongGajiAlert"`

### 2. JavaScript Handler ✅
- **File**: `static/js/bendahara_konfirmasi_transaksi.js`
- Menambahkan event listener untuk form upload Excel
- Mengirim data via AJAX ke `/bendahara/konfirmasi-transaksi/import-potong-gaji`
- Menampilkan hasil import (berhasil/gagal) dengan detail error per baris
- Auto-reload halaman setelah berhasil
- Menambahkan fungsi `escapeHtml()` untuk sanitasi output

### 3. Controller ✅
- **File**: `controllers/bendahara_controller.go`
- Menambahkan fungsi `BendaharaImportPotongGajiExcel`
- Fitur:
  - Upload file Excel (.xlsx/.xls)
  - Deteksi header secara dinamis
  - Mendukung **Simpanan** (metode pembayaran = `potong_gaji`) dan **Angsuran**
  - Validasi ID anggota dan jumlah
  - Auto-hitung `total_simpanan` untuk simpanan
  - Auto-hitung `sisa_pinjaman` untuk angsuran (ambil pinjaman aktif)
  - Status langsung `confirmed`
  - Return JSON dengan ringkasan import

### 4. Route ✅
- **File**: `routes/routes.go`
- Menambahkan: `bendaharaRoutes.POST("/konfirmasi-transaksi/import-potong-gaji", controllers.BendaharaImportPotongGajiExcel)`

### 5. Verifikasi ✅
- Kompilasi berhasil tanpa error (`go build -o test_build.exe .`)
- File `test_build.exe` telah dibuat

## Format Excel yang Didukung

| ID Anggota | Nama Anggota | Jenis Transaksi | Jenis Simpanan | Jumlah |
|------------|--------------|-----------------|----------------|--------|
| 0101250001 | Budi Santoso | Simpanan        | Wajib          | 50000  |
| 0101250002 | Ani Wijaya   | Simpanan        | Sukarela       | 25000  |
| 0101250001 | Budi Santoso | Angsuran        | -              | 100000 |

**Catatan:**
- Jika Jenis Transaksi kosong, default = `Simpanan`
- Jika Jenis Simpanan kosong, default = `Wajib`
- Untuk Angsuran, ID Pinjaman aktif akan dicari otomatis

