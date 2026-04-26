# TODO: Fitur Upload Bukti Transfer Gaji oleh Ketua

## Ringkasan Fitur
1. **Ketua Dashboard**: Tambahkan menu untuk upload bukti transfer gaji bulanan dari bendahara universitas.
2. **Bendahara Konfirmasi Transaksi**: Setelah ketua upload bukti, barulah bendahara bisa melakukan import Excel potong gaji.

## Perubahan yang Diperlukan

### 1. Database Schema (database/koperasi.sql)
Tambahkan tabel baru `bukti_transfer_gaji`:
```sql
CREATE TABLE bukti_transfer_gaji (
    id SERIAL PRIMARY KEY,
    bulan INT NOT NULL,
    tahun INT NOT NULL,
    nama_file VARCHAR(255) NOT NULL,
    path_file VARCHAR(500) NOT NULL,
    diupload_oleh INT REFERENCES pengelola(id_pengelola),
    tgl_upload TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    status VARCHAR(20) DEFAULT 'pending' CHECK (status IN ('pending', 'approved', 'rejected')),
    catatan TEXT,
    UNIQUE(bulan, tahun)
);
```

### 2. Backend - Controllers

#### a. Ketua Controller (controllers/ketua_controller.go)
Tambahkan handler:
- `KetuaUploadBuktiTransferGaji()` - Menampilkan form upload bukti transfer gaji
- `KetuaUploadBuktiTransferGajiPost()` - Memproses upload file bukti transfer gaji
- `KetuaGetBuktiTransferGaji()` - Mengambil data bukti transfer gaji untuk ditampilkan di dashboard

#### b. Bendahara Controller (controllers/bendahara_controller.go)
Modifikasi:
- `BendaharaKonfirmasiTransaksi()` - Tambahkan pengecekan apakah bukti transfer gaji sudah diupload untuk bulan ini
- `BendaharaImportPotongGajiExcel()` - Tambahkan validasi bahwa bukti transfer gaji sudah diupload sebelum bisa import

### 3. Backend - Routes (routes/routes.go)
Tambahkan route baru untuk ketua:
```go
ketuaRoutes.GET("/bukti-transfer-gaji", controllers.KetuaUploadBuktiTransferGaji)
ketuaRoutes.POST("/bukti-transfer-gaji", controllers.KetuaUploadBuktiTransferGajiPost)
```

### 4. Frontend - Template Ketua Dashboard (templates/ketua/ketua_dashboard.html)
Tambahkan section baru untuk:
- Form upload bukti transfer gaji bulanan
- Tampilan status bukti transfer gaji yang sudah diupload
- Notifikasi jika bukti belum diupload

### 5. Frontend - Template Bendahara Konfirmasi Transaksi (templates/bendahara/bendahara_konfirmasi_transaksi.html)
Modifikasi section "Tambah Massal Potong Gaji via Excel":
- Tambahkan pengecekan apakah bukti transfer gaji sudah diupload
- Jika belum, tampilkan pesan warning dan disable form import
- Jika sudah, tampilkan info bukti transfer dan enable form import

### 6. Repository Layer (repository/)
Tambahkan fungsi-fungsi:
- `GetBuktiTransferGaji(bulan, tahun int)` - Mengambil data bukti transfer gaji
- `SaveBuktiTransferGaji(data)` - Menyimpan data bukti transfer gaji
- `CheckBuktiTransferGajiExists(bulan, tahun int)` - Mengecek apakah bukti sudah ada

## Implementasi Detail

### Flow Kerja:
1. Ketua login dan masuk ke dashboard
2. Ketua melihat section "Upload Bukti Transfer Gaji"
3. Ketua upload file bukti transfer (PDF/Image) untuk bulan tertentu
4. Sistem menyimpan file dan catatan di database
5. Bendahara masuk ke halaman konfirmasi transaksi
6. Sistem mengecek apakah bukti transfer untuk bulan ini sudah ada
7. Jika belum ada, form import Excel potong gaji di-disable dengan pesan warning
8. Jika sudah ada, form import Excel aktif dan bisa digunakan

### Catatan:
- File yang diupload bisa berupa PDF, JPG, PNG
- Maksimal ukuran file 5MB
- Validasi bulan dan tahun harus sesuai dengan periode berjalan
- Hanya ketua yang bisa upload bukti transfer gaji
- Bendahara hanya bisa melihat info bukti transfer, tidak bisa upload

