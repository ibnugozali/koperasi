# TODO: Perbaiki Tampilan Bendahara Konfirmasi Transaksi

## Tujuan
Perbaiki tampilan `http://localhost:8081/bendahara/konfirmasi-transaksi` agar rapi dan card memiliki panjang/lebar yang seragam.

## Status: SELESAI

## Perubahan yang Dilakukan:

### 1. Card 3 & 4 Berdampingan (50%-50%)
- Card Import Excel dan Card Entri Manual sekarang dalam satu row
- `col-md-6` + `col-md-6` agar lebar sama (50%-50%)
- Pada mobile (`< 768px`) card tetap full width (`col-12`)

### 2. Tinggi Card Disamakan
- CSS `.content-card`: `height: 100%` 
- Class Bootstrap `h-100` pada card 3 & 4
- Card 1 & 2 menggunakan `col-12` (full width)

### 3. Page Header Tidak Tertimpa Navbar
- `.main-content`: `padding-top: 70px`
- Mobile: `padding-top: 60px`

### 4. Footer Dirapikan
- Flexbox layout: `display: flex; flex-direction: column; min-height: 100vh`
- Footer: `flex-shrink: 0; margin-left: 240px`
- Mobile: `margin-left: 0`

### 5. Semua Tag HTML Tertutup Benar
- `</div>` untuk row, col, card, container, page-wrapper
- `</form>` untuk form Excel
- Tidak ada tag yang tertinggal

## Build: Berhasil
