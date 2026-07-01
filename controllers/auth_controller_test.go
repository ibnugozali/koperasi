package controllers

import "testing"

func TestNormalizeRegisterPhoneKeepsSubscriberDigits(t *testing.T) {
	tests := map[string]string{
		"081234567890":     "81234567890",
		"+6281234567890":   "81234567890",
		"62 812-3456-7890": "81234567890",
	}

	for input, want := range tests {
		if got := normalizeRegisterPhone(input); got != want {
			t.Fatalf("normalizeRegisterPhone(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestIsRegisterPhoneValidRequiresTenToThirteenDigits(t *testing.T) {
	valid := []string{"8123456789", "8123456789012"}
	for _, phone := range valid {
		if !isRegisterPhoneValid(phone) {
			t.Fatalf("expected %q to be valid", phone)
		}
	}

	invalid := []string{"812345678", "81234567890123", "81234abc90"}
	for _, phone := range invalid {
		if isRegisterPhoneValid(phone) {
			t.Fatalf("expected %q to be invalid", phone)
		}
	}
}
