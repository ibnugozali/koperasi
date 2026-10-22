package controllers

import (
	"strings"
	"testing"
)

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

func TestBuildWhatsAppChatURLFormatsPhoneAndMessage(t *testing.T) {
	got := buildWhatsAppChatURL("0812-3456-7890", "Halo Bendahara")
	want := "https://wa.me/6281234567890?text=Halo+Bendahara"

	if got != want {
		t.Fatalf("buildWhatsAppChatURL() = %q, want %q", got, want)
	}
}

func TestBuildWhatsAppChatURLRejectsInvalidPhone(t *testing.T) {
	if got := buildWhatsAppChatURL("08abc", "Halo"); got != "" {
		t.Fatalf("expected invalid phone to produce empty URL, got %q", got)
	}
}

func TestGatewayRejectionErrorExplainsDisconnectedDevice(t *testing.T) {
	err := gatewayRejectionError(
		`{"reason":"request invalid on disconnected device","status":false}`,
		map[string]interface{}{"reason": "request invalid on disconnected device", "status": false},
	)

	if err == nil || !strings.Contains(err.Error(), "scan QR") {
		t.Fatalf("expected actionable disconnected-device message, got %v", err)
	}
}

func TestGatewayRejectionErrorExplainsInvalidToken(t *testing.T) {
	err := gatewayRejectionError(
		`{"reason":"invalid token","status":false}`,
		map[string]interface{}{"reason": "invalid token", "status": false},
	)

	if err == nil || !strings.Contains(err.Error(), "token gateway WA ditolak") {
		t.Fatalf("expected actionable invalid-token message, got %v", err)
	}
}
