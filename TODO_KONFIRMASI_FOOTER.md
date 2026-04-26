# TODO: Perbaiki Footer Halaman Bendahara Konfirmasi Transaksi

## Masalah
Footer di halaman `/bendahara/konfirmasi-transaksi` tidak rapi karena struktur HTML rusak.

## Penyebab Utama
1. **Tag penutup Page Header tidak ada**: Tag pembuka `<div class="row card-row"><div class="col-12">` (Halaman Judul/Header) tidak ditutup sebelum Kartu 1 dimulai, sehingga semua kartu masuk ke dalamnya.
2. **Tag penutup setiap kartu hilang**: Kartu 1-4 masing-masing memiliki `<div class="row card-row"><div class="col-12"><div class="content-card">` tetapi tidak ada tag penutup `</div>` untuk `col-12` dan `row card-row`.
3. **Div di Kartu 3 tidak ditutup**: Tag `<div class="row g-2">` di dalam form Import Excel tidak ditutup.
4. **Tag penutup ekstra terselip di akhir**: Ada `</div>` di akhir yang mestinya untuk menutup `container-fluid` dan `main-content`, tetapi malah letaknya salah akibat masalah nesting di atas.

## Rencana Perbaikan
File: `templates/bendahara/bendahara_konfirmasi_transaksi.html`

1. Tutup tag Page Header sebelum Kartu 1 dimulai.
2. Tambahkan tag `</div>` setelah tiap kartu setelah `content-card`.
3. Tutup tag `<div class="row g-2">` di Kartu 3.
4. Pastikan struktur akhir benar menutup `container-fluid` dan `main-content`.
5. Biarkan posisi footer template seperti semula (setelah main-content ditutup).

## Status
✅ **SELESAI** - Semua perbaikan telah diimplementasikan pada file `templates/bendahara/bendahara_konfirmasi_transaksi.html`

## Perubahan yang Dilakukan
1. ✅ Menutup tag Page Header (`</div></div>`) sebelum Card 1 dimulai.
2. ✅ Menambahkan tag penutup `</div></div>` untuk `col-12` dan `row card-row` pada Card 1, 2, dan 4.
3. ✅ Menutup tag `<div class="row g-2">` di Card 3 sebelum tag `</form>`.
4. ✅ Menambahkan tag penutup `</div></div>` untuk `col-12` dan `row card-row` pada Card 3.
5. ✅ Memperbaiki tag penutup akhir: `container-fluid` dan `main-content` ditutup dengan benar sebelum footer template.

## Hasil yang Diharapkan
- Layout flex yang benar: footer berada di bawah viewport ketika konten pendek.
- Margin sidebar footer rata (footer menyesuaikan `margin-left: 240px` dari CSS).
- Struktur HTML valid.
- Footer tampil rapi di halaman `/bendahara/konfirmasi-transaksi`.
