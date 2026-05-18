package repository

import (
	"gorm.io/gorm"

	"github.com/ridwanmuh3/simba-server/internal/entity"
	"github.com/ridwanmuh3/simba-server/internal/model"
)

type DashboardRepository struct{}

func NewDashboardRepository() *DashboardRepository {
	return &DashboardRepository{}
}

func (r *DashboardRepository) GetItemCount(db *gorm.DB, dapurID uint) (int64, error) {
	var count int64
	err := db.Model(new(entity.Item)).Where("dapur_id = ?", dapurID).Count(&count).Error
	return count, err
}

func (r *DashboardRepository) SumStockByType(db *gorm.DB, stockType string, dapurID uint) (float64, error) {
	var total float64
	err := db.Model(new(entity.StockTracking)).
		Select("COALESCE(SUM(amount),0)").
		Where("type = ? AND dapur_id = ? AND deleted_at IS NULL AND created_at >= NOW() - INTERVAL '1 year'", stockType, dapurID).
		Scan(&total).Error
	return total, err
}

func (r *DashboardRepository) GetFinanceSummary(db *gorm.DB, dapurID uint) (int64, int64, error) {
	var budgetIn int64
	var budgetOut int64

	if err := db.Model(new(entity.Finance)).
		Select("COALESCE(SUM(amount),0)").
		Where("type = ? AND dapur_id = ? AND deleted_at IS NULL AND created_at >= NOW() - INTERVAL '1 year'", "PEMASUKAN", dapurID).
		Scan(&budgetIn).Error; err != nil {
		return 0, 0, err
	}

	if err := db.Model(new(entity.Finance)).
		Select("COALESCE(SUM(amount),0)").
		Where("type = ? AND dapur_id = ? AND deleted_at IS NULL AND created_at >= NOW() - INTERVAL '1 year'", "PENGELUARAN", dapurID).
		Scan(&budgetOut).Error; err != nil {
		return 0, 0, err
	}

	return budgetIn, budgetOut, nil
}

func (r *DashboardRepository) GetMonthlyStats(db *gorm.DB, dapurID uint) ([]model.MonthlyBudgetStat, error) {
	var stats []model.MonthlyBudgetStat
	err := db.
		Table("finances").
		Select(`
			TO_CHAR(DATE_TRUNC('month', created_at), 'Mon YYYY') AS month,
			SUM(CASE WHEN type='PEMASUKAN' THEN amount ELSE 0 END) AS "in",
			SUM(CASE WHEN type='PENGELUARAN' THEN amount ELSE 0 END) AS "out"
		`).
		Where("dapur_id = ? AND deleted_at IS NULL AND created_at >= NOW() - INTERVAL '1 year'", dapurID).
		Group("DATE_TRUNC('month', created_at)").
		Order("DATE_TRUNC('month', created_at) ASC").
		Scan(&stats).Error
	return stats, err
}

func (r *DashboardRepository) GetExpenseComposition(db *gorm.DB, dapurID uint) ([]model.ExpenseComposition, error) {
	var composition []model.ExpenseComposition
	err := db.
		Table("finances").
		Select("category, SUM(amount) as amount").
		Where("type = ? AND dapur_id = ? AND deleted_at IS NULL AND created_at >= NOW() - INTERVAL '1 year'", "PENGELUARAN", dapurID).
		Group("category").
		Scan(&composition).Error
	return composition, err
}

func (r *DashboardRepository) GetRecentActivities(db *gorm.DB, dapurID uint, limit int) ([]model.SystemActivity, error) {
	var activities []model.SystemActivity
	err := db.
		Table("activity_logs").
		Select("id, type, title, description, created_at").
		Where("dapur_id = ? AND deleted_at IS NULL", dapurID).
		Limit(limit).
		Order("created_at ASC").
		Scan(&activities).Error
	return activities, err
}
