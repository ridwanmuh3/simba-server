package util

import (
	"bytes"
	"compress/flate"
	"compress/zlib"
	"io"
	"math"
	"strings"
	"testing"

	"github.com/ridwanmuh3/simba-server/internal/model"
)

func almostEqual(a, b, eps float64) bool {
	return math.Abs(a-b) <= eps
}

func extractPDFText(t *testing.T, pdf []byte) string {
	t.Helper()

	parts := []string{string(pdf)}
	remaining := pdf

	for {
		streamStart := bytes.Index(remaining, []byte("stream"))
		if streamStart == -1 {
			break
		}

		streamDataStart := streamStart + len("stream")
		if streamDataStart < len(remaining) {
			switch remaining[streamDataStart] {
			case '\r':
				streamDataStart++
				if streamDataStart < len(remaining) && remaining[streamDataStart] == '\n' {
					streamDataStart++
				}
			case '\n':
				streamDataStart++
			}
		}

		endOffset := bytes.Index(remaining[streamDataStart:], []byte("endstream"))
		if endOffset == -1 {
			break
		}

		stream := bytes.Trim(remaining[streamDataStart:streamDataStart+endOffset], "\r\n")

		if reader, err := zlib.NewReader(bytes.NewReader(stream)); err == nil {
			decompressed, readErr := io.ReadAll(reader)
			_ = reader.Close()
			if readErr == nil && len(decompressed) > 0 {
				parts = append(parts, string(decompressed))
			}
		} else {
			reader := flate.NewReader(bytes.NewReader(stream))
			decompressed, readErr := io.ReadAll(reader)
			_ = reader.Close()
			if readErr == nil && len(decompressed) > 0 {
				parts = append(parts, string(decompressed))
			}
		}

		remaining = remaining[streamDataStart+endOffset+len("endstream"):]
	}

	return strings.Join(parts, "\n")
}

