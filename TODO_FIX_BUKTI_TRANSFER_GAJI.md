# TODO: Pindahkan Approval Bukti Transfer Gaji ke Bendahara

## Rencana Perubahan
Pindahkan fitur approve/reject bukti transfer gaji dari Ketua ke Bendahara, karena fitur "Tambah Massal Potong Gaji via Excel" adalah tanggung jawab Bendahara.

## Langkah-langkah

### 1. Repository (`repository/bukti_transfer_gaji_repository.go`)
- [x] Tambah fungsi `CheckBuktiTransferGajiApproved(bulan, tahun int) (bool, error)`

### 2. Controller Ketua (`controllers/ketua_controller.go`)
- [x] Ubah `KetuaUploadBuktiTransferGajiPost`: Status dari "approved" → "pending"
- [x] Hapus fungsi `KetuaApproveBuktiTransferGaji`
- [x] Hapus fungsi `KetuaRejectBuktiTransferGaji`

### 3. Controller Bendahara (`controllers/bendahara_controller.go`)
- [x] `BendaharaKonfirmasiTransaksi`: Tambahkan data `BuktiList` ke template
- [x] Tambah fungsi `BendaharaApproveBuktiTransferGaji`
- [x] Tambah fungsi `BendaharaRejectBuktiTransferGaji`
- [x] `BendaharaImportPotongGajiExcel`: Ubah validasi ke `CheckBuktiTransferGajiApproved`

### 4. Routes (`routes/routes.go`)
- [x] Hapus route Ketua: `/ketua/upload-bukti-transfer-gaji/approve/:id`
- [x] Hapus route Ketua: `/ketua/upload-bukti-transfer-gaji/reject/:id`
- [x] Tambah route Bendahara: `/bendahara/konfirmasi-transaksi/bukti-transfer/approve/:id`
- [x] Tambah route Bendahara: `/bendahara/konfirmasi-transaksi/bukti-transfer/reject/:id`

### 5. Template Ketua (`templates/ketua/ketua_upload_bukti_transfer_gaji.html`)
- [x] Hapus tombol Approve/Reject di tabel riwayat
- [x] Tampilkan status sebagai badge read-only

### 6. Template Bendahara (`templates/bendahara/bendahara_konfirmasi_transaksi.html`)
- [x] Tambah card "Riwayat Upload Bukti Transfer Gaji" dengan tombol Approve/Reject
- [x] Update card Import Excel: kunci/buka berdasarkan status "approved"

### 7. Build & Test
- [x] Build ulang (`go build`) - BUILD SUCCESS

## Ringkasan Perubahan

### Alur Baru:
1. **Ketua** mengupload bukti transfer gaji di `/ketua/upload-bukti-transfer-gaji`
2. Status awal: **pending** (bukan approved lagi)
3. **Bendahara** melihat riwayat upload di `/bendahara/konfirmasi-transaksi`
4. Bendahara bisa **Approve** atau **Reject** bukti transfer
5. Fitur **Import Excel Potong Gaji** hanya aktif jika status = **approved**

### File yang Diubah:
| File | Perubahan |
|------|-----------|
| `repository/bukti_transfer_gaji_repository.go` | Tambah `CheckBuktiTransferGajiApproved()` |
| `controllers/ketua_controller.go` | Status upload jadi "pending", hapus fungsi approve/reject |
| `controllers/bendahara_controller.go` | Tambah fungsi approve/reject, update validasi import |
| `routes/routes.go` | Pindah route approve/reject ke bendahara |
| `templates/ketua/ketua_upload_bukti_transfer_gaji.html` | Hapus tombol approve/reject |
| `templates/bendahara/bendahara_konfirmasi_transaksi.html` | Tambah card riwayat + tombol approve/reject |
