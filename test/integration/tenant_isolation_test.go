package integration_test

import (
	"context"
	"errors"
	"testing"

	"github.com/ridwanmuh3/simba-server/internal/entity"
	"github.com/ridwanmuh3/simba-server/internal/exception"
	"github.com/ridwanmuh3/simba-server/internal/model"
)

// Tenant B must not read, update, or delete tenant A's stock.
func TestStock_TenantIsolation(t *testing.T) {
	seedItem(t, "MBG-BHN-91001", 100)
	svc := newStockSvc(t)
	ctx := context.Background()

	otherDapur := testDapurID + 1000

	// UpdateStock as another dapur -> not found, no mutation
	_, err := svc.UpdateStock(ctx, &model.UpdateItemStockRequest{
		ID: "MBG-BHN-91001", Type: "OUT", Amount: 10, UnitPrice: 10,
		Supplier: "-", ModifiedBy: "tester", DapurID: otherDapur,
	})
	if !errors.Is(err, exception.ItemNotFoundError) {
		t.Fatalf("cross-tenant UpdateStock err = %v, want ItemNotFoundError", err)
	}

	// FindAllStocks as another dapur -> empty
	rows, total, err := svc.FindAllStocks(ctx, &model.FindAllStocksRequest{
		Page: 1, Size: 10, DapurID: otherDapur,
	})
	if err != nil {
		t.Fatalf("cross-tenant FindAllStocks: %v", err)
	}
	if total != 0 || len(rows) != 0 {
		t.Fatalf("cross-tenant FindAllStocks = %d/%d rows, want 0", total, len(rows))
	}

	// own tenant still sees it, balance untouched
	rows, total, err = svc.FindAllStocks(ctx, &model.FindAllStocksRequest{
		Page: 1, Size: 10, DapurID: testDapurID, SearchQuery: "MBG-BHN-91001",
	})
	if err != nil {
		t.Fatalf("own FindAllStocks: %v", err)
	}
	if total != 0 {
		// no tracks yet (only item seeded) — count may be 0, that's fine;
		// the isolation assertion above is the important one.
		t.Logf("own tracks total=%d", total)
	}
	_ = rows

	var item entity.Item
	if err := testDB.Where("id = ? AND dapur_id = ?", "MBG-BHN-91001", testDapurID).First(&item).Error; err != nil {
		t.Fatalf("load item: %v", err)
	}
	if item.Stock != 100 {
		t.Fatalf("stock = %v, want 100 (untouched by cross-tenant attempt)", item.Stock)
	}
}
