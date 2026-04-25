package service

import (
	"context"

	"github.com/go-playground/validator/v10"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/ridwanmuh3/simba-server/internal/exception"
	"github.com/ridwanmuh3/simba-server/internal/model"
	"github.com/ridwanmuh3/simba-server/internal/repository"
)

type DashboardService struct {
	db                  *gorm.DB
	log                 *zap.SugaredLogger
	validate            *validator.Validate
	dashboardRepository DashboardRepository
}

type DashboardRepository interface {
	GetItemCount(db *gorm.DB, dapurID uint) (int64, error)
	SumStockByType(db *gorm.DB, stockType string, dapurID uint) (float64, error)
	GetFinanceSummary(db *gorm.DB, dapurID uint) (int64, int64, error)
	GetMonthlyStats(db *gorm.DB, dapurID uint) ([]model.MonthlyBudgetStat, error)
	GetExpenseComposition(db *gorm.DB, dapurID uint) ([]model.ExpenseComposition, error)
	GetRecentActivities(db *gorm.DB, dapurID uint, limit int) ([]model.SystemActivity, error)
}

var _ DashboardRepository = (*repository.DashboardRepository)(nil)

func NewDashboardService(db *gorm.DB, logger *zap.SugaredLogger, validate *validator.Validate, dashboardRepository DashboardRepository) *DashboardService {
	return &DashboardService{
		db:                  db,
		log:                 logger,
		validate:            validate,
		dashboardRepository: dashboardRepository,
	}
}

func (s *DashboardService) GetDashboardStats(ctx context.Context, dapurID uint) (*model.DashboardStatsResponse, error) {
	db := s.db.WithContext(ctx)

	totalItems, err := s.dashboardRepository.GetItemCount(db, dapurID)
	if err != nil {
		s.log.Errorf("failed to count total items: %v", err)
		return nil, exception.InternalServerError
	}

	stockIn, err := s.dashboardRepository.SumStockByType(db, "IN", dapurID)
	if err != nil {
		s.log.Errorf("failed to sum IN stocks: %v", err)
		return nil, exception.InternalServerError
	}

	stockOut, err := s.dashboardRepository.SumStockByType(db, "OUT", dapurID)
	if err != nil {
		s.log.Errorf("failed to sum OUT stocks: %v", err)
		return nil, exception.InternalServerError
	}

	budgetIn, budgetOut, err := s.dashboardRepository.GetFinanceSummary(db, dapurID)
	if err != nil {
		s.log.Errorf("failed to get finance summary: %v", err)
		return nil, exception.InternalServerError
	}

	monthly, err := s.dashboardRepository.GetMonthlyStats(db, dapurID)
	if err != nil {
		s.log.Errorf("failed to get monthly stats: %v", err)
		return nil, exception.InternalServerError
	}

	composition, err := s.dashboardRepository.GetExpenseComposition(db, dapurID)
	if err != nil {
		s.log.Errorf("failed to get expense composition: %v", err)
		return nil, exception.InternalServerError
	}

	activities, err := s.dashboardRepository.GetRecentActivities(db, dapurID, 4)
	if err != nil {
		s.log.Errorf("failed to get recent activities: %v", err)
		return nil, exception.InternalServerError
	}

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
