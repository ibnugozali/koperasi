# Fix Total Saldo Simpanan: ajukan-pengambilan-simpanan vs profil

## Status: ✅ APPROVED - In Progress

**Goal**: Make "Total Saldo Simpanan Anda" identical to Profil's Total Simpanan by excluding pokok.

### ⬜ Step 1: Create TODO [COMPLETED]
### ⬜ Step 2: Edit controllers/anggota_controller.go → AjukanPengambilanSimpanan()
   - Replace `GetSaldoAnggota()` with:
     ```
     simpananByJenis, _ := repository.GetDetailSimpananByJenis(userID)
     totalSimpanan := simpananByJenis["wajib"] + simpananByJenis["sukarela"] + 
                     simpananByJenis["hari_raya"] + simpananByJenis["umroh_haji"] + 
                     simpananByJenis["qurban"]
     ```
### ⬜ Step 3: Test pages
   - http://localhost:8081/anggota/profil → note Total Simpanan
   - http://localhost:8081/anggota/ajukan-pengambilan-simpanan → verify match
### ⬜ Step 4: Update TODO → attempt_completion

**Current Progress**: Ready for code edit
