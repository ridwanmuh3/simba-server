package util

import (
	"errors"
	"testing"

	"github.com/ridwanmuh3/simba-server/internal/entity"
	"github.com/ridwanmuh3/simba-server/internal/exception"
)

func TestCalculateMAC_EmptyTracks(t *testing.T) {
	updated, finalStock, finalTotal, err := CalculateMAC(10, 100, nil)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if len(updated) != 0 {
		t.Fatalf("len(updated) = %d, want 0", len(updated))
	}
	if finalStock != 10 || finalTotal != 1000 {
		t.Fatalf("final = %v/%v, want 10/1000", finalStock, finalTotal)
	}
}

func TestCalculateMAC_InOutChain(t *testing.T) {
	tracks := []entity.StockTracking{
		{Type: "IN", Amount: 100, UnitPrice: 10},
		{Type: "OUT", Amount: 90, UnitPrice: 10},
	}
	updated, finalStock, finalTotal, err := CalculateMAC(0, 10, tracks)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	// avg cost stays 10, OUT deducts 90*10=900 from 1000
	if updated[0].PreviousStock != 0 || updated[0].NewStock != 100 || updated[0].TotalPrice != 1000 {
		t.Fatalf("IN row = %+v", updated[0])
	}
	if updated[1].PreviousStock != 100 || updated[1].NewStock != 10 || updated[1].TotalPrice != 900 {
		t.Fatalf("OUT row = %+v", updated[1])
	}
	if finalStock != 10 || finalTotal != 100 {
		t.Fatalf("final = %v/%v, want 10/100", finalStock, finalTotal)
	}
	// purity: input untouched
	if tracks[0].NewStock != 0 {
		t.Fatalf("input mutated: %+v", tracks[0])
	}
}

func TestCalculateMAC_RejectsNegativeRunningStock(t *testing.T) {
	tracks := []entity.StockTracking{{Type: "OUT", Amount: 200, UnitPrice: 10}}
	_, _, _, err := CalculateMAC(100, 10, tracks)
	if !errors.Is(err, exception.InsufficientStockError) {
		t.Fatalf("err = %v, want InsufficientStockError", err)
	}
}

func TestCalculateMAC_NoFloatDrift(t *testing.T) {
	// 0.1 * 3 in floats is 0.30000000000000004; scaled integers stay exact.
	tracks := []entity.StockTracking{
		{Type: "IN", Amount: 0.1, UnitPrice: 30000},
		{Type: "IN", Amount: 0.1, UnitPrice: 30000},
		{Type: "IN", Amount: 0.1, UnitPrice: 30000},
	}
	updated, finalStock, finalTotal, err := CalculateMAC(0, 30000, tracks)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if finalStock != 0.3 {
		t.Fatalf("finalStock = %v, want 0.3", finalStock)
	}
	if finalTotal != 9000 {
		t.Fatalf("finalTotal = %v, want 9000", finalTotal)
	}
	if updated[2].NewStock != 0.3 || updated[2].TotalPrice != 3000 {
		t.Fatalf("last row = %+v", updated[2])
	}
}

func TestCalculateMAC_InitialStockCounts(t *testing.T) {
	// initial 50 @ 100 = 5000 carried into chain; OUT 20 deducts avg 100*20
	tracks := []entity.StockTracking{{Type: "OUT", Amount: 20, UnitPrice: 120}}
	updated, finalStock, finalTotal, err := CalculateMAC(50, 100, tracks)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if updated[0].PreviousStock != 50 || updated[0].NewStock != 30 {
		t.Fatalf("row = %+v", updated[0])
	}
	if finalStock != 30 || finalTotal != 3000 {
		t.Fatalf("final = %v/%v, want 30/3000", finalStock, finalTotal)
	}
}
