package integration_test

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/go-playground/validator/v10"

	"github.com/ridwanmuh3/simba-server/internal/entity"
	"github.com/ridwanmuh3/simba-server/internal/exception"
	"github.com/ridwanmuh3/simba-server/internal/model"
	"github.com/ridwanmuh3/simba-server/internal/repository"
	"github.com/ridwanmuh3/simba-server/internal/service"
)

type fakeEmitter struct{}

func (f *fakeEmitter) Emit(_ *entity.ActivityLog) {}

func newStockSvc(t *testing.T) *service.StockService {
	t.Helper()
	validate := validator.New()
	return service.NewStockService(testDB, testLog, validate, repository.NewItemRepository(), &fakeEmitter{}, nil)
}

func seedItem(t *testing.T, id string, initialStock float64) {
	t.Helper()
	item := entity.Item{
		ID:           id,
		Name:         "RaceItem-" + id,
		Category:     "RaceCat",
		InitialStock: initialStock,
		Stock:        initialStock,
		MeasureUnit:  "pcs",
		UnitPrice:    10,
		TotalPrice:   initialStock * 10,
		ModifiedBy:   "tester",
		DapurID:      testDapurID,
	}
	if err := testDB.Create(&item).Error; err != nil {
		t.Fatalf("seed item %s: %v", id, err)
	}
	t.Cleanup(func() {
		testDB.Exec("DELETE FROM stock_tracks WHERE item_id = ?", id)
		testDB.Exec("DELETE FROM items WHERE id = ?", id)
	})
}

func TestStockOut_Concurrent(t *testing.T) {
	seedItem(t, "MBG-BHN-90001", 100)
	svc := newStockSvc(t)
	ctx := context.Background()

	var wg sync.WaitGroup
	var ok, insufficient atomic.Int32
	var other atomic.Int32
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := svc.UpdateStock(ctx, &model.UpdateItemStockRequest{
				ID:         "MBG-BHN-90001",
				Type:       "OUT",
				Amount:     5,
				UnitPrice:  10,
				Supplier:   "-",
				ModifiedBy: "tester",
				DapurID:    testDapurID,
			})
			switch {
			case err == nil:
				ok.Add(1)
			case errors.Is(err, exception.InsufficientStockError):
				insufficient.Add(1)
			default:
				other.Add(1)
				t.Logf("unexpected error: %v", err)
			}
		}()
	}
	wg.Wait()

	if got := ok.Load(); got != 20 {
		t.Fatalf("ok = %d, want 20 (insufficient=%d other=%d)", got, insufficient.Load(), other.Load())
	}
	if got := insufficient.Load(); got != 30 {
		t.Fatalf("insufficient = %d, want 30 (ok=%d other=%d)", got, ok.Load(), other.Load())
	}
	if other.Load() != 0 {
		t.Fatalf("other errors = %d, want 0", other.Load())
	}

	var item entity.Item
	if err := testDB.Where("id = ? AND dapur_id = ?", "MBG-BHN-90001", testDapurID).First(&item).Error; err != nil {
		t.Fatalf("load item: %v", err)
	}
	if item.Stock != 0 {
		t.Fatalf("final stock = %v, want 0", item.Stock)
	}

	var negCount int64
	if err := testDB.Model(&entity.StockTracking{}).
		Where("item_id = ? AND new_stock < 0", "MBG-BHN-90001").
		Count(&negCount).Error; err != nil {
		t.Fatalf("count negative new_stock: %v", err)
	}
	if negCount != 0 {
		t.Fatalf("rows with new_stock < 0 = %d, want 0", negCount)
	}
}

