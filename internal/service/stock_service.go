package service

import (
	"context"
	"fmt"

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
	db             *gorm.DB
	log            *zap.SugaredLogger
	validate       *validator.Validate
	itemRepository StockRepository
}

type StockRepository interface {
	Update(db *gorm.DB, entity *entity.Item, id any) error
	FindAllStocks(db *gorm.DB, query *model.FindAllStocksRequest) ([]entity.StockTracking, int64, error)
}

var _ StockRepository = (*repository.ItemRepository)(nil)

func NewStockService(db *gorm.DB, logger *zap.SugaredLogger, validate *validator.Validate, itemRepository StockRepository) *StockService {
	return &StockService{
		db:             db,
		log:            logger,
		validate:       validate,
		itemRepository: itemRepository,
	}
}

func (s *StockService) UpdateStock(ctx context.Context, request *model.UpdateItemStockRequest) (*model.StockResponse, error) {
	tx := s.db.WithContext(ctx).Begin()
	defer tx.Rollback()

	if err := s.validate.Struct(request); err != nil {
		s.log.Errorf("failed to validate request body: %v", err)
		return nil, err
	}

	item := new(entity.Item)
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		First(item, "id = ?", request.ID).Error; err != nil {
		s.log.Errorf("failed to find item by code: %v", err)
		return nil, exception.ItemNotFoundError
	}

	if request.Supplier == "" {
		request.Supplier = "-"
	}

	stockTracking := new(entity.StockTracking)
	stockTracking.ItemID = item.ID
	stockTracking.Type = request.Type
	stockTracking.ModifiedBy = request.ModifiedBy
	stockTracking.PreviousStock = item.Stock
	stockTracking.Amount = request.Amount
	stockTracking.UnitPrice = request.UnitPrice
	stockTracking.Supplier = request.Supplier

	switch request.Type {

	case "IN":
		newStock := util.Round4(item.Stock + request.Amount)

		addedTotal := util.Round2(request.Amount * request.UnitPrice)

		item.Stock = newStock
		item.TotalPrice = util.Round2(item.TotalPrice + addedTotal)

		stockTracking.NewStock = newStock
		stockTracking.TotalPrice = addedTotal

	case "OUT":
		if item.Stock < request.Amount {
			s.log.Errorf("failed to decrease item stock: insufficient stock")
			return nil, fiber.NewError(fiber.StatusBadRequest, "stock must be sufficient enough to decreased")
		}

		if item.Stock <= 0 {
			return nil, fiber.NewError(fiber.StatusBadRequest, "invalid stock state")
		}

		// For OUT invoice lines, use the explicit request.UnitPrice × Amount
		// (billed price) rather than weighted average cost, so invoice totals
		// match what was charged to the receiver.
		newStock := util.Round4(item.Stock - request.Amount)
		lineTotal := util.Round2(request.Amount * request.UnitPrice)

		// Internal stock value tracked with weighted-average for accurate budgeting.
		avg := item.TotalPrice / item.Stock
		deduction := util.Round2(avg * request.Amount)

		item.Stock = newStock
		item.TotalPrice = util.Round2(item.TotalPrice - deduction)

		stockTracking.NewStock = newStock
		stockTracking.TotalPrice = lineTotal

	default:
		s.log.Errorf("invalid stock type '%s'", request.Type)
		return nil, fiber.NewError(fiber.StatusBadRequest, "invalid update stock type")
	}

	if err := s.itemRepository.Update(tx, item, item.ID); err != nil {
		s.log.Errorf("failed to update item stock: %v", err)
		return nil, exception.InternalServerError
	}

	if err := tx.Create(stockTracking).Error; err != nil {
		s.log.Errorf("failed to create stock tracking: %v", err)
		return nil, exception.InternalServerError
	}

	if err := tx.Create(&entity.ActivityLog{
		Type:        "UPDATE-STOCK",
		Title:       "Stok bahan diperbarui",
		Description: fmt.Sprintf("%s - %g %s", item.Name, item.Stock, item.MeasureUnit),
		ActionBy:    request.ModifiedBy,
	}).Error; err != nil {
		s.log.Errorf("failed to save activity log: %v", err)
		return nil, exception.InternalServerError
	}

	if err := tx.Commit().Error; err != nil {
		s.log.Errorf("failed to commit transaction: %v", err)
		return nil, exception.InternalServerError
	}

	return converter.StockToResponse(stockTracking), nil
}

