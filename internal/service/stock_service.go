package service

import (
	"context"
	"fmt"

	"github.com/go-playground/validator/v10"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/ridwanmuh3/simba-server/internal/entity"
	"github.com/ridwanmuh3/simba-server/internal/exception"
	"github.com/ridwanmuh3/simba-server/internal/model"
	"github.com/ridwanmuh3/simba-server/internal/model/converter"
	"github.com/ridwanmuh3/simba-server/internal/repository"
	"github.com/ridwanmuh3/simba-server/internal/util"
)

type StockService struct {
	db              *gorm.DB
	log             *zap.SugaredLogger
	validate        *validator.Validate
	itemRepository  StockRepository
	activityEmitter ActivityEmitterIntf
	lowStockChecker *LowStockChecker
}

type ActivityEmitterIntf interface {
	Emit(log *entity.ActivityLog)
}

type StockRepository interface {
	Save(db *gorm.DB, entity *entity.Item) error
	FindAllStocks(db *gorm.DB, query *model.FindAllStocksRequest) ([]entity.StockTracking, int64, error)
	FindLastStockPrice(db *gorm.DB, itemID string, stockType string, dapurID uint) (float64, error)
	FindItemForUpdate(db *gorm.DB, itemID string, dapurID uint) (*entity.Item, error)
	FindTrackForUpdate(db *gorm.DB, trackID uint, itemID string, dapurID uint) (*entity.StockTracking, error)
	CreateTrack(db *gorm.DB, track *entity.StockTracking) error
	UpdateTrackFields(db *gorm.DB, track *entity.StockTracking) error
	DeleteTrackHard(db *gorm.DB, trackID uint) error
	ListTracksChronological(db *gorm.DB, itemID string, dapurID uint) ([]entity.StockTracking, error)
	UpdateTracksMAC(db *gorm.DB, tracks []entity.StockTracking) error
	StockFinanceSummary(db *gorm.DB, dapurID uint) (master, cost, revenue float64, err error)
	ItemStocksSummary(db *gorm.DB, request *model.GetItemStockSummaryRequest) ([]model.ItemStocksSummaryResponse, int64, error)
}

var _ StockRepository = (*repository.ItemRepository)(nil)

func NewStockService(db *gorm.DB, logger *zap.SugaredLogger, validate *validator.Validate, itemRepository StockRepository, activityEmitter ActivityEmitterIntf, lowStockChecker *LowStockChecker) *StockService {
	return &StockService{
		db:              db,
		log:             logger,
		validate:        validate,
		itemRepository:  itemRepository,
		activityEmitter: activityEmitter,
		lowStockChecker: lowStockChecker,
	}
}

