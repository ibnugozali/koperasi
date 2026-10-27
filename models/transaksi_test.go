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

func TestHitungJasaPinjamanGabunganSaldoLama(t *testing.T) {
	const (
		pinjamanBaru = 1_000_000.0
		saldoLama    = 500_000.0
		jasa         = 2.0
		tenor        = 20
	)

	pinjaman := Pinjaman{JumlahPinjaman: pinjamanBaru, SaldoLamaDigabung: saldoLama}
	totalJasa := HitungJasaPinjaman(pinjaman.PokokGabungan(), jasa, tenor)
	if totalJasa != 600_000 {
		t.Fatalf("expected total service of 600000 on combined principal, got %.0f", totalJasa)
	}

	angsuran := HitungAngsuranBulanan(pinjaman.PokokGabungan(), totalJasa, tenor)
	if angsuran != 105_000 {
		t.Fatalf("expected monthly installment of 105000, got %.0f", angsuran)
	}
}

func TestHitungSisaKewajibanPinjamanMencakupJasaBelumTerbayar(t *testing.T) {
	const (
		pokok      = 1_000_000.0
		jasa       = 200_000.0
		totalBayar = 1_000_000.0
		sisa       = 200_000.0
	)

	got := HitungSisaKewajibanPinjaman(pokok, totalBayar, jasa)
	if got != sisa {
		t.Fatalf("expected remaining obligation of %.0f, got %.0f", sisa, got)
	}

	pokokGabungan := 1_800_000 + got
	jasaBaru := HitungJasaPinjaman(pokokGabungan, 2, 10)
	if jasaBaru != 400_000 {
		t.Fatalf("expected new service of 400000 on combined principal, got %.0f", jasaBaru)
	}
	if angsuran := HitungAngsuranBulanan(pokokGabungan, jasaBaru, 10); angsuran != 240_000 {
		t.Fatalf("expected monthly installment of 240000, got %.0f", angsuran)
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
