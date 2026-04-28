# TODO Eksekusi Fix Notifikasi WA

## Status
- [x] Step 1: auth_controller.go - getBendaharaWhatsAppPhone() sudah dibuat, sendBendaharaWhatsAppNotification() sudah punya fallback
- [x] Step 2: anggota_controller.go - 4 call sites (pinjaman, simpanan, angsuran, penarikan) sudah otomatis mendapat fallback karena sendBendaharaWhatsAppNotification handle empty phone
- [x] Step 3: repository/pengelola_repository.go - GetBendahara() diperbaiki, tidak lagi filter no_telepon <> ''
- [x] Step 4: Build & Test - go build berhasil (koperasi_wa_fix_test.exe)

## Ringkasan Perubahan

### 1. repository/pengelola_repository.go
- **Masalah**: `GetBendahara()` mem-filter `AND TRIM(COALESCE(no_telepon, '')) <> ''`, menyebabkan `sql.ErrNoRows` ketika bendahara tidak punya nomor telepon di tabel pengelola.
- **Fix**: Hapus filter no_telepon dari WHERE clause, sehingga GetBendahara() selalu return row bendahara aktif meski nomor kosong.
- **Efek**: `anggota_controller.go` tidak lagi masuk ke branch `else { log.Printf("bendahara tidak ditemukan...") }`, melainkan masuk ke branch `if err == nil` dan memanggil `sendBendaharaWhatsAppNotification()`.

### 2. controllers/auth_controller.go
- **Masalah**: `sendBendaharaWhatsAppNotification()` tidak punya fallback ketika `rawBendaharaPhone` kosong.
- **Fix**: 
  - Tambah `getBendaharaWhatsAppPhone()` (mirip `getKetuaWhatsAppPhone()`) yang mencari nomor dari:
    1. Pengaturan `wa_bendahara_phone` / `nomor_wa_bendahara` / `telepon_bendahara`
    2. Profil pengelola level bendahara aktif
  - Update `sendBendaharaWhatsAppNotification()` untuk fallback ke `getBendaharaWhatsAppPhone()` ketika `rawBendaharaPhone` kosong.

### 3. controllers/anggota_controller.go
- **Tidak perlu perubahan** karena `sendBendaharaWhatsAppNotification(bendahara.NoTelepon, ...)` sudah otomatis menggunakan fallback internal.

## Cara Kerja Sekarang

### Flow Notifikasi WA Bendahara (dari anggota):
1. Anggota ajukan pinjaman/simpanan/angsuran/penarikan
2. `repository.GetBendahara()` → return bendahara row (meski no_telepon kosong)
3. `sendBendaharaWhatsAppNotification(bendahara.NoTelepon, ...)` 
   - Jika `bendahara.NoTelepon` kosong → fallback ke `getBendaharaWhatsAppPhone()`
   - `getBendaharaWhatsAppPhone()` → cari di pengaturan `wa_bendahara_phone` → jika kosong, cari di profil pengelola bendahara aktif
4. Kirim WA ke nomor yang ditemukan

### Flow Notifikasi WA Ketua (saat registrasi):
1. `Register()` → ambil `ketuaTelepon` dari halaman `hubungi_kami`
2. `sendKetuaWhatsAppNotification(ketuaTelepon, ...)` 
   - Jika `ketuaTelepon` kosong → fallback ke `getKetuaWhatsAppPhone()`
   - `getKetuaWhatsAppPhone()` → cari di pengaturan `wa_ketua_phone` → jika kosong, cari di halaman `hubungi_kami` → jika kosong, cari di profil pengelola ketua aktif
3. Kirim WA ke nomor yang ditemukan

## Build Status
✅ `go build -o koperasi_wa_fix_test.exe .` berhasil
