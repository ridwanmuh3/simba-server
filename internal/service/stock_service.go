package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

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
	tx := s.db.WithContext(ctx).Begin()
	defer tx.Rollback()

	// 1. VALIDATION
	if err := s.validate.Struct(request); err != nil {
		return nil, err
	}
	if request.Amount <= 0 {
		return nil, fiber.NewError(fiber.StatusBadRequest, "amount must be greater than 0")
	}
	if request.Supplier == "" {
		request.Supplier = "-"
	}

	// 2. LOCK ITEM — scoped to dapur
	item := new(entity.Item)
	if err := tx.
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id = ? AND dapur_id = ?", request.ID, request.DapurID).
		First(item).Error; err != nil {
		return nil, exception.ItemNotFoundError
	}

	// 3. GET LAST TRACKING (DETERMINISTIC) — scoped to dapur
	var last entity.StockTracking
	err := tx.
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("item_id = ? AND dapur_id = ? AND deleted_at IS NULL", item.ID, request.DapurID).
		Order("created_at DESC, id DESC").
		First(&last).Error

	var previousStock float64
	var currentTotalPrice float64

	if err == nil {
		previousStock = last.NewStock
		currentTotalPrice = item.TotalPrice
	} else if errors.Is(err, gorm.ErrRecordNotFound) {
		previousStock = item.Stock
		currentTotalPrice = item.TotalPrice
	} else {
		return nil, exception.InternalServerError
	}

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
		amount := request.Amount

		if previousStock == 0 {
			amount = 0
		}

		if amount > previousStock {
			amount = previousStock
		}

		var avg float64
		if previousStock > 0 {
			avg = util.Round4(currentTotalPrice / previousStock)
		} else {
			avg = 0
		}

		newStock := util.Round4(previousStock - amount)
		if newStock < 0 {
			newStock = 0
		}

		deduction := util.Round2(avg * amount)
		lineTotal := util.Round2(amount * request.UnitPrice)

		newTotalPrice := util.Round2(currentTotalPrice - deduction)
		if newTotalPrice < 0 {
			newTotalPrice = 0
		}

		item.Stock = newStock
		item.TotalPrice = newTotalPrice

		tr.NewStock = newStock
		tr.TotalPrice = lineTotal

	default:
		return nil, fiber.NewError(fiber.StatusBadRequest, "invalid stock type")
	}

	if tr.NewStock < 0 {
		tr.NewStock = 0
	}
	if item.Stock < 0 {
		item.Stock = 0
	}

	// 6. SAVE ITEM
	if err := s.itemRepository.Save(tx, item); err != nil {
		return nil, exception.InternalServerError
	}

	// 7. SAVE TRACKING
	if err := tx.Create(tr).Error; err != nil {
		return nil, exception.InternalServerError
	}

	// 8. COMMIT
	if err := tx.Commit().Error; err != nil {
		return nil, exception.InternalServerError
	}

	// 9. ASYNC: emit activity + check low stock
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
	tx := s.db.WithContext(ctx).Begin()
	defer tx.Rollback()

	// 1. VALIDATION
	if err := s.validate.Struct(request); err != nil {
		return nil, err
	}
	if request.Supplier == "" {
		request.Supplier = "-"
	}

	// 2. LOCK ITEM — scoped to dapur
	item := new(entity.Item)
	if err := tx.
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id = ? AND dapur_id = ?", request.ID, request.DapurID).
		First(item).Error; err != nil {
		return nil, exception.ItemNotFoundError
	}

	// 3. LOCK & MUTATE TARGET TRACKING — scoped to dapur
	stock := new(entity.StockTracking)
	if err := tx.
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id = ? AND item_id = ? AND dapur_id = ?", request.StockID, request.ID, request.DapurID).
		First(stock).Error; err != nil {
		return nil, exception.ItemNotFoundError
	}

	stock.Amount = request.Amount
	stock.UnitPrice = request.UnitPrice
	stock.Supplier = request.Supplier
	stock.ModifiedBy = request.ModifiedBy

	if err := tx.Model(stock).Updates(map[string]any{
		"amount":      stock.Amount,
		"unit_price":  stock.UnitPrice,
		"supplier":    stock.Supplier,
		"modified_by": stock.ModifiedBy,
	}).Error; err != nil {
		return nil, exception.InternalServerError
	}

	// 4. REBUILD MAC
	tracks, runningStock, totalPrice, err := util.RecalculateMAC(tx, item, request.DapurID)
	if err != nil {
		s.log.Errorf("failed to recalculate MAC: %v", err)
		return nil, exception.InternalServerError
	}

	// 5. UPDATE ITEM
	item.Stock = util.Round4(runningStock)
	item.TotalPrice = util.Round2(totalPrice)

	if err := tx.Save(item).Error; err != nil {
		return nil, exception.InternalServerError
	}

	// 6. COMMIT
	if err := tx.Commit().Error; err != nil {
		return nil, exception.InternalServerError
	}

	// 7. ASYNC: emit activity + check low stock
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
	tx := s.db.WithContext(ctx).Begin()
	defer tx.Rollback()

	// 1. VALIDATION
	if err := s.validate.Struct(request); err != nil {
		return false, err
	}

	// 2. LOCK ITEM — scoped to dapur
	item := new(entity.Item)
	if err := tx.
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id = ? AND dapur_id = ?", request.ID, request.DapurID).
		First(item).Error; err != nil {
		return false, exception.ItemNotFoundError
	}

	// 3. GET TARGET TRACKING — scoped to dapur
	stock := new(entity.StockTracking)
	if err := tx.
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id = ? AND item_id = ? AND dapur_id = ?", request.StockID, request.ID, request.DapurID).
		First(stock).Error; err != nil {
		return false, exception.ItemNotFoundError
	}

	// 4. HARD DELETE
	if err := tx.
		Unscoped().
		Delete(&entity.StockTracking{}, "id = ?", stock.ID).Error; err != nil {
		return false, exception.InternalServerError
	}

	// 5. REBUILD MAC
	_, runningStock, totalPrice, err := util.RecalculateMAC(tx, item, request.DapurID)
	if err != nil {
		s.log.Errorf("failed to recalculate MAC after delete: %v", err)
		return false, exception.InternalServerError
	}

	// 6. UPDATE ITEM
	item.Stock = util.Round4(runningStock)
	item.TotalPrice = util.Round2(totalPrice)

	if err := tx.Save(item).Error; err != nil {
		return false, exception.InternalServerError
	}

	// 6. COMMIT
	if err := tx.Commit().Error; err != nil {
		return false, exception.InternalServerError
	}

	// 7. ASYNC: emit activity
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

	var (
		masterItemsTotalBudget float64
		cost                   float64
		revenue                float64
	)

	db.Model(new(entity.Item)).
		Select("COALESCE(SUM(total_price),0)").
		Where("dapur_id = ? AND deleted_at IS NULL", dapurID).
		Scan(&masterItemsTotalBudget)

	db.Model(new(entity.StockTracking)).
		Select("COALESCE(SUM(amount * unit_price),0)").
		Where("type = ? AND dapur_id = ? AND deleted_at IS NULL", "IN", dapurID).
		Scan(&cost)

	db.Model(new(entity.StockTracking)).
		Select("COALESCE(SUM(amount * unit_price),0)").
		Where("type = ? AND dapur_id = ? AND deleted_at IS NULL", "OUT", dapurID).
		Scan(&revenue)

	profit := revenue - cost
	currentBudget := masterItemsTotalBudget + profit

	return &model.StocksFinanceSummaryResponse{
		MasterItemsTotalBudget: masterItemsTotalBudget,
		BudgetIn:               cost,
		BudgetOut:              revenue,
		Profit:                 profit,
		CurrentBudget:          currentBudget,
	}, nil
}