func (s *StockService) DeleteStock(ctx context.Context, request *model.DeleteStockRequest) (bool, error) {
	tx := s.db.WithContext(ctx).Begin()
	defer tx.Rollback()

	if err := s.validate.Struct(request); err != nil {
		return false, err
	}

	item := new(entity.Item)
	if err := tx.
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id = ?", request.ID).
		First(item).Error; err != nil {
		return false, exception.ItemNotFoundError
	}

	stock := new(entity.StockTracking)
	if err := tx.
		Where("id = ? AND item_id = ?", request.StockID, request.ID).
		First(stock).Error; err != nil {
		return false, exception.ItemNotFoundError
	}

	// We delete the tracking record first.
	if err := tx.
		Where("id = ? AND item_id = ?", request.StockID, item.ID).
		Delete(&entity.StockTracking{}).Error; err != nil {
		return false, exception.InternalServerError
	}

	// Recalculate stock and total price from history
	var stockTracks []entity.StockTracking
	if err := tx.Where("item_id = ?", item.ID).
		Order("created_at ASC").
		Find(&stockTracks).Error; err != nil {
		return false, exception.InternalServerError
	}

	runningStock := item.InitialStock
	totalPrice := util.Round2(item.InitialStock * item.UnitPrice)

	for i := range stockTracks {
		prevStock := runningStock
		stockTracks[i].PreviousStock = util.Round4(prevStock)

		switch stockTracks[i].Type {

		case "IN":
			runningStock = util.Round4(runningStock + stockTracks[i].Amount)
			totalPrice = util.Round2(totalPrice + stockTracks[i].Amount*stockTracks[i].UnitPrice)

		case "OUT":
			if prevStock > 0 {
				avg := totalPrice / prevStock
				deduction := util.Round2(avg * stockTracks[i].Amount)
				runningStock = util.Round4(runningStock - stockTracks[i].Amount)
				totalPrice = util.Round2(totalPrice - deduction)
			} else {
				runningStock = util.Round4(runningStock - stockTracks[i].Amount)
			}
		}

		stockTracks[i].NewStock = util.Round4(runningStock)
	}

	for i := range stockTracks {
		if err := tx.Model(&entity.StockTracking{}).
			Where("id = ?", stockTracks[i].ID).
			Updates(map[string]any{
				"previous_stock": stockTracks[i].PreviousStock,
				"new_stock":      stockTracks[i].NewStock,
			}).Error; err != nil {
			return false, exception.InternalServerError
		}
	}

	item.Stock = util.Round4(runningStock)
	item.TotalPrice = util.Round2(totalPrice)

	if err := tx.Save(item).Error; err != nil {
		return false, exception.InternalServerError
	}

	action := "DELETE-STOCK"
	if stock.Type == "IN" {
		action = "DELETE-STOCK-IN"
	} else {
		action = "DELETE-STOCK-OUT"
	}

	if err := tx.Create(&entity.ActivityLog{
		Type:        action,
		Title:       "Stock updated",
		Description: fmt.Sprintf("%s - %g %s", item.Name, item.Stock, item.MeasureUnit),
		ActionBy:    item.ModifiedBy,
	}).Error; err != nil {
		return false, exception.InternalServerError
	}

	if err := tx.Commit().Error; err != nil {
		return false, exception.InternalServerError
	}

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

func (s *StockService) GetStocksFinanceSummary(ctx context.Context) (*model.StocksFinanceSummaryResponse, error) {
	db := s.db.WithContext(ctx)

	var (
		masterItemsTotalBudget float64
		cost                   float64
		revenue                float64
		profit                 float64
		currentBudget          float64
	)

	db.
		Model(new(entity.Item)).
		Select("COALESCE(SUM(total_price),0)").
		Where("deleted_at IS NULL").
		Scan(&masterItemsTotalBudget)

	db.
		Model(new(entity.StockTracking)).
		Select("COALESCE(SUM(amount * unit_price),0)").
		Where("type = ?", "IN").
		Where("deleted_at IS NULL").
		Scan(&cost)

	db.
		Model(new(entity.StockTracking)).
		Select("COALESCE(SUM(amount * unit_price),0)").
		Where("type = ?", "OUT").
		Where("deleted_at IS NULL").
		Scan(&revenue)

	profit = revenue - cost

	currentBudget = masterItemsTotalBudget + profit

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

	baseQuery := db.Table("items").Where("items.deleted_at IS NULL")

	if request.SearchQuery != "" {
		baseQuery = baseQuery.Where("items.name LIKE ?", "%"+request.SearchQuery+"%")
	}

	// Hitung total data untuk pagination
	err := baseQuery.Count(&total).Error
	if err != nil {
		s.log.Errorf("failed to count items summary: %v", err)
		return nil, 0, err
	}

	if total == 0 {
		return []model.ItemStocksSummaryResponse{}, 0, nil
	}

	// Eksekusi query utama dengan kalkulasi dinamis
	err = baseQuery.
		Select(`
            items.id AS item_id,
            items.name AS name,
            items.category AS category,
            items.measure_unit AS measure_unit,
            items.stock AS current_stock,
            items.total_price AS stock_value,
            items.initial_stock AS initial_stock,
            
            -- Kalkulasi Harga Beli Rata-rata (Total Uang Masuk / Total Qty Masuk)
            -- Jika belum ada transaksi IN, fallback ke items.unit_price
            COALESCE(
                SUM(CASE WHEN st.type = 'IN' THEN st.total_price ELSE 0 END) / 
                NULLIF(SUM(CASE WHEN st.type = 'IN' THEN st.amount ELSE 0 END), 0), 
            items.unit_price) AS buy_price, 

            -- Kalkulasi Harga Jual Rata-rata (Total Uang Keluar / Total Qty Keluar)
            -- Jika belum ada transaksi OUT, fallback ke 0
            COALESCE(
                SUM(CASE WHEN st.type = 'OUT' THEN st.total_price ELSE 0 END) / 
                NULLIF(SUM(CASE WHEN st.type = 'OUT' THEN st.amount ELSE 0 END), 0), 
            0) AS sell_price,

            COALESCE(SUM(CASE WHEN st.type = 'IN' THEN st.amount ELSE 0 END), 0) AS total_in,
            COALESCE(SUM(CASE WHEN st.type = 'OUT' THEN st.amount ELSE 0 END), 0) AS total_out
        `).
		Joins("LEFT JOIN stock_tracks st ON st.item_id = items.id AND st.deleted_at IS NULL").
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
		Order("items.updated_at DESC").
		Limit(request.Size).
		Offset(offset).
		Find(&responses).Error

	if err != nil {
		s.log.Errorf("failed to fetch item stocks summary: %v", err)
		return nil, 0, err
	}

	return responses, total, nil
}