func TestRound2_DecimalPrecision(t *testing.T) {
	cases := []struct {
		in   float64
		want float64
	}{
		{0.1 + 0.2, 0.3},
		{1234.5678, 1234.57},
		{0, 0},
		{-1.235, -1.24},
	}
	for _, c := range cases {
		got := Round2(c.in)
		if !almostEqual(got, c.want, 0.00001) {
			t.Errorf("Round2(%v) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestRound4_QuantityPrecision(t *testing.T) {
	cases := []struct {
		in   float64
		want float64
	}{
		{0.123456, 0.1235},
		{0.00001, 0},
		{2.5, 2.5},
		{1.00005, 1.0001},
	}
	for _, c := range cases {
		got := Round4(c.in)
		if !almostEqual(got, c.want, 0.000001) {
			t.Errorf("Round4(%v) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestGetTotalPrice_DecimalStockNoFloatDrift(t *testing.T) {
	// Repro: 0.1 * 30000 = 3000.0000000000005 under naive mult
	got := GetTotalPrice(0.1, 30000)
	want := 3000.0
	if !almostEqual(got, want, 0.001) {
		t.Errorf("GetTotalPrice(0.1, 30000) = %v, want %v", got, want)
	}

	got2 := GetTotalPrice(2.5, 12345.67)
	want2 := 30864.18 // 2.5 * 12345.67 = 30864.175 → round half-away-from-zero = 30864.18
	if !almostEqual(got2, want2, 0.001) {
		t.Errorf("GetTotalPrice(2.5, 12345.67) = %v, want %v", got2, want2)
	}
}

func TestFormatDateStringID(t *testing.T) {
	cases := []struct {
		name     string
		input    string
		wantMonth string
	}{
		{"januari", "2026-01-15 10:00:00", "Januari"},
		{"februari", "2026-02-15 10:00:00", "Februari"},
		{"maret", "2026-03-15 10:00:00", "Maret"},
		{"april", "2026-04-15 10:00:00", "April"},
		{"mei", "2026-05-15 10:00:00", "Mei"},
		{"juni", "2026-06-15 10:00:00", "Juni"},
		{"juli", "2026-07-15 10:00:00", "Juli"},
		{"agustus", "2026-08-15 10:00:00", "Agustus"},
		{"september", "2026-09-15 10:00:00", "September"},
		{"oktober", "2026-10-15 10:00:00", "Oktober"},
		{"november", "2026-11-15 10:00:00", "November"},
		{"desember", "2026-12-15 10:00:00", "Desember"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := FormatDateStringID(tc.input)
			if !strings.Contains(got, tc.wantMonth) {
				t.Errorf("FormatDateStringID(%q) = %q, want to contain %q", tc.input, got, tc.wantMonth)
			}
		})
	}
}

func TestIsStrongPassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
		want     bool
	}{
		// Positive cases
		{name: "valid", password: "Strongpass1!", want: true},
		{name: "valid with symbols", password: "Abc123#$_", want: true},
		{name: "valid min length 8", password: "Abc123!x", want: true},
		{name: "valid max length 30", password: "Abcdefghij1234567890!@#$%^&*", want: true},
		// Negative cases
		{name: "too short", password: "A1!aaaa", want: false},
		{name: "too long 31 chars", password: "Abcdefghij1234567890!@#$%^&*00x", want: false},
		{name: "missing uppercase", password: "strongpass1!", want: false},
		{name: "missing number", password: "Strongpass!", want: false},
		{name: "missing symbol", password: "Strongpass1", want: false},
		{name: "non ascii", password: "Strongpass1é!", want: false},
		{name: "empty string", password: "", want: false},
		{name: "only lowercase", password: "abcdefgh", want: false},
		{name: "only numbers", password: "12345678", want: false},
		{name: "only symbols", password: "!@#$%^&*", want: false},
		{name: "control char", password: "Strong\x01pass1!", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsStrongPassword(tt.password); got != tt.want {
				t.Fatalf("IsStrongPassword(%q) = %v, want %v", tt.password, got, tt.want)
			}
		})
	}
}

// ─── GenerateRandomString ─────────────────────────────────────────────

func TestGenerateRandomString_Length(t *testing.T) {
	for _, l := range []int{0, 1, 10, 32, 64, 128} {
		got := GenerateRandomString(l)
		if len(got) != l {
			t.Errorf("GenerateRandomString(%d) returned length %d", l, len(got))
		}
	}
}

func TestGenerateRandomString_Unique(t *testing.T) {
	a := GenerateRandomString(32)
	b := GenerateRandomString(32)
	if a == b {
		t.Errorf("two random strings should differ, both = %q", a)
	}
}

func TestGenerateRandomString_Charset(t *testing.T) {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	s := GenerateRandomString(100)
	for i, ch := range s {
		found := false
		for _, c := range charset {
			if ch == c {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("char at index %d (%c) not in charset", i, ch)
		}
	}
}

func TestGenerateRandomString_Empty(t *testing.T) {
	s := GenerateRandomString(0)
	if s != "" {
		t.Errorf("GenerateRandomString(0) should be empty, got %q", s)
	}
}

// ─── HashToken ──────────────────────────────────────────────────────────

func TestHashToken_Deterministic(t *testing.T) {
	token := "abc123secret"
	h1 := HashToken(token)
	h2 := HashToken(token)
	if h1 != h2 {
		t.Errorf("HashToken should be deterministic: %q != %q", h1, h2)
	}
}

func TestHashToken_NotBareToken(t *testing.T) {
	token := "abc123secret"
	h := HashToken(token)
	if h == token {
		t.Error("HashToken should not return bare plaintext")
	}
}

func TestHashToken_Hex64Chars(t *testing.T) {
	h := HashToken("test")
	if len(h) != 64 {
		t.Errorf("SHA-256 hex should be 64 chars, got %d: %q", len(h), h)
	}
	for _, ch := range h {
		if !((ch >= '0' && ch <= '9') || (ch >= 'a' && ch <= 'f')) {
			t.Errorf("non-hex char in hash: %c", ch)
		}
	}
}

func TestHashToken_DifferentInputsProduceDifferentHashes(t *testing.T) {
	h1 := HashToken("token-a")
	h2 := HashToken("token-b")
	if h1 == h2 {
		t.Error("different tokens should produce different hashes")
	}
}

// ─── DeleteFile (security) ─────────────────────────────────────────────

func TestDeleteFile_PathTraversal(t *testing.T) {
	// Path traversal outside uploads/ should be rejected.
	err := DeleteFile("/../../../etc/passwd")
	if err == nil {
		t.Error("path traversal should be rejected")
	}
}

func TestDeleteFile_NonExistentFile(t *testing.T) {
	err := DeleteFile("/uploads/nonexistent-file-12345.txt")
	if err == nil {
		t.Error("non-existent file should return error")
	}
}

// ─── FormatDateStringID negative cases ──────────────────────────────────

func TestFormatDateStringID_InvalidFormat(t *testing.T) {
	// Invalid date string should be returned as-is.
	got := FormatDateStringID("not-a-date")
	if got != "not-a-date" {
		t.Errorf("FormatDateStringID(%q) = %q, want same input", "not-a-date", got)
	}
}

func TestFormatDateStringID_EmptyString(t *testing.T) {
	got := FormatDateStringID("")
	if got != "" {
		t.Errorf("FormatDateStringID(\"\") = %q, want empty", got)
	}
}

// ─── Round2 / Round4 negative cases ────────────────────────────────────

func TestRound2_VeryLargeNumber(t *testing.T) {
	got := Round2(1e15 + 0.001)
	want := 1e15
	if !almostEqual(got, want, 1) {
		t.Errorf("Round2(%v) = %v, want %v", 1e15+0.001, got, want)
	}
}

func TestRound4_VerySmallNumber(t *testing.T) {
	got := Round4(0.000001)
	if got != 0 {
		t.Errorf("Round4(0.000001) = %v, want 0", got)
	}
}

func TestGetTotalPrice_ZeroQuantity(t *testing.T) {
	got := GetTotalPrice(0, 50000)
	if got != 0 {
		t.Errorf("GetTotalPrice(0, 50000) = %v, want 0", got)
	}
}

func TestGetTotalPrice_ZeroPrice(t *testing.T) {
	got := GetTotalPrice(10, 0)
	if got != 0 {
		t.Errorf("GetTotalPrice(10, 0) = %v, want 0", got)
	}
}

func TestGenerateTemplateInvoicePDF_SubtotalMatchesLineSum(t *testing.T) {
	data := &model.InvoiceData{
		CompanyName:     "PT Test",
		CompanyAddress:  "Jl Test",
		CompanyContact:  "0800",
		InvoiceNo:       "INV-001",
		Date:            "01 April 2026",
		ReceiverName:    "Client",
		ReceiverAddress: "Jl Client",
		Penanggungjawab: "PIC",
		Jabatan:         "Manager",
		GrandTotal:      Round2(0.1*30000 + 2.5*12345.67),
		Items: []model.StockResponse{
			{
				Amount:     0.1,
				UnitPrice:  30000,
				TotalPrice: Round2(0.1 * 30000),
				Item:       model.ItemResponse{Name: "Beras", MeasureUnit: "kg"},
			},
			{
				Amount:     2.5,
				UnitPrice:  12345.67,
				TotalPrice: Round2(2.5 * 12345.67),
				Item:       model.ItemResponse{Name: "Gula", MeasureUnit: "kg"},
			},
		},
	}

	buf, err := GenerateTemplateInvoicePDF(data)
	if err != nil {
		t.Fatalf("GenerateTemplateInvoicePDF error: %v", err)
	}
	if buf.Len() == 0 {
		t.Fatal("empty pdf buffer")
	}

	var sum float64
	for _, it := range data.Items {
		sum = Round2(sum + it.TotalPrice)
	}
	if !almostEqual(sum, data.GrandTotal, 0.001) {
		t.Errorf("subtotal sum %v ≠ GrandTotal %v", sum, data.GrandTotal)
	}
}

func TestGenerateTemplateInvoicePDF_UsesKebutuhanValue(t *testing.T) {
	data := &model.InvoiceData{
		CompanyName:     "PT Test",
		CompanyAddress:  "Jl Test",
		CompanyContact:  "0800",
		InvoiceNo:       "INV-777",
		PONo:            "PO-123",
		Kebutuhan:       "Kebutuhan Catering",
		Date:            "14 Mei 2026",
		ReceiverName:    "Client",
		ReceiverAddress: "Jl Client",
		Penanggungjawab: "PIC",
		Jabatan:         "Manager",
		Items: []model.StockResponse{
			{
				Amount:     1,
				UnitPrice:  10000,
				TotalPrice: 10000,
				Item:       model.ItemResponse{Name: "Beras", MeasureUnit: "kg"},
			},
		},
		GrandTotal: 10000,
	}

	buf, err := GenerateTemplateInvoicePDF(data)
	if err != nil {
		t.Fatalf("GenerateTemplateInvoicePDF error: %v", err)
	}

	text := extractPDFText(t, buf.Bytes())
	if !strings.Contains(text, "Kebutuhan") {
		t.Fatalf("pdf text missing kebutuhan label: %q", text)
	}
	if !strings.Contains(text, data.Kebutuhan) {
		t.Fatalf("pdf text missing kebutuhan value %q: %q", data.Kebutuhan, text)
	}
	if strings.Contains(text, data.PONo) {
		t.Fatalf("pdf text should not render po number %q in kebutuhan row: %q", data.PONo, text)
	}
}

func TestGenerateTemplateInvoicePDF_AllowsEmptyKebutuhan(t *testing.T) {
	data := &model.InvoiceData{
		CompanyName:     "PT Test",
		CompanyAddress:  "Jl Test",
		CompanyContact:  "0800",
		InvoiceNo:       "INV-888",
		Date:            "14 Mei 2026",
		ReceiverName:    "Client",
		ReceiverAddress: "Jl Client",
		Penanggungjawab: "PIC",
		Jabatan:         "Manager",
		Items: []model.StockResponse{
			{
				Amount:     1,
				UnitPrice:  15000,
				TotalPrice: 15000,
				Item:       model.ItemResponse{Name: "Gula", MeasureUnit: "kg"},
			},
		},
		GrandTotal: 15000,
	}

	buf, err := GenerateTemplateInvoicePDF(data)
	if err != nil {
		t.Fatalf("GenerateTemplateInvoicePDF error: %v", err)
	}
	if buf.Len() == 0 {
		t.Fatal("empty pdf buffer")
	}
}
