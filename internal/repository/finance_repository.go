package repository

import (
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/ridwanmuh3/simba-server/internal/entity"
	"github.com/ridwanmuh3/simba-server/internal/model"
)

type FinanceRepository struct {
	Repository[entity.Finance]
}

func NewFinanceRepository() *FinanceRepository {
	return &FinanceRepository{}
}

func (r *FinanceRepository) FindByIdForUpdate(db *gorm.DB, id any, dapurID uint) (*entity.Finance, error) {
	finance := new(entity.Finance)
	err := db.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id = ? AND dapur_id = ?", id, dapurID).First(finance).Error
	return finance, err
}

func (r *FinanceRepository) FindById(db *gorm.DB, id any, dapurID uint) (*entity.Finance, error) {
	finance := new(entity.Finance)
	err := db.Where("id = ? AND dapur_id = ?", id, dapurID).First(finance).Error
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
		Order(OrderBy()).
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

func (r *FinanceRepository) FindAllUnpaginated(db *gorm.DB, dapurID uint) ([]entity.Finance, int64, error) {
	var finances []entity.Finance
	var total int64

	q := db.Model(new(entity.Finance)).Where("dapur_id = ?", dapurID)
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := db.Where("dapur_id = ?", dapurID).Find(&finances).Error; err != nil {
		return nil, 0, err
	}

	return finances, total, nil
}

func (r *FinanceRepository) FilterFinance(query *model.FindAllFinanceRequest) func(*gorm.DB) *gorm.DB {
	return func(tx *gorm.DB) *gorm.DB {
		if query.DapurID > 0 {
			tx = tx.Where("dapur_id = ?", query.DapurID)
		}
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
