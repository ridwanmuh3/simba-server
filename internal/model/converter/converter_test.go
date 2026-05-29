package converter

import (
	"testing"
	"time"

	"gorm.io/gorm"

	"github.com/ridwanmuh3/simba-server/internal/entity"
)

func TestUserToResponse(t *testing.T) {
	now := time.Now()
	user := &entity.User{
		Model:      gorm.Model{ID: 11, CreatedAt: now},
		Username:   "tester01",
		Fullname:   "Tester One",
		Role:       "Admin",
		IsActive:   true,
		LastActive: now,
	}

	got := UserToResponse(user)
	if got.ID != 11 || got.Username != user.Username || got.Fullname != user.Fullname || got.Role != user.Role {
		t.Fatalf("response mismatch: %+v", got)
	}
	if !got.IsActive || !got.CreatedAt.Equal(now) || !got.LastActive.Equal(now) {
		t.Fatalf("time/status mismatch: %+v", got)
	}
}

func TestItemAndStockToResponse(t *testing.T) {
	now := time.Now()
	item := entity.Item{
		ID:           "MBG-BHN-0001",
		Name:         "Beras",
		Category:     "Pokok",
		Stock:        5,
		InitialStock: 10,
		MeasureUnit:  "Kg",
		UnitPrice:    12000,
		TotalPrice:   60000,
		CreatedAt:    now,
	}

	itemResp := ItemToResponse(&item)
	if itemResp.ID != item.ID || itemResp.TotalPrice != item.TotalPrice || !itemResp.CreatedAt.Equal(now) {
		t.Fatalf("item response mismatch: %+v", itemResp)
	}

	stock := entity.StockTracking{
		Model:         gorm.Model{ID: 9, CreatedAt: now},
		Type:          "IN",
		Amount:        3,
		PreviousStock: 5,
		NewStock:      8,
		UnitPrice:     10000,
		TotalPrice:    30000,
		Supplier:      "Supplier",
		ModifiedBy:    "Admin",
		Item:          item,
	}

	stockResp := StockToResponse(&stock)
	if stockResp.ID != 9 || stockResp.Item.ID != item.ID || stockResp.TotalPrice != stock.TotalPrice {
		t.Fatalf("stock response mismatch: %+v", stockResp)
	}
}

func TestFinanceToResponse(t *testing.T) {
	now := time.Now()
	finance := &entity.Finance{
		Model:       gorm.Model{ID: 4, CreatedAt: now},
		Type:        "PEMASUKAN",
		Category:    "Penjualan",
		Description: "Harian",
		Amount:      1000,
		ExtraNote:   "note",
		ProofImage:  "/uploads/proof.png",
		ModifiedBy:  "Admin",
	}

	got := FinanceToResponse(finance)
	if got.ID != 4 || got.Type != finance.Type || got.Amount != finance.Amount || got.ProofImage != finance.ProofImage {
		t.Fatalf("finance response mismatch: %+v", got)
	}
	if !got.CreatedAt.Equal(now) {
		t.Fatalf("CreatedAt = %v, want %v", got.CreatedAt, now)
	}
}
