# Rumus Aplikasi Koperasi

Dokumen ini merangkum rumus-rumus bisnis utama yang digunakan di aplikasi koperasi, lengkap dengan fungsi dan lokasi implementasinya di source code.

Catatan:
- Nomor baris di bawah mengacu pada kondisi source code saat dokumen ini dibuat pada 13 Mei 2026.
- Beberapa rumus muncul di backend dan frontend sekaligus. Saya cantumkan lokasi implementasi utama dan titik validasi/tampilan yang relevan.

## 1. Register

| Nama Rumus | Rumus | Fungsi | Lokasi |
| --- | --- | --- | --- |
| Nominal Simpanan Pokok | `nominalSimpananPokok = parseInt(nominal_simpanan) \|\| 100000` | Mengambil nominal simpanan pokok dari pengaturan, dengan fallback `100000` | `templates/utama/register.html:148`, `controllers/auth_controller.go:604` |
| Validasi Potong Gaji | `gaji_bersih < nominal_simpanan_pokok` | Menentukan apakah metode `potong_gaji` boleh dipilih/diproses | `templates/utama/register.html:243-259`, `controllers/auth_controller.go:750-751` |

## 2. Simpanan dan Saldo

| Nama Rumus | Rumus | Fungsi | Lokasi |
| --- | --- | --- | --- |
| Total Simpanan | `totalSimpanan = simpananPokok + totalSimpananLainnya` | Menghitung total simpanan anggota | `repository/anggota_repository.go:344` |
| Saldo Bersih | `saldoBersih = totalSimpanan - totalSisaPinjaman` | Menghitung saldo bersih anggota | `repository/anggota_repository.go:367` |
| Total Simpanan Wajib | `simpananWajib = totalSimpananWajib + totalSimpananWajibDetail` | Menggabungkan simpanan wajib dari potongan otomatis dan konfirmasi manual | `repository/anggota_repository.go:420` |
| Kekurangan Simpanan Wajib | `kekurangan = nominalSimpananWajib - simpananWajib[idAnggota]` | Menghitung kekurangan setoran simpanan wajib | `repository/anggota_repository.go:617` |
| Nominal Potongan Simpanan Wajib | `jumlahPotong = nominalSimpananWajib` | Menentukan nominal potongan simpanan wajib otomatis | `repository/anggota_repository.go:814` |
| Sisa Gaji | `sisaGaji = gajiBulanan - potongan` | Menghitung sisa gaji setelah dipotong | `repository/transaksi_repository.go:682,707,731,756`, `controllers/admin_controller.go:1573`, `controllers/bendahara_controller.go:1117,1590,3832` |

## 3. Pinjaman

| Nama Rumus | Rumus | Fungsi | Lokasi |
| --- | --- | --- | --- |
| Bunga Nominal | `bungaNominal = jumlahPinjaman * bunga / 100` | Menghitung bunga pinjaman | `controllers/anggota_controller.go:386,417` |
| Total Kewajiban | `totalKewajiban = totalPinjaman + bungaNominal` | Menghitung total kewajiban pinjaman | `controllers/anggota_controller.go:387,418` |
| Sisa Angsuran | `sisaAngsuran = jangkaWaktu - angsuranTerbayar` | Menghitung sisa jumlah angsuran | `controllers/anggota_controller.go:432` |
| Sisa Pinjaman | `sisaPinjaman = jumlahPinjaman - totalAngsuranTerbayar` | Menghitung sisa pinjaman | `controllers/anggota_controller.go:682,806,2598` |
| Persentase Pinjaman Terbayar | `persentaseTerbayar = totalAngsuranTerbayar / totalKewajiban * 100` | Menghitung persentase pelunasan pinjaman | `controllers/anggota_controller.go:440` |
| Persentase Pinjaman Terbayar Versi 2 | `(jumlahPinjaman - sisaPinjaman) / jumlahPinjaman * 100` | Alternatif perhitungan persentase pelunasan | `controllers/anggota_controller.go:835` |

## 4. Limit Pengajuan Pinjaman

