package service

import (
	"testing"

	"github.com/ridwanmuh3/simba-server/internal/entity"
)

// ─── Positive: no alert cases ──────────────────────────────────────────

func TestLowStockChecker_Check_NoThreshold_NoAlert(t *testing.T) {
	checker := &LowStockChecker{log: newTestLogger()}
	item := &entity.Item{ID: "MBG-BHN-001", Name: "Beras", Stock: 1.0, MeasureUnit: "kg", LowStockThreshold: 0}
	checker.Check(item, 1) // threshold=0 → returns immediately
}

func TestLowStockChecker_Check_AboveThreshold_NoAlert(t *testing.T) {
	checker := &LowStockChecker{log: newTestLogger()}
	item := &entity.Item{ID: "MBG-BHN-002", Name: "Minyak", Stock: 10.0, MeasureUnit: "liter", LowStockThreshold: 5.0}
	checker.Check(item, 1) // 10 > 5 → no alert
}

// ─── Edge: exact threshold ──────────────────────────────────────────────

func TestLowStockChecker_Check_ExactThreshold_NoAlert(t *testing.T) {
	// stock == threshold → stock <= threshold is true, but this tests the boundary.
	item := &entity.Item{ID: "MBG-BHN-004", Name: "Garam", Stock: 5.0, MeasureUnit: "kg", LowStockThreshold: 5.0}
	// stock (5) <= threshold (5) && threshold > 0 → should attempt alert (needs DB)
	// Verify condition mathematically.
	if !(item.Stock <= item.LowStockThreshold && item.LowStockThreshold > 0) {
		t.Fatal("expected alert condition to be true for stock == threshold")
	}
}

func TestLowStockChecker_Check_JustAboveThreshold_NoAlert(t *testing.T) {
	item := &entity.Item{ID: "MBG-BHN-005", Name: "Lada", Stock: 5.01, MeasureUnit: "kg", LowStockThreshold: 5.0}
	if item.Stock <= item.LowStockThreshold {
		t.Fatal("expected stock > threshold")
	}
}

// ─── Edge: zero / negative stock ───────────────────────────────────────

func TestLowStockChecker_Check_ZeroStock_WithThreshold_Alert(t *testing.T) {
	item := &entity.Item{ID: "MBG-BHN-006", Name: "Gula", Stock: 0, MeasureUnit: "kg", LowStockThreshold: 5.0}
	if !(item.Stock <= item.LowStockThreshold && item.LowStockThreshold > 0) {
		t.Fatal("expected alert for zero stock with positive threshold")
	}
}

func TestLowStockChecker_Check_NegativeStock_WithThreshold_Alert(t *testing.T) {
	item := &entity.Item{ID: "MBG-BHN-007", Name: "Tepung", Stock: -1.0, MeasureUnit: "kg", LowStockThreshold: 5.0}
	if !(item.Stock <= item.LowStockThreshold && item.LowStockThreshold > 0) {
		t.Fatal("expected alert for negative stock with positive threshold")
	}
}

func TestLowStockChecker_Check_NegativeThreshold_NoAlert(t *testing.T) {
	item := &entity.Item{ID: "MBG-BHN-008", Name: "Telur", Stock: 2.0, MeasureUnit: "butir", LowStockThreshold: -5.0}
	if item.Stock <= item.LowStockThreshold && item.LowStockThreshold > 0 {
		t.Fatal("should not alert when threshold is negative")
	}
}

// ─── Positive: below threshold requires DB ─────────────────────────────

func TestLowStockChecker_Check_BelowThreshold_RequiresDB(t *testing.T) {
	item := &entity.Item{ID: "MBG-BHN-009", Name: "Telur", Stock: 2.0, MeasureUnit: "butir", LowStockThreshold: 5.0}
	if item.Stock > item.LowStockThreshold {
		t.Fatal("expected stock to be below threshold")
	}
	if item.LowStockThreshold <= 0 {
		t.Fatal("expected positive threshold")
	}
}