func TestStockEdit_NegativeChainRejected(t *testing.T) {
	seedItem(t, "MBG-BHN-90002", 0)
	svc := newStockSvc(t)
	ctx := context.Background()

	// IN 100
	if _, err := svc.UpdateStock(ctx, &model.UpdateItemStockRequest{
		ID: "MBG-BHN-90002", Type: "IN", Amount: 100, UnitPrice: 10,
		Supplier: "-", ModifiedBy: "tester", DapurID: testDapurID,
	}); err != nil {
		t.Fatalf("IN 100: %v", err)
	}
	// OUT 90
	if _, err := svc.UpdateStock(ctx, &model.UpdateItemStockRequest{
		ID: "MBG-BHN-90002", Type: "OUT", Amount: 90, UnitPrice: 10,
		Supplier: "-", ModifiedBy: "tester", DapurID: testDapurID,
	}); err != nil {
		t.Fatalf("OUT 90: %v", err)
	}

	var outTrack entity.StockTracking
	if err := testDB.Where("item_id = ? AND type = ?", "MBG-BHN-90002", "OUT").First(&outTrack).Error; err != nil {
		t.Fatalf("load OUT track: %v", err)
	}

	// Edit OUT 90 -> 200 would make chain negative (100-200 < 0)
	_, err := svc.EditStock(ctx, &model.EditStockRequest{
		ID: "MBG-BHN-90002", StockID: int(outTrack.ID), Amount: 200, UnitPrice: 10,
		Supplier: "-", ModifiedBy: "tester", DapurID: testDapurID,
	})
	if !errors.Is(err, exception.InsufficientStockError) {
		t.Fatalf("edit error = %v, want InsufficientStockError", err)
	}

	var item entity.Item
	if err := testDB.Where("id = ? AND dapur_id = ?", "MBG-BHN-90002", testDapurID).First(&item).Error; err != nil {
		t.Fatalf("load item: %v", err)
	}
	if item.Stock != 10 {
		t.Fatalf("stock after rejected edit = %v, want 10", item.Stock)
	}

	var after entity.StockTracking
	if err := testDB.Where("id = ?", outTrack.ID).First(&after).Error; err != nil {
		t.Fatalf("load OUT track after: %v", err)
	}
	if after.Amount != 90 {
		t.Fatalf("OUT amount after rejected edit = %v, want 90", after.Amount)
	}
}

func TestItemUpdate_InitialStockNegativeChainRejected(t *testing.T) {
	seedItem(t, "MBG-BHN-90003", 100)
	svc := newStockSvc(t)
	ctx := context.Background()

	// OUT 90 leaves stock 10
	if _, err := svc.UpdateStock(ctx, &model.UpdateItemStockRequest{
		ID: "MBG-BHN-90003", Type: "OUT", Amount: 90, UnitPrice: 10,
		Supplier: "-", ModifiedBy: "tester", DapurID: testDapurID,
	}); err != nil {
		t.Fatalf("OUT 90: %v", err)
	}

	validate := validator.New()
	itemSvc := service.NewItemService(testDB, testLog, validate, repository.NewItemRepository())
	newInitial := float64(50) // running 50 - 90 < 0
	_, err := itemSvc.Update(ctx, &model.UpdateItemRequest{
		ID:           "MBG-BHN-90003",
		Name:         "RaceItem-MBG-BHN-90003",
		Category:     "RaceCat",
		MeasureUnit:  "pcs",
		UnitPrice:    10,
		InitialStock: &newInitial,
		ModifiedBy:   "tester",
		DapurID:      testDapurID,
	})
	if !errors.Is(err, exception.InsufficientStockError) {
		t.Fatalf("update error = %v, want InsufficientStockError", err)
	}

	var item entity.Item
	if err := testDB.Where("id = ? AND dapur_id = ?", "MBG-BHN-90003", testDapurID).First(&item).Error; err != nil {
		t.Fatalf("load item: %v", err)
	}
	if item.InitialStock != 100 {
		t.Fatalf("initial stock = %v, want 100 (unchanged)", item.InitialStock)
	}
	if item.Stock != 10 {
		t.Fatalf("stock = %v, want 10 (unchanged)", item.Stock)
	}
}
