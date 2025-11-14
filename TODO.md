# TODO: Perbaikan Upload Logo dengan Background Transparan

## Tugas Utama
- [x] Perbaiki logika remove background di preview (admin_logo.js)
- [x] Modifikasi upload untuk mengirim gambar transparan sebagai base64
- [x] Update fungsi UploadLogo di Go untuk decode dan simpan PNG transparan
- [x] Pastikan navbar menampilkan logo dengan background transparan

## Langkah-langkah Detail
1. **Perbaiki admin_logo.js**
   - [x] Tingkatkan akurasi deteksi background (gunakan algoritma yang lebih baik)
   - [x] Pastikan preview menampilkan background transparan dengan benar
   - [x] Modifikasi upload untuk mengirim data canvas sebagai base64

2. **Update admin_controller.go**
   - [x] Tambahkan import untuk image processing
   - [x] Modifikasi UploadLogo untuk menerima base64
   - [x] Decode base64 dan simpan sebagai PNG

3. **Perbaiki admin_navbar.html**
   - [x] Tambahkan CSS untuk background transparan pada img logo
   - [x] Pastikan logo ditampilkan dengan benar di sidebar

4. **Testing**
   - [x] Test upload logo baru
   - [x] Verifikasi preview menampilkan transparan
   - [x] Pastikan navbar update otomatis dengan logo transparan
   - [x] Pastikan logo tersimpan setelah refresh halaman
   - [x] Pastikan logo tersimpan setelah login kembali
