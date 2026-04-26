# TODO: Perbaikan Nomor Rekening & Info Angsuran

## Ringkasan
Perbaikan untuk:
1. **Nomor rekening koperasi kosong** di halaman `/anggota/simpanan`
2. **Info metode angsuran** (`potong_gaji`) tidak tampil di dashboard bendahara

---

## Daftar Tugas

- [x] 1. **Repository**: Tambahkan field `MetodeAngsuran` + `Scan` di `GetPendingAngsuran()`
  File: `repository/transaksi_repository.go`
- [x] 2. **Template**: Update `bendahara_konfirmasi_transaksi.html` tampilkan info angsuran otomatis
  File: `templates/bendahara/bendahara_konfirmasi_transaksi.html`

---

## Perubahan yang Dilakukan

### 1. Fix Nomor Rekening Kosong (`/anggota/simpanan`)
- **File**: `repository/anggota_repository.go`
- **Fungsi**: `GetNomorRekening(jenis string)`
- **Perubahan**: Tambahkan fallback ke tabel `pengaturan` (key `nomor_rekening`) jika tabel `nomor_rekening` belum punya data untuk jenis tersebut.

```go
if err == sql.ErrNoRows {
    // Fallback ke tabel pengaturan jika belum ada di nomor_rekening
    err = db.QueryRow("SELECT nilai FROM pengaturan WHERE nama_pengaturan = 'nomor_rekening'").Scan(&nomor)
    if err == sql.ErrNoRows {
        return "", nil
    }
    if err != nil {
        return "", err
    }
    return nomor, nil
}
```

### 2. Tampilkan Info Metode Angsuran di Dashboard Bendahara
- **File**: `repository/transaksi_repository.go`
- **Fungsi**: `GetPendingAngsuran()`
- **Perubahan**: SELECT tambahkan `COALESCE(p.metode_angsuran, '')` dan Scan tambahkan `&a.MetodeAngsuran`.

- **File**: `templates/bendahara/bendahara_konfirmasi_transaksi.html`
- **Perubahan**: Di kolom "Jumlah Angsuran", tampilkan info jika `MetodeAngsuran == "potong_gaji"`:

```html
{{if eq .Angsuran.MetodeAngsuran "potong_gaji"}}
<div class="small text-info mt-1">
    <i class="fa-solid fa-circle-info me-1"></i>Angsuran diambil otomatis dari rekening/gaji anggota
</div>
{{end}}
```

---

## Cara Setup Nomor Rekening

Jika nomor rekening masih kosong, ada 2 cara mengisinya:

### Opsi A: Via Tabel `pengaturan` (Recommended)
```sql
INSERT INTO pengaturan (nama_pengaturan, nilai, keterangan)
VALUES ('nomor_rekening', '1234567890 (Bank ABC)', 'Nomor rekening koperasi untuk transfer');
```

### Opsi B: Via Tabel `nomor_rekening`
```sql
INSERT INTO nomor_rekening (jenis, nomor) VALUES ('simpanan', '1234567890 (Bank ABC)');
INSERT INTO nomor_rekening (jenis, nomor) VALUES ('angsuran', '1234567890 (Bank ABC)');
INSERT INTO nomor_rekening (jenis, nomor) VALUES ('register', '1234567890 (Bank ABC)');
```

> **Note**: Tabel `nomor_rekening` memiliki prioritas lebih tinggi dari `pengaturan`. Jika data ada di `nomor_rekening`, itu yang akan ditampilkan.

---

## Status: ✅ SELESAI