| Nama Rumus | Rumus | Fungsi | Lokasi |
| --- | --- | --- | --- |
| Limit Mahasiswa | `limitPinjaman = 5 * totalSimpanan` | Menghitung limit pinjaman untuk mahasiswa | `controllers/anggota_controller.go:1503,1686`, `templates/anggota/anggota_ajukan_pinjaman.html:477`, `templates/ketua/ketua_persyaratan_pinjaman.html:364` |
| Limit Dosen/Tenaga Pendidikan | `limitPinjaman = 0.4 * gajiBulanan * jangkaWaktu` | Menghitung limit pinjaman berdasarkan kemampuan bayar | `controllers/anggota_controller.go:1706,1725`, `templates/anggota/anggota_ajukan_pinjaman.html:481,496,511`, `templates/ketua/ketua_persyaratan_pinjaman.html:366` |

## 5. Angsuran

| Nama Rumus | Rumus | Fungsi | Lokasi |
| --- | --- | --- | --- |
| Pokok Per Bulan | `pokokPerBulan = jumlahPinjaman / jangkaWaktu` | Menghitung pokok angsuran per bulan | `controllers/anggota_controller.go:2266,2368`, `repository/transaksi_repository.go:1380` |
| Jasa Per Bulan | `jasaPerBulan = bunga / jangkaWaktu` | Menghitung jasa per bulan jika bunga sudah nominal | `controllers/anggota_controller.go:2267` |
| Jasa Per Bulan dari Persen | `jasaPerBulan = (bunga / 100 * jumlahPinjaman) / jangkaWaktu` | Menghitung jasa per bulan jika bunga masih persen | `controllers/anggota_controller.go:2369` |
| Perkiraan Angsuran Bulanan | `perkiraanAngsuranBulan = pokokPerBulan + jasaPerBulan` | Menghitung total angsuran bulanan | `controllers/anggota_controller.go:2268,2370`, `repository/transaksi_repository.go:1380`, `templates/anggota/anggota_ajukan_pinjaman.html:544`, `templates/ketua/ketua_persyaratan_pinjaman.html:354` |
| Sisa Pinjaman Setelah Bayar | `sisaSetelah = sisaSebelum - jumlahAngsuran` | Menghitung sisa pinjaman setelah pembayaran angsuran | `controllers/anggota_controller.go:2562`, `controllers/bendahara_controller.go:5460` |

## 6. Laporan

| Nama Rumus | Rumus | Fungsi | Lokasi |
| --- | --- | --- | --- |
| Arus Kas | `arusKas = totalSimpanan - (totalPinjaman + totalPengambilan)` | Menghitung arus kas koperasi | `repository/transaksi_repository.go:512` |
| Total Pembayaran Bulanan | `totalPembayaran = simpananWajibBulanan + simpananHariRayaBulanan + simpananSukarelaBulanan + simpananLainnyaBulanan + angsuranBulanan` | Menghitung total beban pembayaran bulanan anggota | `repository/transaksi_repository.go:1383` |

## 7. SHU

| Nama Rumus | Rumus | Fungsi | Lokasi |
| --- | --- | --- | --- |
| Kontribusi Simpanan SHU | `kontribusiSimpanan = simpananPokok + totalSimpananWajib + totalSimpananHariRaya + totalSimpananSukarela + totalSimpananLainnya` | Dasar kontribusi simpanan untuk pembagian SHU | `repository/transaksi_repository.go:1409` |
| Pool SHU Pinjaman | `shuPoolPinjaman = totalJasaPeriode * 0.5` | Alokasi 50% SHU dari jasa pinjaman | `repository/transaksi_repository.go:1428` |
| Pool SHU Simpanan | `shuPoolSimpanan = totalJasaPeriode * 0.5` | Alokasi 50% SHU dari kontribusi simpanan | `repository/transaksi_repository.go:1429` |
| SHU dari Pinjaman | `shuPinjaman = (jasaDibayar / totalJasaPeriode) * shuPoolPinjaman` | Bagian SHU anggota dari kontribusi pinjaman | `repository/transaksi_repository.go:1442` |
| SHU dari Simpanan | `shuSimpanan = (kontribusiSimpanan / totalKontribusiSimpanan) * shuPoolSimpanan` | Bagian SHU anggota dari kontribusi simpanan | `repository/transaksi_repository.go:1446` |
| Total SHU Anggota | `jumlah_shu = shuPinjaman + shuSimpanan` | Total SHU yang diterima anggota | `repository/transaksi_repository.go:1451` |

## Catatan

- Beberapa rumus muncul di backend dan frontend sekaligus untuk kebutuhan validasi dan tampilan.
- Nilai fallback seperti `100000` dipakai saat pengaturan belum tersedia di database.
- Untuk rumus yang muncul lebih dari sekali, saya cantumkan titik implementasi utamanya dan titik lain yang relevan untuk audit.
