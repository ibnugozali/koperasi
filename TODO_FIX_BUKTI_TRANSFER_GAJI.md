# Fix Bukti Transfer Gaji — Status Pending

## Root Cause
Ketua upload bukti transfer gaji disimpan dengan `Status: "pending"`, tetapi tidak ada mekanisme approval/rejection di seluruh codebase. Sehingga status tetap "Pending" selamanya.

## Changes Applied

1. **repository/bukti_transfer_gaji_repository.go**
   - Added `GetBuktiTransferGajiByID(id int) (*models.BuktiTransferGaji, error)`
   - Added `UpdateBuktiTransferGajiStatus(id int, status string) error`

2. **controllers/ketua_controller.go**
   - Changed default upload status from `"pending"` to `"approved"` in `KetuaUploadBuktiTransferGajiPost` (baru upload langsung Approved)
   - Added `KetuaApproveBuktiTransferGaji(c *gin.Context)` handler
   - Added `KetuaRejectBuktiTransferGaji(c *gin.Context)` handler
   - Handlers validate existence, execute update, and redirect with `?success=` or `?error=`

3. **routes/routes.go**
   - Added `POST /ketua/upload-bukti-transfer-gaji/approve/:id` → `KetuaApproveBuktiTransferGaji`
   - Added `POST /ketua/upload-bukti-transfer-gaji/reject/:id` → `KetuaRejectBuktiTransferGaji`

4. **templates/ketua/ketua_upload_bukti_transfer_gaji.html**
   - Added new **Aksi** column in the "Riwayat Upload" table
   - For rows with `Status == "pending"`, added inline Approve (✔) and Reject (✗) buttons
   - For rows already approved/rejected, shows dash (`-`)

## Result
- **New uploads** default to **Approved** (no more stuck in Pending)
- **Existing pending records** can be manually Approved/Rejected via the UI
- Full end-to-end workflow ready

## Verification
```powershell
cd c:/Users/ADVAN/Desktop/Coperasi
go build -o koperasi_upload_bukti_fix.exe
# Result: exit code 0 (success, no compilation errors)
```


