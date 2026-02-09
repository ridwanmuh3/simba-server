package service

import (
	"context"

	"github.com/go-playground/validator/v10"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/ridwanmuh3/simba-server/internal/entity"
	"github.com/ridwanmuh3/simba-server/internal/exception"
	"github.com/ridwanmuh3/simba-server/internal/model"
)

type DashboardService struct {
	db       *gorm.DB
	log      *zap.SugaredLogger
	validate *validator.Validate
}

func NewDashboardService(db *gorm.DB, logger *zap.SugaredLogger, validate *validator.Validate) *DashboardService {
	return &DashboardService{
		db:       db,
		log:      logger,
		validate: validate,
	}
}

func (s *DashboardService) GetDashboardStats(
	ctx context.Context,
) (*model.DashboardStatsResponse, error) {
	tx := s.db.WithContext(ctx).Begin()
	defer tx.Rollback()

	var (
		totalItems  int64
		stockIn     int64
		stockOut    int64
		totalBudget int64
		budgetIn    int64
		budgetOut   int64
	)

	// ===============================
	// TOTAL ITEM
	// ===============================
	if err := tx.
		Model(&entity.Item{}).
		Count(&totalItems).Error; err != nil {
		s.log.Errorf("failed to count total items: %v", err)
		return nil, exception.InternalServerError
	}

	// ===============================
	// STOCK MASUK / KELUAR
	// ===============================
	tx.
		Model(&entity.StockTracking{}).
		Select("COALESCE(SUM(amount),0)").
		Where("type = ?", "IN").
		Scan(&stockIn)

	tx.
		Model(&entity.StockTracking{}).
		Select("COALESCE(SUM(amount),0)").
		Where("type = ?", "OUT").
		Scan(&stockOut)

	// ===============================
	// BUDGET TOTAL
	// ===============================
	tx.
		Model(&entity.Finance{}).
		Select("COALESCE(SUM(amount),0)").
		Scan(&totalBudget)

	tx.
		Model(&entity.Finance{}).
		Select("COALESCE(SUM(amount),0)").
		Where("type = ?", "IN").
		Scan(&budgetIn)

	tx.
		Model(&entity.Finance{}).
		Select("COALESCE(SUM(amount),0)").
		Where("type = ?", "OUT").
		Scan(&budgetOut)

	// ===============================
	// GRAFIK BULANAN
	// ===============================
	var monthly []model.MonthlyBudgetStat

	tx.
		Table("finances").
		Select(`
			TO_CHAR(created_at, 'Mon') AS month,
			SUM(CASE WHEN type='IN' THEN amount ELSE 0 END) AS in,
			SUM(CASE WHEN type='OUT' THEN amount ELSE 0 END) AS out
		`).
		Group("month").
		Order("MIN(created_at)").
		Scan(&monthly)

	// ===============================
	// KOMPOSISI PENGELUARAN
	// ===============================
	var composition []model.ExpenseComposition

	tx.
		Table("finances").
		Select("category, SUM(amount) as amount").
		Where("type = ?", "OUT").
		Group("category").
		Scan(&composition)

	var activities []model.SystemActivity
	tx.
		Table("activity_logs").
		Select("id, type, title, description, created_at").Limit(5).Order("created_at DESC").
		Scan(&activities)

	if err := tx.Commit().Error; err != nil {
		s.log.Errorf("failed to commit transaction database: %v", err)
		return nil, exception.InternalServerError
	}

	return &model.DashboardStatsResponse{
		TotalItems:       totalItems,
		StockIn:          stockIn,
		StockOut:         stockOut,
		TotalBudget:      totalBudget,
		BudgetIn:         budgetIn,
		BudgetOut:        budgetOut,
		MonthlyBudget:    monthly,
		ExpenseByType:    composition,
		SystemActivities: activities,
	}, nil
}
