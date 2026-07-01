package controllers

import "testing"

func TestFindReferensiIdentitasHeaderIndexPrefersSpecificHeader(t *testing.T) {
	headerMap := map[string]int{
		normalizeHeader("Nama Lengkap"):    0,
		normalizeHeader("Identitas"):       1,
		normalizeHeader("Nomer Identitas"): 2,
	}

	got := findReferensiIdentitasHeaderIndex(headerMap)
	if got != 2 {
		t.Fatalf("expected Nomer Identitas column index 2, got %d", got)
	}
}

func TestFindReferensiIdentitasHeaderIndexRejectsGenericIdentitas(t *testing.T) {
	headerMap := map[string]int{
		normalizeHeader("Nama Lengkap"): 0,
		normalizeHeader("Identitas"):    1,
	}

	got := findReferensiIdentitasHeaderIndex(headerMap)
	if got != -1 {
		t.Fatalf("expected generic Identitas column to be rejected, got %d", got)
	}
}

func TestFindReferensiIdentitasColumnRejectsRowNumberColumn(t *testing.T) {
	rows := [][]string{
		{"Nama Lengkap", "Nomer Identitas"},
		{"Siti Fatimah", "1"},
		{"Teguh Iman Santoso", "2"},
		{"Supriyadi", "3"},
	}
	headerMap := map[string]int{
		normalizeHeader("Nama Lengkap"):    0,
		normalizeHeader("Nomer Identitas"): 1,
	}

	got := findReferensiIdentitasColumn(rows, 0, headerMap)
	if got != -1 {
		t.Fatalf("expected sequential row-number column to be rejected, got %d", got)
	}
}

func TestFindReferensiIdentitasColumnSkipsRowNumberCandidate(t *testing.T) {
	rows := [][]string{
		{"Nama Lengkap", "Nomer Identitas", "NIP"},
		{"Siti Fatimah", "1", "198001012005012001"},
		{"Teguh Iman Santoso", "2", "197901022004011002"},
		{"Supriyadi", "3", "198203032006041003"},
	}
	headerMap := map[string]int{
		normalizeHeader("Nama Lengkap"):    0,
		normalizeHeader("Nomer Identitas"): 1,
		normalizeHeader("NIP"):             2,
	}

	got := findReferensiIdentitasColumn(rows, 0, headerMap)
	if got != 2 {
		t.Fatalf("expected NIP column index 2, got %d", got)
	}
}
