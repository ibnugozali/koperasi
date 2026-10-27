package models

import "testing"

func TestHitungJasaPinjamanPerBulan(t *testing.T) {
	const (
		pinjaman = 2_000_000.0
		jasa     = 2.0
		tenor    = 20
	)

	totalJasa := HitungJasaPinjaman(pinjaman, jasa, tenor)
	if totalJasa != 800_000 {
		t.Fatalf("expected total monthly flat service of 800000, got %.0f", totalJasa)
	}

	angsuran := HitungAngsuranBulanan(pinjaman, totalJasa, tenor)
	if angsuran != 140_000 {
		t.Fatalf("expected monthly installment of 140000, got %.0f", angsuran)
	}
}

func TestHitungJasaPinjamanRejectsInvalidTerm(t *testing.T) {
	if totalJasa := HitungJasaPinjaman(2_000_000, 2, 0); totalJasa != 0 {
		t.Fatalf("expected no service for invalid term, got %.0f", totalJasa)
	}
	if installment := HitungAngsuranBulanan(2_000_000, 800_000, 0); installment != 0 {
		t.Fatalf("expected no installment for invalid term, got %.0f", installment)
	}
}
