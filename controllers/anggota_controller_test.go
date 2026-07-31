package controllers

import "testing"

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
