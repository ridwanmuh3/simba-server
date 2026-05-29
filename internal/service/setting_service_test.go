package service

import "testing"

func TestParseDocumentSequence(t *testing.T) {
	tests := []struct {
		name       string
		value      string
		prefix     string
		wantSeq    int
		wantParsed bool
	}{
		{"invoice padded", "INV-009", "INV", 9, true},
		{"po with extra segment", "PO-2026-012", "PO", 12, true},
		{"case insensitive prefix", "inv-010", "INV", 10, true},
		{"wrong prefix", "PO-010", "INV", 0, false},
		{"empty value", "", "INV", 0, false},
		{"missing separator", "INV010", "INV", 0, false},
		{"bad number", "INV-ABC", "INV", 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotSeq, gotParsed := parseDocumentSequence(tt.value, tt.prefix)
			if gotSeq != tt.wantSeq || gotParsed != tt.wantParsed {
				t.Fatalf("parseDocumentSequence(%q, %q) = (%d, %v), want (%d, %v)",
					tt.value, tt.prefix, gotSeq, gotParsed, tt.wantSeq, tt.wantParsed)
			}
		})
	}
}
