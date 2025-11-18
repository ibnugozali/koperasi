# TODO: Perbaiki Logika Ajukan Pinjaman

## 1. Update Controller AjukanPinjamanPost
- [ ] Tambahkan logika menghitung total simpanan menggunakan GetSaldoAnggota
- [ ] Tentukan jenis anggota berdasarkan UnitKerja (01=Dosen, 02=Staff, 03=Mahasiswa)
- [ ] Hitung limit pinjaman berdasarkan rumus:
  - Mahasiswa: 5x total simpanan (maksimal)
  - Dosen/Staff: (40% * gaji * tenor) + total simpanan
- [ ] Validasi jumlah pinjaman tidak melebihi limit
- [ ] Hitung perkiraan angsuran: (pinjaman / tenor) + (pinjaman * 0.02)

## 2. Update Template anggota_ajukan_pinjaman.html
- [ ] Tambahkan tampilan limit pinjaman maksimal
- [ ] Update script JS untuk menghitung perkiraan angsuran real-time
- [ ] Pastikan field gaji hanya untuk Dosen/Staff

## 3. Test dan Validasi
- [ ] Test untuk Mahasiswa: limit = 5x simpanan
- [ ] Test untuk Dosen/Staff: limit = (0.4 * gaji * tenor) + simpanan
- [ ] Pastikan error jika melebihi limit
- [ ] Verifikasi perhitungan angsuran
