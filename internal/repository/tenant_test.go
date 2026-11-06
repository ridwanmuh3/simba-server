package repository

import (
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/ridwanmuh3/simba-server/internal/entity"
	"github.com/ridwanmuh3/simba-server/internal/model"
)

func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&entity.Item{}, &entity.StockTracking{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

// Tenant A must not see, update, or delete tenant B's rows.
func TestTenantScope_IsolatesDapur(t *testing.T) {
	db := newTestDB(t)
	repo := NewItemRepository()

	for _, it := range []entity.Item{
		{ID: "A1", Name: "Beras-A", Category: "c", Stock: 10, InitialStock: 10, UnitPrice: 100, TotalPrice: 1000, ModifiedBy: "t", DapurID: 1},
		{ID: "B1", Name: "Beras-B", Category: "c", Stock: 20, InitialStock: 20, UnitPrice: 100, TotalPrice: 2000, ModifiedBy: "t", DapurID: 2},
	} {
		if err := db.Create(&it).Error; err != nil {
			t.Fatalf("seed: %v", err)
		}
	}

	// read: tenant 1 sees only its own item
	got, err := repo.FindItemByIDTenant(db, "B1", 1)
	if err == nil {
		t.Fatalf("tenant 1 read tenant 2 item: %+v", got)
	}
	got, err = repo.FindItemByIDTenant(db, "A1", 1)
	if err != nil || got == nil {
		t.Fatalf("tenant 1 read own item: %v %+v", err, got)
	}

	// locked read (update path) is scoped too
	if _, err := repo.FindItemForUpdate(db, "B1", 1); err == nil {
		t.Fatal("tenant 1 locked-read tenant 2 item")
	}

	// FindAll scoped via Tenant in FilterItem
	items, total, err := repo.FindAll(db, &model.FindAllItemsRequest{Page: 1, Size: 10, DapurID: 1})
	if err != nil {
		t.Fatalf("find all: %v", err)
	}
	if total != 1 || len(items) != 1 || items[0].ID != "A1" {
		t.Fatalf("tenant 1 findall = %d/%v, want only A1", total, items)
	}
	var count int64
	if err := db.Model(new(entity.Item)).Scopes(Tenant(1)).Count(&count).Error; err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 1 {
		t.Fatalf("tenant 1 count = %d, want 1", count)
	}

	// delete via tenant scope cannot remove other tenant's row
	if err := db.Scopes(Tenant(1)).Where("id = ?", "B1").Delete(&entity.Item{}).Error; err != nil {
		t.Fatalf("scoped delete: %v", err)
	}
	var b entity.Item
	if err := db.Where("id = ?", "B1").First(&b).Error; err != nil {
		t.Fatalf("tenant 2 item deleted by tenant 1 scope: %v", err)
	}
}

func TestTenantScope_ZeroDapurIsUnscoped(t *testing.T) {
	db := newTestDB(t)
	var count int64
	if err := db.Model(new(entity.Item)).Scopes(Tenant(0)).Count(&count).Error; err != nil {
		t.Fatalf("count: %v", err)
	}
}
