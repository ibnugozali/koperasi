package controllers

import (
	"koperasi-simpan-pinjam/models"
	"testing"
)

func TestIsNaturalNumberInput(t *testing.T) {
	if !isNaturalNumberInput("12345") {
		t.Fatalf("expected digits-only string to be accepted")
	}
	if isNaturalNumberInput("abc123") {
		t.Fatalf("expected letters to be rejected")
	}
	if isNaturalNumberInput("123.45") {
		t.Fatalf("expected decimal values to be rejected")
	}
	if isNaturalNumberInput("-100") {
		t.Fatalf("expected negative values to be rejected")
	}
	if isNaturalNumberInput("0") {
		t.Fatalf("expected zero to be rejected")
	}
	if isNaturalNumberInput("0123") {
		t.Fatalf("expected leading zeroes to be rejected")
	}
	if isNaturalNumberInput("") {
		t.Fatalf("expected empty input to be rejected")
	}
	if isNaturalNumberInput("123abc") {
		t.Fatalf("expected trailing letters to be rejected")
	}
	if isNaturalNumberInput("+123") {
		t.Fatalf("expected plus sign to be rejected")
	}
}

func TestParseOptionalNaturalAmount(t *testing.T) {
	amount, filled, message := parseOptionalNaturalAmount("Simpanan Sukarela", "")
	if amount != 0 || filled || message != "" {
		t.Fatalf("expected empty amount to be treated as not filled, got amount=%.0f filled=%v message=%q", amount, filled, message)
	}

	amount, filled, message = parseOptionalNaturalAmount("Simpanan Sukarela", "1000")
	if amount != 1000 || !filled || message != "" {
		t.Fatalf("expected valid natural amount, got amount=%.0f filled=%v message=%q", amount, filled, message)
	}

	for _, value := range []string{"0", "0123", "abc123", "123abc", "-100", "+100", "123.45"} {
		if _, _, message := parseOptionalNaturalAmount("Simpanan Sukarela", value); message == "" {
			t.Fatalf("expected %q to be rejected", value)
		}
	}
}

func TestIsValidManualTunaiCicilanAmount(t *testing.T) {
	if !isValidManualTunaiCicilanAmount(5000, 5000, 5000) {
		t.Fatalf("expected exact amount within the allowed range to be accepted")
	}
	if !isValidManualTunaiCicilanAmount(4500, 4000, 5000) {
		t.Fatalf("expected amount within the allowed range to be accepted")
	}
	if isValidManualTunaiCicilanAmount(3999, 4000, 5000) {
		t.Fatalf("expected amount below the minimum to be rejected")
	}
	if isValidManualTunaiCicilanAmount(5001, 4000, 5000) {
		t.Fatalf("expected amount above the maximum to be rejected")
	}
	if isValidManualTunaiCicilanAmount(0, 4000, 5000) {
		t.Fatalf("expected zero amount to be rejected")
	}
}

func TestSelectPendingAngsuranForManualTunai(t *testing.T) {
	pendingList := []models.Angsuran{
		{IDAngsuran: 1, JumlahAngsuran: 1000, SisaPinjaman: 4000},
		{IDAngsuran: 2, JumlahAngsuran: 2000, SisaPinjaman: 3000},
	}

	selected := selectPendingAngsuranForManualTunai(pendingList, 5000)
	if selected == nil || selected.IDAngsuran != 1 {
		t.Fatalf("expected first pending record to be selected for exact total due payment")
	}

	selected = selectPendingAngsuranForManualTunai(pendingList, 6000)
	if selected == nil || selected.IDAngsuran != 1 {
		t.Fatalf("expected first pending record to be selected as fallback")
	}
}

func TestIsAngsuranTerbayarRequiresKetuaApproval(t *testing.T) {
	if isAngsuranTerbayar("confirmed") {
		t.Fatalf("expected confirmed to be treated as pending before ketua approval")
	}
	if isAngsuranTerbayar("pending") {
		t.Fatalf("expected pending to be treated as unpaid")
	}
	if !isAngsuranTerbayar("diterima") {
		t.Fatalf("expected diterima to be treated as paid")
	}
	if !isAngsuranTerbayar("lunas") {
		t.Fatalf("expected lunas to be treated as paid")
	}
}

func TestIsAngsuranProfilTerbayarRequiresKetuaApproval(t *testing.T) {
	if isAngsuranProfilTerbayar("confirmed") {
		t.Fatalf("expected confirmed to be treated as pending before ketua approval")
	}
	if isAngsuranProfilTerbayar("pending") {
		t.Fatalf("expected pending to be treated as unpaid")
	}
	if !isAngsuranProfilTerbayar("diterima") {
		t.Fatalf("expected diterima to be treated as paid")
	}
}

func TestCalculatePinjamanLimit(t *testing.T) {
	if got := calculatePinjamanLimit("03", 1000000, 0, 12); got != 5000000 {
		t.Fatalf("expected mahasiswa limit to be 5x total simpanan, got %.0f", got)
	}

	if got := calculatePinjamanLimit("01", 0, 5000000, 12); got != 24000000 {
		t.Fatalf("expected dosen limit to be 0.4 x gaji x tenor, got %.0f", got)
	}
}

func TestIsJumlahPinjamanWithinLimit(t *testing.T) {
	if !isJumlahPinjamanWithinLimit(5000000, "03", 1000000, 0, 12) {
		t.Fatalf("expected amount at the limit to be accepted")
	}
	if isJumlahPinjamanWithinLimit(5000001, "03", 1000000, 0, 12) {
		t.Fatalf("expected amount above the limit to be rejected")
	}
	if !isJumlahPinjamanWithinLimit(24000000, "01", 0, 5000000, 12) {
		t.Fatalf("expected amount at the limit to be accepted")
	}
	if isJumlahPinjamanWithinLimit(24000001, "01", 0, 5000000, 12) {
		t.Fatalf("expected amount above the limit to be rejected")
	}
}
