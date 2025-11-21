# TODO: Add Transaction Confirmation Feature for Bendahara

## 1. Modify Anggota Controller (controllers/anggota_controller.go)
- [ ] Change AjukanPinjamanPost: Set status to 'pending' instead of 'aktif'
- [ ] Change AnggotaSimpananPost: Set status to 'pending' (ensure Detail model has status)
- [ ] Change AnggotaAngsuranPost: Set status to 'pending' instead of 'valid'

## 2. Update Routes (routes/routes.go)
- [ ] Add GET /bendahara/konfirmasi-transaksi for viewing pending transactions
- [ ] Add POST /bendahara/konfirmasi-transaksi/:type/:id for confirming/rejecting transactions

## 3. Add Bendahara Controller Functions (controllers/bendahara_controller.go)
- [ ] Add BendaharaKonfirmasiTransaksi function to display pending transactions
- [ ] Add BendaharaKonfirmasiTransaksiPost function to confirm/reject transactions

## 4. Update Repository (repository/transaksi_repository.go)
- [ ] Add GetPendingSimpanan function to get simpanan with status 'pending'
- [ ] Add GetPendingPinjaman function to get pinjaman with status 'pending'
- [ ] Add GetPendingAngsuran function to get angsuran with status 'pending'
- [ ] Add UpdateSimpananStatus function
- [ ] Add UpdatePinjamanStatus function
- [ ] Add UpdateAngsuranStatus function

## 5. Update Navbar (templates/layouts/bendahara_navbar.html)
- [ ] Add "Konfirmasi Transaksi" submenu under "Transaksi & Laporan"

## 6. Create Template (templates/bendahara/bendahara_konfirmasi_transaksi.html)
- [ ] Create template to display pending transactions
- [ ] Include buttons to confirm/reject each transaction

## 7. Update Models if needed (models/transaksi.go)
- [ ] Ensure Status fields support 'pending' (already do)