func (s *StockService) UpdateStock(ctx context.Context, request *model.UpdateItemStockRequest) (*model.StockResponse, error) {
	// 1. VALIDATION (before Begin: no connection held on invalid input)
	if err := s.validate.Struct(request); err != nil {
		return nil, err
	}
	if request.Amount <= 0 {
		return nil, exception.InvalidAmountError
	}
	if request.Type != "IN" && request.Type != "OUT" {
		return nil, exception.InvalidStockTypeError
	}
	if request.Supplier == "" {
		request.Supplier = "-"
	}

	tx := s.db.WithContext(ctx).Begin()
	defer tx.Rollback()

	// 2. LOCK ITEM — scoped to dapur; this alone serializes writers.
	item, err := s.itemRepository.FindItemForUpdate(tx, request.ID, request.DapurID)
	if err != nil {
		return nil, exception.ItemNotFoundError
	}

	// 3. SINGLE SOURCE OF TRUTH: item.Stock / item.TotalPrice
	previousStock := item.Stock
	currentTotalPrice := item.TotalPrice

	// 4. INIT TRACKING
	tr := &entity.StockTracking{
		ItemID:        item.ID,
		Type:          request.Type,
		ModifiedBy:    request.ModifiedBy,
		PreviousStock: util.Round4(previousStock),
		Amount:        request.Amount,
		UnitPrice:     request.UnitPrice,
		Supplier:      request.Supplier,
		DapurID:       request.DapurID,
	}

	// 5. CORE
	switch request.Type {
	case "IN":
		newStock := util.Round4(previousStock + request.Amount)
		addedTotal := util.Round2(request.Amount * request.UnitPrice)
		item.Stock = newStock
		item.TotalPrice = util.Round2(currentTotalPrice + addedTotal)
		tr.NewStock = newStock
		tr.TotalPrice = addedTotal
	case "OUT":
		if request.Amount > previousStock {
			return nil, exception.InsufficientStockError
		}
		var avg float64
		if previousStock > 0 {
			avg = util.Round4(currentTotalPrice / previousStock)
		}
		newStock := util.Round4(previousStock - request.Amount)
		deduction := util.Round2(avg * request.Amount)
		lineTotal := util.Round2(request.Amount * request.UnitPrice)
		newTotalPrice := util.Round2(currentTotalPrice - deduction)
		if newTotalPrice < 0 {
			newTotalPrice = 0
		}
		item.Stock = newStock
		item.TotalPrice = newTotalPrice
		tr.NewStock = newStock
		tr.TotalPrice = lineTotal
	}

	// 6. SAVE ITEM + TRACKING via repository
	if err := s.itemRepository.Save(tx, item); err != nil {
		return nil, exception.InternalServerError
	}
	if err := s.itemRepository.CreateTrack(tx, tr); err != nil {
		return nil, exception.InternalServerError
	}

	// 7. COMMIT
	if err := tx.Commit().Error; err != nil {
		return nil, exception.InternalServerError
	}

	// 8. ASYNC
	s.activityEmitter.Emit(&entity.ActivityLog{
		Type:        "UPDATE-STOCK",
		Title:       "Stok bahan diperbarui",
		Description: fmt.Sprintf("%s: %.2f → %.2f %s", item.Name, tr.PreviousStock, tr.NewStock, item.MeasureUnit),
		ActionBy:    request.ModifiedBy,
		DapurID:     request.DapurID,
	})
	if s.lowStockChecker != nil {
		s.lowStockChecker.Check(item, request.DapurID)
	}

	return converter.StockToResponse(tr), nil
}

func (s *StockService) EditStock(ctx context.Context, request *model.EditStockRequest) (*model.StockResponse, error) {
	if err := s.validate.Struct(request); err != nil {
		return nil, err
	}
	if request.Supplier == "" {
		request.Supplier = "-"
	}

	tx := s.db.WithContext(ctx).Begin()
	defer tx.Rollback()

	item, err := s.itemRepository.FindItemForUpdate(tx, request.ID, request.DapurID)
	if err != nil {
		return nil, exception.ItemNotFoundError
	}
	stock, err := s.itemRepository.FindTrackForUpdate(tx, uint(request.StockID), request.ID, request.DapurID)
	if err != nil {
		return nil, exception.ItemNotFoundError
	}

	stock.Amount = request.Amount
	stock.UnitPrice = request.UnitPrice
	stock.Supplier = request.Supplier
	stock.ModifiedBy = request.ModifiedBy
	if err := s.itemRepository.UpdateTrackFields(tx, stock); err != nil {
		return nil, exception.InternalServerError
	}

	// REBUILD MAC: repository load/save + pure CalculateMAC
	tracks, runningStock, totalPrice, err := recalculateMAC(s.itemRepository, tx, item, request.DapurID)
	if err != nil {
		s.log.Errorf("failed to recalculate MAC: %v", err)
		if err == exception.InsufficientStockError {
			return nil, err
		}
		return nil, exception.InternalServerError
	}

	item.Stock = util.Round4(runningStock)
	item.TotalPrice = util.Round2(totalPrice)
	if err := s.itemRepository.Save(tx, item); err != nil {
		return nil, exception.InternalServerError
	}
	if err := tx.Commit().Error; err != nil {
		return nil, exception.InternalServerError
	}

	s.activityEmitter.Emit(&entity.ActivityLog{
		Type:        "EDIT-STOCK-" + stock.Type,
		Title:       "Stock entry edited and chain recalculated",
		Description: fmt.Sprintf("%s [%s]: %.2f @ %.2f %s", item.Name, stock.Type, request.Amount, request.UnitPrice, item.MeasureUnit),
		ActionBy:    request.ModifiedBy,
		DapurID:     request.DapurID,
	})
	if s.lowStockChecker != nil {
		s.lowStockChecker.Check(item, request.DapurID)
	}

	var updatedTarget *entity.StockTracking
	for i := range tracks {
		if tracks[i].ID == uint(request.StockID) {
			t := tracks[i]
			updatedTarget = &t
		}
	}
	if updatedTarget == nil {
		return nil, exception.InternalServerError
	}
	return converter.StockToResponse(updatedTarget), nil
}

