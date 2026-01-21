package repository

import (
	"time"

	"gorm.io/gorm"

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
	return db.CreateInBatches(items, len(items)).Error
}

func (r *ItemRepository) FindAll(db *gorm.DB, query *model.FindAllItemsRequest) ([]entity.Item, int64, error) {
	var items []entity.Item
	if err := db.Scopes(r.FilterItem(query)).Offset((query.Page - 1) * query.Size).Limit(query.Size).Find(&items).Error; err != nil {
		return nil, 0, err
	}

	var total int64
	if err := db.Model(&entity.Item{}).Scopes(r.FilterItem(query)).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	return items, total, nil
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
		if searchQuery := query.SearchQuery; searchQuery != "" {
			searchQuery = "%" + searchQuery + "%"
			tx = tx.Where("code ILIKE ? OR name ILIKE ? OR category ILIKE ?", searchQuery, searchQuery, searchQuery)
		}

		const layout = "2006-01-02"

		if query.StartDate != "" {
			parsedStart, err := time.Parse(layout, query.StartDate)
			if err == nil {
				tx = tx.Where("created_at >= ?", parsedStart)
			}
		}

		if query.EndDate != "" {
			parsedEnd, err := time.Parse(layout, query.EndDate)
			if err == nil {
				endOfDay := time.Date(
					parsedEnd.Year(), parsedEnd.Month(), parsedEnd.Day(),
					23, 59, 59, 999999999,
					parsedEnd.Location(),
				)
				tx = tx.Where("created_at <= ?", endOfDay)
			}
		}

		return tx
	}
}
