# TODO Fix Notifikasi WA Ketua & Bendahara

## Progress

- [x] Analisis akar masalah
- [x] Step 1: Fix `repository.GetBendahara()` - hapus filter no_telepon <> ''
- [ ] Step 2: Verify & Fix `sendKetuaWhatsAppNotification` fallback ke `getKetuaWhatsAppPhone()`
- [ ] Step 3: Verify ketua WA notifikasi triggers di semua flow yang relevan
- [ ] Step 4: Rebuild & test

## Analisis Masalah

### Masalah 1: Bendahara tidak mendapatkan notifikasi WA (anggota_controller.go)
**Akar masalah:** `repository.GetBendahara()` mem-filter `AND TRIM(COALESCE(no_telepon, '')) <> ''`
- Jika bendahara di tabel `pengelola` tidak punya nomor telepon → fungsi mengembalikan `sql.ErrNoRows`
- Di `anggota_controller.go`, 4 call site (pinjaman, simpanan, angsuran, pengambilan) wrap dalam `if err == nil` → blok tidak dieksekusi → `sendBendaharaWhatsAppNotification()` tidak pernah dipanggil
- Akibatnya: notifikasi WA ke bendahara **sama sekali tidak terkirim** meskipun ada transaksi baru

### Masalah 2: Ketua tidak mendapat notifikasi (umum)
**Akar masalah:** 
- Notifikasi ke ketua (WA) hanya dikirim saat Bendahara melakukan **confirm** transaksi (`BendaharaKonfirmasiTransaksiPost`)
- Tidak ada notifikasi ke ketua saat Anggota mengajukan transaksi baru (pinjaman/simpanan/angsuran/pengambilan)
- Ini mungkin yang dimaksud user: ketua "tidak tahu" ada data masuk sampai di-check manual

### Masalah 3: Anggota mendapat notifikasi yang seharusnya untuk Ketua
**Akar masalah:**
- Perlu verifikasi apakah ada kebocoran notifikasi WA atau notifikasi in-app yang salah arah

## Fix yang Dilakukan

### Fix 1: `repository/pengelola_repository.go` ✅
```sql
-- BEFORE (error jika bendahara tidak punya no_telp):
WHERE LOWER(TRIM(level)) = 'bendahara'
  AND TRIM(COALESCE(no_telepon, '')) <> ''

-- AFTER (selalu ambil bendahara yang ada):
WHERE LOWER(TRIM(level)) = 'bendahara'
```
- Impact: Sekarang `GetBendahara()` mengembalikan data meskipun nomor telepon kosong
- Fallback: `sendBendaharaWhatsAppNotification()` akan menggunakan `getBendaharaWhatsAppPhone()` dari tabel pengaturan (`wa_bendahara_phone`)

## Fix Selanjutnya (menunggu approval)

### Fix 2: Tambah notifikasi WA ke Ketua saat Anggota submit transaksi baru
- Lokasi: `anggota_controller.go` (4 call site)
- Action: Tambah pemanggilan `sendKetuaWhatsAppTransactionNotification()` saat anggota berhasil mengajukan simpanan/angsuran/pinjaman/pengambilan
- Tujuannya: Ketua mendapat notifikasi real-time bahwa ada transaksi baru menunggu konfirmasi bendahara

### Fix 3: Verifikasi notifikasi in-app/admin badge
- Perlu tambah counter/badge di navbar admin & ketua untuk calon anggota pending
- Ini terpisah dari WA notifikasi
