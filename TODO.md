# TODO: Perbaiki Pinjaman - Tambahkan Kolom Gaji Bulanan

- [x] Tambahkan kolom input "Jumlah Gaji Bulanan (Rp)" di Bagian 2: Rincian Permohonan Pinjaman dalam file templates/pelayanan/pinjaman.html
- [x] Pindahkan posisi kolom "Jumlah Gaji Bulanan (Rp)" ke atas "Jumlah Pinjaman yang Diajukan (Rp)"
- [x] Hapus Bagian 1: Data Pribadi Peminjam (Anggota) dan renumber bagian berikutnya
- [x] Tambahkan pilihan Unit Kerja di atas kolom "Jumlah Gaji Bulanan (Rp)"
- [x] Verifikasi bahwa formulir ditampilkan dengan benar setelah perubahan (server berjalan di localhost:8081)

# TODO: Tambahkan Tombol Tolak di Konfirmasi Admin

- [x] Tambahkan tombol "Tolak" di kolom Aksi pada templates/admin/konfirmasi.html dengan konfirmasi JavaScript
- [x] Tambah route POST /admin/reject/:id di routes/routes.go
- [x] Tambah function RejectMembership di controllers/admin_controller.go untuk menghapus anggota
- [x] Test tombol tolak menghapus anggota dari database
