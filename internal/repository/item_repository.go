package repository

import (
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
		Columns:   []clause.Column{{Name: "name"}},
		DoUpdates: clause.AssignmentColumns([]string{"stock", "unit_price", "modified_by"}),
	}).Create(&items).Error
}

func (r *ItemRepository) FindAll(db *gorm.DB, query *model.FindAllItemsRequest) ([]entity.Item, int64, error) {
	var items []entity.Item
	if err := db.Scopes(r.FilterItem(query)).Offset((query.Page - 1) * query.Size).Limit(query.Size).Order("created_at DESC").Find(&items).Error; err != nil {
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
	if err := db.Preload("Item").Scopes(r.FilterStock(query)).Offset((query.Page - 1) * query.Size).Limit(query.Size).Order("created_at DESC").Find(&stocks).Error; err != nil {
		return nil, 0, err
	}

	var total int64
	if err := db.Model(new(entity.StockTracking)).Scopes(r.FilterStock(query)).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	return stocks, total, nil
}

func (r *ItemRepository) CountByName(db *gorm.DB, name string) (int64, error) {
	var count int64
	err := db.Model(new(entity.Item)).Where("name = ?", name).Count(&count).Error
	return count, err
}

func (r *ItemRepository) CountAll(db *gorm.DB) (int64, error) {
	var count int64
	err := db.Model(new(entity.Item)).Count(&count).Error
	return count, err
}

func (r *ItemRepository) FilterItem(query *model.FindAllItemsRequest) func(tx *gorm.DB) *gorm.DB {
	return func(tx *gorm.DB) *gorm.DB {
		return tx.Scopes(
			SearchScope([]string{"id", "name", "category"}, query.SearchQuery),
			DateRangeScope("created_at", query.StartDate, query.EndDate),
		)
	}
}

func (r *ItemRepository) FilterStock(query *model.FindAllStocksRequest) func(tx *gorm.DB) *gorm.DB {
	return func(tx *gorm.DB) *gorm.DB {
		tx = tx.Joins("LEFT JOIN items AS item ON item.id = stock_tracks.item_id")
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

		return tx
	}
}
