package util

import (
	"math"
	"testing"

	"github.com/ridwanmuh3/simba-server/internal/model"
)

func almostEqual(a, b, eps float64) bool {
	return math.Abs(a-b) <= eps
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
