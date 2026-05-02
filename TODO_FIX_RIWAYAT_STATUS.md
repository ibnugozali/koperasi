# TODO: Fix Riwayat Status Display - Menunggu Konfirmasi Ketua

## Problem Analysis
Current issue with http://localhost:8081/anggota/riwayat page:

1. **Transaction Flow**:
   - Anggota submits → status = "pending"
   - Bendahara checks/verifies at /bendahara/konfirmasi-transaksi → status = "confirmed"
   - Ketua approves at /ketua/konfirmasi-transaksi → status = "diterima" or "lunas"

2. **Current Problem**:
   - When status = "pending", it only shows "Dalam Proses"
   - Should show "Menunggu konfirmasi Ketua" because:
     - Bendahara only does checking/verification at /bendahara/konfirmasi-transaksi
     - Final confirmation always remains at /ketua/konfirmasi-transaksi

## Plan

### Files to Modify:
1. `controllers/anggota_controller.go` - Update status display in `AnggotaRiwayatPage` function
2. (Optional) `repository/riwayat_repository.go` - Add more status filtering if needed

### Changes:

#### In `controllers/anggota_controller.go`:

Update the status mapping in `AnggotaRiwayatPage` function:

Current mapping:
```go
switch status {
case "pending":
    status = "Dalam Proses"
case "confirmed", "diterima", "aktif":
    status = "Diterima"
case "rejected", "gagal":
    status = "Ditolak"
case "lunas":
    status = "Lunas"
}
```

New mapping (add more descriptive statuses):
```go
switch status {
case "pending":
    status = "Menunggu konfirmasi Ketua"
case "confirmed":
    status = "Menunggu konfirmasi Ketua"  // confirmed by bendahara, waiting for chairman
case "diterima":
    status = "Diterima"
case "aktif":
    status = "Aktif"
case "rejected", "gagal":
    status = "Ditolak"
case "lunas":
    status = "Lunas"
default:
    status = "Dalam Proses"
}
```

## Implementation Steps

1. Update status display in AnggotaRiwayatPage
2. Update template if needed for better status labels
3. Test and verify the changes
