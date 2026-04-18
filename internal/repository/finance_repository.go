package repository

import (
	"github.com/ridwanmuh3/simba-server/internal/entity"
	"github.com/ridwanmuh3/simba-server/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type FinanceRepository struct {
	Repository[entity.Finance]
}

func NewFinanceRepository() *FinanceRepository {
	return &FinanceRepository{}
}

func (r *FinanceRepository) FindByIdForUpdate(db *gorm.DB, id any) (*entity.Finance, error) {
	finance := new(entity.Finance)
	err := db.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id = ?", id).First(finance).Error
	return finance, err
}

func (r *FinanceRepository) FindById(db *gorm.DB, id any) (*entity.Finance, error) {
	finance := new(entity.Finance)
	err := db.Where("id = ?", id).First(finance).Error
	return finance, err
}

func (r *FinanceRepository) FindAll(db *gorm.DB, query *model.FindAllFinanceRequest) ([]entity.Finance, int64, error) {
	var finances []entity.Finance
	scope := r.FilterFinance(query)

	total, err := r.Count(db, query)
	if err != nil {
		return nil, 0, err
	}

	if err := db.Scopes(scope).
		Offset((query.Page - 1) * query.Size).
		Limit(query.Size).
		Order("created_at DESC").
		Find(&finances).Error; err != nil {
		return nil, 0, err
	}

	return finances, total, nil
}

func (r *FinanceRepository) Count(db *gorm.DB, query *model.FindAllFinanceRequest) (int64, error) {
	var total int64
	if err := db.Model(new(entity.Finance)).
		Scopes(r.FilterFinance(query)).
		Count(&total).Error; err != nil {
		return 0, err
	}
	return total, nil
}

func (r *FinanceRepository) FindAllUnpaginated(db *gorm.DB) ([]entity.Finance, int64, error) {
	var finances []entity.Finance
	var total int64

	if err := db.Model(new(entity.Finance)).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := db.Find(&finances).Error; err != nil {
		return nil, 0, err
	}

	return finances, total, nil
}

func (r *FinanceRepository) FilterFinance(query *model.FindAllFinanceRequest) func(*gorm.DB) *gorm.DB {
	return func(tx *gorm.DB) *gorm.DB {
		tx = tx.Scopes(
			SearchScope(
				[]string{"category", "description", "extra_note", "type"},
				query.SearchQuery,
			),
			DateRangeScope("created_at", query.StartDate, query.EndDate),
		)
		return tx
	}
}
