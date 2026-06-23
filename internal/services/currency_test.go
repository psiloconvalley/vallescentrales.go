// internal/services/currency_test.go
// Tests for currency formatting and conversion.
// No DB required — tests pure logic only.

package services

import (
	"testing"
)

// ─── FormatMXN ───────────────────────────────────────────────────────────────

func TestFormatMXN(t *testing.T) {
	tests := []struct {
		amount   float64
		expected string
	}{
		{0, "$0 MXN"},
		{1000, "$1,000 MXN"},
		{50000, "$50,000 MXN"},
		{1500000, "$1,500,000 MXN"},
		{25000000, "$25,000,000 MXN"},
		{999, "$999 MXN"},
		{1500000.75, "$1,500,001 MXN"}, // rounds to nearest whole
	}

	for _, tt := range tests {
		got := FormatMXN(tt.amount)
		if got != tt.expected {
			t.Errorf("FormatMXN(%f) = %q, want %q", tt.amount, got, tt.expected)
		}
	}
}

// ─── FormatUSD ───────────────────────────────────────────────────────────────

func TestFormatUSD(t *testing.T) {
	tests := []struct {
		amount   float64
		expected string
	}{
		{0, "$0 USD"},
		{1000, "$1,000 USD"},
		{85714, "$85,714 USD"},
		{1250000, "$1,250,000 USD"},
	}

	for _, tt := range tests {
		got := FormatUSD(tt.amount)
		if got != tt.expected {
			t.Errorf("FormatUSD(%f) = %q, want %q", tt.amount, got, tt.expected)
		}
	}
}

// ─── formatNumber ────────────────────────────────────────────────────────────

func TestFormatNumber(t *testing.T) {
	tests := []struct {
		n        float64
		decimals int
		expected string
	}{
		{0, 0, "0"},
		{999, 0, "999"},
		{1000, 0, "1,000"},
		{10000, 0, "10,000"},
		{100000, 0, "100,000"},
		{1000000, 0, "1,000,000"},
		{1234567890, 0, "1,234,567,890"},
		{1234.56, 2, "1,234.56"},
	}

	for _, tt := range tests {
		got := formatNumber(tt.n, tt.decimals)
		if got != tt.expected {
			t.Errorf("formatNumber(%f, %d) = %q, want %q", tt.n, tt.decimals, got, tt.expected)
		}
	}
}

// ─── ConvertMXNToUSD ─────────────────────────────────────────────────────────

func TestConvertMXNToUSD(t *testing.T) {
	cs := &CurrencyService{
		rate: 0.058, // approximate rate
	}

	tests := []struct {
		mxn      float64
		expected float64
	}{
		{0, 0},
		{1000000, 58000},        // 1M MXN ≈ $58K USD
		{1500000, 87000},        // 1.5M MXN ≈ $87K USD
		{500000, 29000},
	}

	for _, tt := range tests {
		got := cs.ConvertMXNToUSD(tt.mxn)
		if got != tt.expected {
			t.Errorf("ConvertMXNToUSD(%f) = %f, want %f", tt.mxn, got, tt.expected)
		}
	}
}

// ─── Rate ────────────────────────────────────────────────────────────────────

func TestCurrencyService_Rate_ReturnsFallback(t *testing.T) {
	cs := &CurrencyService{
		rate: fallbackMXNtoUSD,
	}

	rate := cs.Rate()
	if rate != fallbackMXNtoUSD {
		t.Errorf("expected fallback rate %f, got %f", fallbackMXNtoUSD, rate)
	}
	if rate <= 0 {
		t.Error("fallback rate must be positive")
	}
}
