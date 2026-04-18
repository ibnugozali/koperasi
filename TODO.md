# Progress Fix Resume Pinjaman Lunas Tidak Hilang

**Status: ✅ PLAN APPROVED - Executing step-by-step**

## TODO Steps (from approved plan):
- [x] 1. Create TODO.md ✅
- [✅] 2. Run DB UPDATE query: **UPDATE 0** rows (no pinjaman need fix, state already correct)"
- [✅] 3. Strengthened repo query **with JOIN exclude sisa<=0** ✅
- [✅] 4. Server restarted `go run main.go` → Running :8081 ✅
- [ ] 5. Test /anggota/ajukan-pinjaman → Resume card hilang if sisa<=0
- [ ] 6. Verify auto-update works on new angsuran lunas
- [ ] 7. attempt_completion

**Next:** DB query execution...

**DB Query Ready (safe, from koperasi.sql):**
```
UPDATE pinjaman SET status = 'lunas' WHERE id_pinjaman IN (
  SELECT p.id_pinjaman FROM pinjaman p LEFT JOIN (
    SELECT id_pinjaman, SUM(CASE WHEN status IN ('confirmed','lunas','diterima') THEN sisa_pinjaman ELSE 0 END) AS total_angsuran
    FROM angsuran GROUP BY id_pinjaman
  ) a ON p.id_pinjaman = a.id_pinjaman
  WHERE (p.jumlah_pinjaman - COALESCE(a.total_angsuran,0)) <= 0 AND p.status != 'lunas'
);
```

