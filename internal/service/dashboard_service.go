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
	db := s.db.WithContext(ctx)

	var (
		totalItems int64
		stockIn    int64
		stockOut   int64
		// totalBudget int64
		budgetIn  int64
		budgetOut int64
	)

	// ===============================
	// TOTAL ITEM
	// ===============================
	if err := db.
		Model(new(entity.Item)).
		Count(&totalItems).Error; err != nil {
		s.log.Errorf("failed to count total items: %v", err)
		return nil, exception.InternalServerError
	}

	// ===============================
	// STOCK MASUK / KELUAR
	// ===============================
	db.
		Model(new(entity.StockTracking)).
		Select("COALESCE(SUM(amount),0)").
		Where("type = ?", "IN").
		Scan(&stockIn)

	db.
		Model(new(entity.StockTracking)).
		Select("COALESCE(SUM(amount),0)").
		Where("type = ?", "OUT").
		Scan(&stockOut)

	// ===============================
	// BUDGET TOTAL
	// ===============================
	// db.
	// 	Model(new(entity.Finance)).
	// 	Select("COALESCE(SUM(amount),0)").
	// 	Scan(&totalBudget)

	db.
		Model(new(entity.Finance)).
		Select("COALESCE(SUM(amount),0)").
		Where("type = ?", "PEMASUKAN").
		Scan(&budgetIn)

	db.
		Model(new(entity.Finance)).
		Select("COALESCE(SUM(amount),0)").
		Where("type = ?", "PENGELUARAN").
		Scan(&budgetOut)

	// ===============================
	// GRAFIK BULANAN
	// ===============================
	var monthly []model.MonthlyBudgetStat

	db.
		Table("finances").
		Select(`
			TO_CHAR(created_at, 'Mon') AS month,
			SUM(CASE WHEN type='PEMASUKAN' THEN amount ELSE 0 END) AS in,
			SUM(CASE WHEN type='PENGELUARAN' THEN amount ELSE 0 END) AS out
		`).
		Group("month").
		Order("MIN(created_at)").
		Scan(&monthly)

	// ===============================
	// KOMPOSISI PENGELUARAN
	// ===============================
	var composition []model.ExpenseComposition

	db.
		Table("finances").
		Select("category, SUM(amount) as amount").
		Where("type = ?", "PENGELUARAN").
		Group("category").
		Scan(&composition)

	var activities []model.SystemActivity
	db.
		Table("activity_logs").
		Select("id, type, title, description, created_at").Limit(4).Order("created_at DESC").
		Scan(&activities)

	return &model.DashboardStatsResponse{
		TotalItems:       totalItems,
		StockIn:          stockIn,
		StockOut:         stockOut,
		TotalBudget:      budgetIn - budgetOut,
		BudgetIn:         budgetIn,
		BudgetOut:        budgetOut,
		MonthlyBudget:    monthly,
		ExpenseByType:    composition,
		SystemActivities: activities,
	}, nil
}