func (s *StockService) DeleteStock(ctx context.Context, request *model.DeleteStockRequest) (bool, error) {
	if err := s.validate.Struct(request); err != nil {
		return false, err
	}

	tx := s.db.WithContext(ctx).Begin()
	defer tx.Rollback()

	item, err := s.itemRepository.FindItemForUpdate(tx, request.ID, request.DapurID)
	if err != nil {
		return false, exception.ItemNotFoundError
	}
	stock, err := s.itemRepository.FindTrackForUpdate(tx, uint(request.StockID), request.ID, request.DapurID)
	if err != nil {
		return false, exception.ItemNotFoundError
	}

	if err := s.itemRepository.DeleteTrackHard(tx, stock.ID); err != nil {
		return false, exception.InternalServerError
	}

	_, runningStock, totalPrice, err := recalculateMAC(s.itemRepository, tx, item, request.DapurID)
	if err != nil {
		s.log.Errorf("failed to recalculate MAC after delete: %v", err)
		if err == exception.InsufficientStockError {
			return false, err
		}
		return false, exception.InternalServerError
	}

	item.Stock = util.Round4(runningStock)
	item.TotalPrice = util.Round2(totalPrice)
	if err := s.itemRepository.Save(tx, item); err != nil {
		return false, exception.InternalServerError
	}
	if err := tx.Commit().Error; err != nil {
		return false, exception.InternalServerError
	}

	action := "DELETE-STOCK"
	if stock.Type == "IN" {
		action = "DELETE-STOCK-IN"
	} else {
		action = "DELETE-STOCK-OUT"
	}
	s.activityEmitter.Emit(&entity.ActivityLog{
		Type:        action,
		Title:       "Stock recalculated after delete",
		Description: fmt.Sprintf("%s: %.2f %s", item.Name, item.Stock, item.MeasureUnit),
		ActionBy:    request.ModifiedBy,
		DapurID:     request.DapurID,
	})

	return true, nil
}

func (s *StockService) FindAllStocks(ctx context.Context, request *model.FindAllStocksRequest) ([]model.StockResponse, int64, error) {
	db := s.db.WithContext(ctx)
	stocksTracking, total, err := s.itemRepository.FindAllStocks(db, request)
	if err != nil {
		s.log.Errorf("failed to find all items stocks: %v", err)
		return nil, 0, exception.InternalServerError
	}
	responses := make([]model.StockResponse, len(stocksTracking))
	for i, stock := range stocksTracking {
		responses[i] = *converter.StockToResponse(&stock)
	}
	return responses, total, nil
}

func (s *StockService) GetStocksFinanceSummary(ctx context.Context, dapurID uint) (*model.StocksFinanceSummaryResponse, error) {
	db := s.db.WithContext(ctx)
	masterItemsTotalBudget, cost, revenue, err := s.itemRepository.StockFinanceSummary(db, dapurID)
	if err != nil {
		s.log.Errorf("failed to get stocks finance summary: %v", err)
		return nil, exception.InternalServerError
	}
	profit := revenue - cost
	return &model.StocksFinanceSummaryResponse{
		MasterItemsTotalBudget: masterItemsTotalBudget,
		BudgetIn:               cost,
		BudgetOut:              revenue,
		Profit:                 profit,
		CurrentBudget:          masterItemsTotalBudget + profit,
	}, nil
}

func (s *StockService) GetItemStocksSummary(ctx context.Context, request *model.GetItemStockSummaryRequest) ([]model.ItemStocksSummaryResponse, int64, error) {
	db := s.db.WithContext(ctx)
	if err := s.validate.Struct(request); err != nil {
		s.log.Errorf("failed to validate request body: %v", err)
		return nil, 0, err
	}
	responses, total, err := s.itemRepository.ItemStocksSummary(db, request)
	if err != nil {
		s.log.Errorf("failed to fetch item stocks summary: %v", err)
		return nil, 0, err
	}
	return responses, total, nil
}

func (s *StockService) GetLastStockPrice(ctx context.Context, itemID string, stockType string, dapurID uint) (float64, error) {
	return s.itemRepository.FindLastStockPrice(s.db.WithContext(ctx), itemID, stockType, dapurID)
}
