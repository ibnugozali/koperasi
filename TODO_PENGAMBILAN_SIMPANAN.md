# Total Saldo Simpanan dan Pengembalian Saat Keluar

## Status: ✅ COMPLETED

**Goal**: Keep total simpanan consistent across member and staff views, include Simpanan Pokok, and return Simpanan Pokok/Wajib only through the approved membership-exit process.

### ✅ Total simpanan
- Member profile, withdrawal, loan, and staff detail views now sum all savings types, including Simpanan Pokok.
- Approved Simpanan Pokok withdrawals remain deducted from the displayed balance.

### ✅ Withdrawal restrictions
- Simpanan Pokok and Simpanan Wajib are excluded from regular withdrawal options and rejected by the backend endpoint.
- Both balances are included in the member-exit refund request, which is processed after the exit request is approved.

### ✅ Validation
- `go test ./...`
- `git diff --check`