func (s *StockService) GetItemStocksSummary(ctx context.Context, request *model.GetItemStockSummaryRequest) ([]model.ItemStocksSummaryResponse, int64, error) {
	db := s.db.WithContext(ctx)

	if err := s.validate.Struct(request); err != nil {
		s.log.Errorf("failed to validate request body: %v", err)
		return nil, 0, err
	}

	var responses []model.ItemStocksSummaryResponse
	var total int64
	offset := (request.Page - 1) * request.Size

	baseQuery := db.Table("items").Where("items.deleted_at IS NULL AND items.dapur_id = ?", request.DapurID)

	if request.SearchQuery != "" {
		baseQuery = baseQuery.Where("items.name LIKE ?", "%"+request.SearchQuery+"%")
	}

	err := baseQuery.Count(&total).Error
	if err != nil {
		s.log.Errorf("failed to count items summary: %v", err)
		return nil, 0, err
	}

	if total == 0 {
		return []model.ItemStocksSummaryResponse{}, 0, nil
	}

	loc, _ := time.LoadLocation("Asia/Jakarta")
	joinClause := "LEFT JOIN stock_tracks st ON st.item_id = items.id AND st.deleted_at IS NULL AND st.dapur_id = ?"
	joinArgs := []interface{}{request.DapurID}

	if request.StartDate != "" {
		if parsed, err := time.Parse(time.RFC3339Nano, request.StartDate); err == nil {
			p := parsed.In(loc)
			start := time.Date(p.Year(), p.Month(), p.Day(), 0, 0, 0, 0, loc)
			joinClause += " AND st.created_at >= ?"
			joinArgs = append(joinArgs, start.UTC())
		}
	}
	if request.EndDate != "" {
		if parsed, err := time.Parse(time.RFC3339Nano, request.EndDate); err == nil {
			p := parsed.In(loc)
			end := time.Date(p.Year(), p.Month(), p.Day(), 23, 59, 59, 999999999, loc)
			joinClause += " AND st.created_at <= ?"
			joinArgs = append(joinArgs, end.UTC())
		}
	}

	err = baseQuery.
		Select(`
            items.id AS item_id,
            items.name AS name,
            items.category AS category,
            items.measure_unit AS measure_unit,
            items.stock AS current_stock,
            items.total_price AS stock_value,
            items.initial_stock AS initial_stock,
            COALESCE(
                SUM(CASE WHEN st.type = 'IN' THEN st.total_price ELSE 0 END) /
                NULLIF(SUM(CASE WHEN st.type = 'IN' THEN st.amount ELSE 0 END), 0),
            items.unit_price) AS buy_price,
            COALESCE(
                SUM(CASE WHEN st.type = 'OUT' THEN st.total_price ELSE 0 END) /
                NULLIF(SUM(CASE WHEN st.type = 'OUT' THEN st.amount ELSE 0 END), 0),
            0) AS sell_price,
            COALESCE(SUM(CASE WHEN st.type = 'IN' THEN st.amount ELSE 0 END), 0) AS total_in,
            COALESCE(SUM(CASE WHEN st.type = 'OUT' THEN st.amount ELSE 0 END), 0) AS total_out
        `).
		Joins(joinClause, joinArgs...).
		Group(`
            items.id,
            items.name,
            items.category,
            items.measure_unit,
            items.stock,
            items.total_price,
            items.initial_stock,
            items.unit_price
        `).
		Order("items.created_at DESC, items.id DESC").
		Limit(request.Size).
		Offset(offset).
		Find(&responses).Error

	if err != nil {
		s.log.Errorf("failed to fetch item stocks summary: %v", err)
		return nil, 0, err
	}

	return responses, total, nil
}

func (s *StockService) GetLastStockPrice(ctx context.Context, itemID string, stockType string, dapurID uint) (float64, error) {
	return s.itemRepository.FindLastStockPrice(s.db.WithContext(ctx), itemID, stockType, dapurID)
}
