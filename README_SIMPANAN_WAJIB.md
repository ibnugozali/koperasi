# Setup Tabel Simpanan Wajib Otomatis

## ✅ Setup Otomatis (Recommended)

Tabel akan **dibuat otomatis** saat Anda menjalankan aplikasi!

```bash
go run main.go
```

Aplikasi akan secara otomatis:
1. Membuat tabel `konfigurasi_simpanan_wajib`
2. Membuat tabel `log_pemotongan_simpanan`
3. Menambahkan data konfigurasi default

Periksa log console, Anda akan melihat:
```
✓ Tabel simpanan wajib siap digunakan
✓ Data default konfigurasi simpanan wajib ditambahkan
```

## Manual Setup (Opsional)

Jika Anda ingin membuat tabel secara manual:

### Pilihan 1: Menggunakan psql (Command Line)
```bash
psql -U postgres -d koperasi -f database/add_simpanan_wajib_tables.sql
```

### Pilihan 2: Menggunakan pgAdmin
1. Buka pgAdmin
2. Connect ke database `koperasi`
3. Klik kanan pada database > Query Tool
4. Buka file `database/add_simpanan_wajib_tables.sql`
5. Klik Execute (F5)

### Pilihan 3: Menggunakan DBeaver atau Database Client lainnya
1. Connect ke database
2. Buka SQL Editor
3. Copy paste isi file `add_simpanan_wajib_tables.sql`
4. Execute script

## Verifikasi

Setelah menjalankan script, verifikasi bahwa tabel sudah dibuat:

```sql
-- Check if tables exist
SELECT table_name 
FROM information_schema.tables 
WHERE table_schema = 'public' 
AND table_name IN ('konfigurasi_simpanan_wajib', 'log_pemotongan_simpanan');

-- Check default data
SELECT * FROM konfigurasi_simpanan_wajib;
```

Anda seharusnya melihat:
- 2 tabel baru: `konfigurasi_simpanan_wajib` dan `log_pemotongan_simpanan`
- 1 baris data default di tabel `konfigurasi_simpanan_wajib`

## Troubleshooting

Jika ada error:
1. Pastikan Anda sudah login sebagai user yang memiliki akses CREATE TABLE
2. Pastikan database `koperasi` sudah ada
3. Pastikan tabel `anggota` sudah ada (karena ada foreign key reference)

## Setelah Setup

Setelah tabel berhasil dibuat (otomatis atau manual):

1. **Jalankan aplikasi**: `go run main.go`
2. **Login sebagai bendahara**
3. **Buka menu**: Manajemen Anggota > Setting Simpanan Wajib
4. **Atur konfigurasi**:
   - Aktifkan toggle pemotongan otomatis
   - Pilih tanggal pemotongan (1-31)
   - Pilih metode: Persentase atau Nominal Tetap
   - Simpan konfigurasi

## Fitur Setting Simpanan Wajib

### Konfigurasi yang Tersedia:
- **Status Aktif/Nonaktif**: Toggle untuk mengaktifkan/menonaktifkan fitur
- **Tanggal Pemotongan**: Tanggal setiap bulan untuk melakukan pemotongan (1-31)
- **Metode Pemotongan**:
  - **Persentase**: Potong % dari gaji bulanan (contoh: 5%)
  - **Nominal Tetap**: Potong jumlah tetap setiap bulan (contoh: Rp 50.000)

### Cara Kerja:
1. Sistem akan otomatis memotong gaji anggota pada tanggal yang ditentukan
2. Hasil pemotongan masuk ke simpanan wajib dengan status "confirmed"
3. Log pemotongan disimpan untuk tracking
4. Anggota dengan gaji 0 (mahasiswa) tidak akan dipotong
5. Tidak akan memotong dua kali untuk bulan yang sama

### Proses Manual:
Jika ingin melakukan pemotongan secara manual (tidak menunggu tanggal):
1. Buka halaman Setting Simpanan Wajib
2. Klik tombol "Jalankan Proses Pemotongan"
3. Sistem akan memproses pemotongan untuk semua anggota yang belum dipotong bulan ini

## Struktur Tabel

### konfigurasi_simpanan_wajib
Menyimpan pengaturan pemotongan:
- `id`: Primary key
- `tanggal_potong`: Tanggal pemotongan (1-31)
- `persentase_potong`: Persentase pemotongan (0-100)
- `nominal_tetap`: Nominal tetap jika menggunakan metode nominal
- `tipe_pemotongan`: 'persentase' atau 'nominal_tetap'
- `status_aktif`: true/false untuk mengaktifkan fitur
- `created_at`, `updated_at`: Timestamp

### log_pemotongan_simpanan
Menyimpan riwayat pemotongan:
- `id_log`: Primary key
- `id_anggota`: ID anggota yang dipotong
- `bulan`, `tahun`: Periode pemotongan
- `gaji_bulanan`: Gaji pada saat dipotong
- `jumlah_potong`: Jumlah yang dipotong
- `tgl_proses`: Timestamp proses
- `status`: 'berhasil' atau 'gagal'
- `keterangan`: Detail tambahan
