# Fix Informasi Kemampuan Bayar Visibility

## Task
Hide "Informasi Kemampuan Bayar" initially when "Resume Pinjaman" appears on /anggota/ajukan-pinjaman, and show it after user clicks "Melakukan Pinjaman Lagi".

## Steps
- [x] 1. Remove duplicate "Informasi Kemampuan Bayar" block (the one inside `{{ if not .ResumeGabungan }}`).
- [x] 2. Add `id="informasiKemampuanBayar"` to the remaining first block.
- [x] 3. Add conditional `style="display:none;"` on that block when `.ResumeGabungan` exists.
- [x] 4. Fix inline JS for "Melakukan Pinjaman Lagi":
  - Remove invalid `</style>` wrappers around the `<script>`.
  - Use `btnPinjamLagi.closest('.floating-resume-pinjaman')` instead of broken `getElementById('resumePinjamanCardGabungan')`.
  - Add code to show `#informasiKemampuanBayar` when clicking "Melakukan Pinjaman Lagi".

