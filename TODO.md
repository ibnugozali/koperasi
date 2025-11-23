# TODO: Fix Template Error for bendahara_edit_bunga.html

## Tasks
- [ ] Update define in templates/layouts/bendahara_navbar.html to "layouts/bendahara_navbar.html"
- [ ] Update template includes in templates/layouts/bendahara_layout.html
- [ ] Update template includes in templates/bendahara_edit_bunga.html
- [ ] Update template includes in templates/bendahara/bendahara_transaksi_content.html
- [ ] Update template includes in templates/bendahara/bendahara_riwayat_content.html
- [ ] Update template includes in templates/bendahara/bendahara_laporan.html
- [ ] Update template includes in templates/bendahara/bendahara_konfirmasi_transaksi.html
- [ ] Update template includes in templates/bendahara/bendahara_dashboard_content.html
- [ ] Update template includes in templates/bendahara/bendahara_anggota_konfirmasi.html
- [ ] Fix render path in controllers/bendahara_controller.go BendaharaUpdateBunga function
- [ ] Test the application to ensure error is resolved

## Completed Tasks
- [x] Fix 400 Bad Request error on POST "/anggota/simpanan"
  - Updated AnggotaSimpananPost controller to handle multiple savings types (wajib, sukarela, hari_raya)
  - Added proper validation for multiple form fields
  - Tested application build and startup successfully
