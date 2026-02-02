# Fitur Pengajuan Keluar dari Koperasi

## 📋 Deskripsi
Fitur ini memungkinkan anggota untuk mengajukan keluar dari koperasi dengan mengisi form pengembalian simpanan. Pengajuan akan masuk ke halaman konfirmasi Ketua untuk disetujui.

## 🗄️ Setup Database

### Jika Database Belum Dibuat (Fresh Install)
File `database/koperasi.sql` sudah diperbarui dengan kolom `data_keluar`. Jalankan:
```bash
psql -U postgres -d koperasi < database/koperasi.sql
```

### Jika Database Sudah Ada (Update)
Jalankan file SQL untuk menambahkan kolom `data_keluar`:
```bash
psql -U postgres -d koperasi -f database/update_add_data_keluar.sql
```

Atau manual via psql:
```sql
ALTER TABLE anggota ADD COLUMN data_keluar JSONB DEFAULT NULL;
```

## 🎯 Cara Menggunakan

### Untuk Anggota:
1. Login sebagai anggota
2. Buka menu **Profil Saya**
3. Klik tombol **"Keluar dari Koperasi"** (tombol merah)
4. Modal akan muncul dengan form:
   - **Simpanan Wajib**: Jumlah simpanan wajib yang akan dikembalikan
   - **Simpanan Lainnya**: Total simpanan sukarela, hari raya, umroh/haji, dan qurban yang akan dikembalikan
   - **Biaya Admin**: Biaya administrasi (jika ada)
   - **Alasan Keluar**: Wajib diisi
5. Klik **"Ajukan Keluar"**
6. Status akan berubah menjadi `pending_keluar`
7. Menunggu persetujuan dari Ketua

### Untuk Ketua:
1. Login sebagai Ketua
2. Buka menu **Konfirmasi Anggota**
3. Scroll ke section **"Konfirmasi Anggota Keluar"**
4. Lihat daftar anggota yang mengajukan keluar dengan data:
   - Nama anggota
   - ID Anggota
   - NIK
   - Telepon
   - Fakultas
   - Tanggal Gabung
5. Klik **"Lihat Detail"** untuk melihat detail lengkap termasuk data pengembalian simpanan
6. Klik **"Setujui"** untuk menyetujui atau **"Tolak"** untuk menolak pengajuan

## 📊 Struktur Data

### Kolom Tabel Anggota
```sql
status_anggota VARCHAR(50)  -- Status: 'pending_keluar', 'keluar', dll
data_keluar JSONB           -- Data pengajuan dalam format JSON
```

### Format JSON data_keluar
```json
{
  "simpanan_wajib": 500000,
  "simpanan_lainnya": 300000,
  "biaya_admin": 50000,
  "alasan": "Pindah tugas ke kota lain",
  "tanggal_pengajuan": "2026-02-02T10:30:00Z"
}
```

## 🔍 Query Berguna

### Melihat Semua Pengajuan Keluar
```sql
SELECT 
    id_anggota, 
    nama_anggota, 
    status_anggota,
    data_keluar->>'simpanan_wajib' as simpanan_wajib,
    data_keluar->>'simpanan_lainnya' as simpanan_lainnya,
    data_keluar->>'biaya_admin' as biaya_admin,
    data_keluar->>'alasan' as alasan,
    data_keluar->>'tanggal_pengajuan' as tanggal_pengajuan
FROM anggota 
WHERE status_anggota = 'pending_keluar';
```

### Menghitung Total Pengajuan Keluar
```sql
SELECT 
    COUNT(*) as total_pengajuan,
    SUM((data_keluar->>'simpanan_wajib')::numeric) as total_simpanan_wajib,
    SUM((data_keluar->>'simpanan_lainnya')::numeric) as total_simpanan_lainnya
FROM anggota 
WHERE status_anggota = 'pending_keluar';
```

## ⚙️ File yang Dimodifikasi

1. **templates/anggota/anggota_profil.html**
   - Modal form pengajuan keluar yang lengkap
   - Validasi JavaScript
   - Alert sukses/error

2. **controllers/anggota_controller.go**
   - Fungsi `KeluarKoperasi` untuk handle pengajuan

3. **routes/routes.go**
   - Update fungsi `add` untuk mendukung float64
   - Route `/anggota/keluar` sudah ada

4. **database/koperasi.sql**
   - Tambah kolom `data_keluar JSONB`

5. **database/update_add_data_keluar.sql**
   - Script ALTER TABLE untuk database yang sudah ada

## 🎨 Catatan Penting

- **Simpanan Pokok** tidak dikembalikan sesuai ketentuan koperasi
- Pengajuan harus disetujui Ketua terlebih dahulu
- Data disimpan dalam format JSON untuk fleksibilitas
- Anggota dapat membatalkan pengajuan sebelum disetujui (dengan menghubungi admin)
- Setelah disetujui, status akan diubah menjadi `keluar` dan proses pengembalian simpanan dilakukan

## 🐛 Troubleshooting

### Error: column "data_keluar" does not exist
Jalankan file SQL update:
```bash
psql -U postgres -d koperasi -f database/update_add_data_keluar.sql
```

### Modal tidak muncul
Pastikan Bootstrap JS sudah di-load dengan benar di halaman anggota_profil.html

### Validasi form gagal
Periksa console browser untuk error JavaScript dan pastikan semua input field memiliki name yang benar
