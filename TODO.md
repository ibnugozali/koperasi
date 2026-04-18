# Fix Total Pinjaman di Profil Anggota (/anggota/profil) - Tidak Sesuai Masa

## Status: ✅ Approved & In Progress

### 1. [x] Gather Information (Completed)
   - Analyzed files: controllers/anggota_controller.go, repository/anggota_repository.go, templates/anggota/anggota_profil.html, models/anggota.go
   - Root cause: No date filter in pinjaman/angsuran queries (lifetime totals vs monthly)

### 2. [✅] Add Period-Filtered Repository Functions
   - `repository/anggota_repository.go`:
     - `GetRingkasanPinjamanByPeriod(id, month, year) (totalAktif, sisa float64, err)` ✅

### 3. [✅] Update Controller Handler
   - `controllers/anggota_controller.go` (AnggotaProfil):
     - Parse query params `?bulan=X&tahun=Y` (default current month) ✅
     - Use filtered repo functions ✅
     - Pass period data to template ✅

### 3. [✅] Update Controller Handler
   - `controllers/anggota_controller.go` (AnggotaProfil):
     - Parse query params `?bulan=X&tahun=Y` (default current month)
     - Use filtered repo functions ✅
     - Pass period data to template

### 4. [✅] Update Profile Template
   - `templates/anggota/anggota_profil.html`:
     - Add month/year selector form ✅
     - Display "Total Pinjaman [Period]" label ✅
     - JS reload with params

### 5. [ ] Test & Verify
   - Restart: `go run main.go`
   - Visit /anggota/profil → Check monthly totals match DB
   - Test selectors, all-time fallback

### 6. [ ] Completion
   - Update TODO.md ✅
   - attempt_completion

### 5. [ ] Test & Verify
   - Restart: `go run main.go`
   - Visit /anggota/profil → Check monthly totals match DB
   - Test selectors, all-time fallback

**Next Step: Fix compilation errors & test**

