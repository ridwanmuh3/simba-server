package repository

import (
	"errors"
	"slices"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/ridwanmuh3/simba-server/internal/entity"
	"github.com/ridwanmuh3/simba-server/internal/model"
)

type ItemRepository struct {
	Repository[entity.Item]
}

func NewItemRepository() *ItemRepository {
	return &ItemRepository{}
}

func (r *ItemRepository) AddBatches(db *gorm.DB, items []entity.Item) error {
	return db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "name"}, {Name: "dapur_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"category",
			"initial_stock",
			"stock",
			"measure_unit",
			"unit_price",
			"total_price",
			"modified_by",
			"updated_at",
			"deleted_at",
		}),
	}).Create(&items).Error
}

func (r *ItemRepository) FindAll(db *gorm.DB, query *model.FindAllItemsRequest) ([]entity.Item, int64, error) {
	var items []entity.Item

	if err := db.Scopes(r.FilterItem(query)).Offset((query.Page - 1) * query.Size).Limit(query.Size).Order(OrderBy()).Find(&items).Error; err != nil {
		return nil, 0, err
	}

	var total int64
	if err := db.Model(new(entity.Item)).Scopes(r.FilterItem(query)).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	return items, total, nil
}

func (r *ItemRepository) FindAllStocks(db *gorm.DB, query *model.FindAllStocksRequest) ([]entity.StockTracking, int64, error) {
	var stocks []entity.StockTracking
	if err := db.Preload("Item").Scopes(r.FilterStock(query)).Offset((query.Page - 1) * query.Size).Limit(query.Size).Order(OrderBy()).Find(&stocks).Error; err != nil {
		return nil, 0, err
	}

	var total int64
	if err := db.Model(new(entity.StockTracking)).Scopes(r.FilterStock(query)).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	return stocks, total, nil
}

func (r *ItemRepository) FindLastStockPrice(db *gorm.DB, itemID string, stockType string, dapurID uint) (float64, error) {
	var stock entity.StockTracking
	err := db.Select("unit_price").
		Where("item_id = ? AND type = ? AND dapur_id = ?", itemID, stockType, dapurID).
		Order("created_at DESC").
		Take(&stock).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, nil
	}
	return stock.UnitPrice, err
}

func (r *ItemRepository) CountByName(db *gorm.DB, name string, dapurID uint) (int64, error) {
	var count int64
	err := db.Model(new(entity.Item)).Where("name = ? AND dapur_id = ?", name, dapurID).Count(&count).Error
	return count, err
}

func (r *ItemRepository) CountAll(db *gorm.DB) (int64, error) {
	var count int64
	err := db.Model(new(entity.Item)).Count(&count).Error
	return count, err
}

func (r *ItemRepository) FilterItem(query *model.FindAllItemsRequest) func(tx *gorm.DB) *gorm.DB {
	return func(tx *gorm.DB) *gorm.DB {
		if query.DapurID > 0 {
			tx = tx.Where("dapur_id = ?", query.DapurID)
		}
		return tx.Scopes(
			SearchScope([]string{"id", "name", "category"}, query.SearchQuery),
			DateRangeScope("created_at", query.StartDate, query.EndDate),
		)
	}
}

func (r *ItemRepository) FilterStock(query *model.FindAllStocksRequest) func(tx *gorm.DB) *gorm.DB {
	return func(tx *gorm.DB) *gorm.DB {
		tx = tx.Joins("LEFT JOIN items AS item ON item.id = stock_tracks.item_id")
		if query.DapurID > 0 {
			tx = tx.Where("stock_tracks.dapur_id = ?", query.DapurID)
		}
		tx = tx.Scopes(
			SearchScope(
				[]string{"stock_tracks.item_id", "item.name", "item.category", "stock_tracks.supplier"},
				query.SearchQuery,
			),
			DateRangeScope("stock_tracks.created_at", query.StartDate, query.EndDate),
		)

		if slices.Contains([]string{"IN", "OUT"}, query.Type) {
			tx = tx.Where("stock_tracks.type = ?", query.Type)
		}

		if query.Category != "" {
			tx = tx.Where("item.category = ?", query.Category)
		}

		return tx
	}
}
