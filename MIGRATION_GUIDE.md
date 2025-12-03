# Panduan Migration: Ubah Unit Kerja dari Kode ke Nama Lengkap

## Masalah
Database error: `pq: value too long for type character varying(2)`

Kolom `unit_kerja` di tabel `anggota` masih menggunakan `VARCHAR(2)` yang hanya bisa menyimpan kode 2 digit (01, 02, 03). Sekarang kita perlu menyimpan nama lengkap (Dosen, Staff, Mahasiswa).

## Solusi

### Opsi 1: Jalankan Migration Script (RECOMMENDED)

1. Buka PostgreSQL command line atau pgAdmin
2. Connect ke database `koperasi_db`
3. Jalankan file: `database/migration_unit_kerja.sql`

**Atau via command line:**
```bash
psql -U postgres -d koperasi_db -f database/migration_unit_kerja.sql
```

### Opsi 2: Manual Query (jika Opsi 1 gagal)

Jalankan query berikut satu per satu di PostgreSQL:

```sql
-- 1. Backup data lama
ALTER TABLE anggota ADD COLUMN IF NOT EXISTS unit_kerja_backup VARCHAR(2);
UPDATE anggota SET unit_kerja_backup = unit_kerja WHERE unit_kerja IS NOT NULL;

-- 2. Ubah tipe data
ALTER TABLE anggota ALTER COLUMN unit_kerja TYPE VARCHAR(50);

-- 3. Convert kode ke nama lengkap
UPDATE anggota 
SET unit_kerja = CASE 
    WHEN unit_kerja_backup = '01' THEN 'Dosen'
    WHEN unit_kerja_backup = '02' THEN 'Staff'
    WHEN unit_kerja_backup = '03' THEN 'Mahasiswa'
    ELSE unit_kerja
END
WHERE unit_kerja_backup IS NOT NULL;

-- 4. Verifikasi
SELECT id_anggota, nama_anggota, unit_kerja FROM anggota LIMIT 10;
```

## Perubahan yang Sudah Dilakukan

✅ **Backend (controllers/bendahara_controller.go):**
- Menghapus fungsi `mapUnitKerja` yang mengkonversi nama ke kode
- Unit kerja sekarang disimpan langsung sebagai nama lengkap

✅ **Database Schema (database/koperasi.sql):**
- Kolom `unit_kerja` diubah dari `VARCHAR(2)` ke `VARCHAR(50)`

⏳ **Database Migration:**
- Perlu dijalankan untuk database yang sudah ada

## Setelah Migration

Setelah migration berhasil, aplikasi akan:
- Menyimpan "Dosen", "Staff", "Mahasiswa" di kolom unit_kerja
- Tidak lagi menggunakan kode 01, 02, 03
- Data lama (jika ada) akan dikonversi otomatis

## Testing

Setelah migration, coba:
1. Import file Excel baru dengan unit kerja "Dosen", "Staff", atau "Mahasiswa"
2. Pastikan tidak ada error "value too long"
3. Verifikasi data tersimpan dengan benar
