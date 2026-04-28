# TODO FIX

## Langkah 1: Tambah endpoint backend untuk cek data pending cocok
- [ ] Controller `BendaharaCatatSimpanan`: sebelum insert, panggil `repository.GetPendingSimpananByCriteria` dan pastikan hasil > 0. Jika 0, return error.
- [ ] Controller `BendaharaCatatAngsuran`: sebelum insert, panggil `repository.GetPendingAngsuranByCriteria` dan pastikan hasil > 0. Jika 0, return error.

## Langkah 2: Update frontend JS untuk validasi sebelum submit
- [ ] `static/js/bendahara_konfirmasi_transaksi.js`: validasi form simpanan tunai → fetch/cek data pending cocok via AJAX ke endpoint baru, jika tidak cocok tampilkan error.
- [ ] Validasi form angsuran tunai → fetch/cek data pending cocok via AJAX, jika tidak cocok tampilkan error.

## Langkah 3: Update template HTML untuk pesan informasi
- [ ] `templates/bendahara/bendahara_konfirmasi_transaksi.html`: tambah alert informasi bahwa entri manual tunai hanya untuk data yang sudah ada di Pending.

