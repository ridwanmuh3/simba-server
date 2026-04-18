package repository

import (
	"github.com/ridwanmuh3/simba-server/internal/entity"
	"github.com/ridwanmuh3/simba-server/internal/model"
	"gorm.io/gorm"
)

type DashboardRepository struct{}

func NewDashboardRepository() *DashboardRepository {
	return &DashboardRepository{}
}

func (r *DashboardRepository) GetItemCount(db *gorm.DB) (int64, error) {
	var count int64
	err := db.Model(new(entity.Item)).Count(&count).Error
	return count, err
}

func (r *DashboardRepository) CountItems(db *gorm.DB) (int64, error) {
	return r.GetItemCount(db)
}

func (r *DashboardRepository) SumStockByType(db *gorm.DB, stockType string) (int64, error) {
	var total int64
	err := db.Model(new(entity.StockTracking)).
		Select("COALESCE(SUM(amount),0)").
		Where("type = ?", stockType).
		Scan(&total).Error
	return total, err
}

func (r *DashboardRepository) GetFinanceSummary(db *gorm.DB) (int64, int64, error) {
	var budgetIn int64
	var budgetOut int64

	if err := db.Model(new(entity.Finance)).
		Select("COALESCE(SUM(amount),0)").
		Where("type = ?", "PEMASUKAN").
		Scan(&budgetIn).Error; err != nil {
		return 0, 0, err
	}

	if err := db.Model(new(entity.Finance)).
		Select("COALESCE(SUM(amount),0)").
		Where("type = ?", "PENGELUARAN").
		Scan(&budgetOut).Error; err != nil {
		return 0, 0, err
	}

	return budgetIn, budgetOut, nil
}

func (r *DashboardRepository) GetMonthlyStats(db *gorm.DB) ([]model.MonthlyBudgetStat, error) {
	var stats []model.MonthlyBudgetStat
	err := db.
		Table("finances").
		Select(`
			TO_CHAR(DATE_TRUNC('month', created_at), 'Mon YYYY') AS month,
			SUM(CASE WHEN type='PEMASUKAN' THEN amount ELSE 0 END) AS "in",
			SUM(CASE WHEN type='PENGELUARAN' THEN amount ELSE 0 END) AS "out"
		`).
		Group("DATE_TRUNC('month', created_at)").
		Order("DATE_TRUNC('month', created_at) ASC").
		Scan(&stats).Error
	return stats, err
}

func (r *DashboardRepository) GetMonthlyBudget(db *gorm.DB) ([]model.MonthlyBudgetStat, error) {
	return r.GetMonthlyStats(db)
}

func (r *DashboardRepository) GetExpenseComposition(db *gorm.DB) ([]model.ExpenseComposition, error) {
	var composition []model.ExpenseComposition
	err := db.
		Table("finances").
		Select("category, SUM(amount) as amount").
		Where("type = ?", "PENGELUARAN").
		Group("category").
		Scan(&composition).Error
	return composition, err
}

func (r *DashboardRepository) GetRecentActivities(db *gorm.DB, limit int) ([]model.SystemActivity, error) {
	var activities []model.SystemActivity
	err := db.
		Table("activity_logs").
		Select("id, type, title, description, created_at").
		Limit(limit).
		Order("created_at DESC").
		Scan(&activities).Error
	return activities, err
}
