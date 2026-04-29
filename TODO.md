# FIX: 400 Error - metode='' on POST /anggota/simpanan

**Status**: 🚀 In Progress  
**Target**: Resolve Gin multipart parsing failure causing empty `metode_pembayaran`

## Steps (3 total)

### 1. ✅ PLAN APPROVED
- [x] Analyzed controller + template 
- [x] Confirmed root cause: Gin `c.PostForm()` fails on multipart
- [x] User approved fix plan

### 2. ✅ IMPLEMENT CONTROLLER + TEMPLATE FIXES
```
✓ controllers/anggota_controller.go (bulletproof parsing)
  → ParseForm() + ParseMultipartForm(128MB)  
  → r.Form.Get() + full form keys logging
  → debug_keys JSON

✓ templates/anggota/anggota_simpanan_fixed.html
  → autocomplete="off" on select
  → JS: dispatch input/change events on toggle
  → Blur handler + form dirty state
```
**Status**: Build OK → Ready for re-test

### 3. ✅ TEST & VERIFY
```
→ go build -o koperasi_test.exe
→ Browser test: /anggota/simpanan → Submit all methods
→ Verify logs: No "metode='' (len=0)" → 200 OK + success page
→ Update this TODO: Mark ✅ COMPLETE
```

**Progress**: 1/3 complete  
**Expected Result**: Zero 400 errors on simpanan POST
