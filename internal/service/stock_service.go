package service

import (
	"context"
	"errors"
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

	// 2. LOCK ITEM
	item := new(entity.Item)
	if err := tx.
		Clauses(clause.Locking{Strength: "UPDATE"}).
		First(item, "id = ?", request.ID).Error; err != nil {
		return nil, exception.ItemNotFoundError
	}

	// 3. GET LAST TRACKING (DETERMINISTIC)
	var last entity.StockTracking
	err := tx.
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("item_id = ? AND deleted_at IS NULL", item.ID).
		Order("created_at DESC, id DESC"). // 🔥 penting (hindari collision)
		First(&last).Error

	var previousStock float64
	var currentTotalPrice float64

	if err == nil {
		// 🔥 source of truth dari tracking terakhir
		previousStock = last.NewStock

		// totalPrice tetap dari item (karena itu agregat resmi)
		currentTotalPrice = item.TotalPrice
	} else if errors.Is(err, gorm.ErrRecordNotFound) {
		// no tracking yet — use item master as baseline
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
	}

	// 5. CORE (MAC SAFE, BASED ON previousStock)
	switch request.Type {

	case "IN":
		newStock := util.Round4(previousStock + request.Amount)
		addedTotal := util.Round2(request.Amount * request.UnitPrice)

		item.Stock = newStock
		item.TotalPrice = util.Round2(currentTotalPrice + addedTotal)

		tr.NewStock = newStock
		tr.TotalPrice = addedTotal

	case "OUT":
		if previousStock == 0 {
			return nil, fiber.NewError(fiber.StatusBadRequest, "invalid stock state")
		}
		if util.Round4(previousStock) < util.Round4(request.Amount) {
			return nil, fiber.NewError(fiber.StatusBadRequest, "insufficient stock")
		}

		avg := util.Round4(currentTotalPrice / previousStock)

		newStock := util.Round4(previousStock - request.Amount)
		deduction := util.Round2(avg * request.Amount)                        // MAC (inventory)
		lineTotal := util.Round2(request.Amount * request.UnitPrice) // invoice

		item.Stock = newStock
		item.TotalPrice = util.Round2(currentTotalPrice - deduction)

		tr.NewStock = newStock
		tr.TotalPrice = lineTotal

	default:
		return nil, fiber.NewError(fiber.StatusBadRequest, "invalid stock type")
	}

	if tr.NewStock < 0 {
		return nil, fiber.NewError(fiber.StatusBadRequest, "stock cannot be negative")
	}

	// 6. SAVE ITEM
	if err := s.itemRepository.Update(tx, item, item.ID); err != nil {
		return nil, exception.InternalServerError
	}

	// 7. SAVE TRACKING
	if err := tx.Create(tr).Error; err != nil {
		return nil, exception.InternalServerError
	}

	// 8. ACTIVITY
	if err := tx.Create(&entity.ActivityLog{
		Type:  "UPDATE-STOCK",
		Title: "Stok bahan diperbarui",
		Description: fmt.Sprintf(
			"%s: %.2f → %.2f %s",
			item.Name,
			tr.PreviousStock,
			tr.NewStock,
			item.MeasureUnit,
		),
		ActionBy: request.ModifiedBy,
	}).Error; err != nil {
		return nil, exception.InternalServerError
	}

	// 9. COMMIT
	if err := tx.Commit().Error; err != nil {
		return nil, exception.InternalServerError
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

	// 2. LOCK ITEM
	item := new(entity.Item)
	if err := tx.
		Clauses(clause.Locking{Strength: "UPDATE"}).
		First(item, "id = ?", request.ID).Error; err != nil {
		return nil, exception.ItemNotFoundError
	}

	// 3. LOCK & MUTATE TARGET TRACKING
	stock := new(entity.StockTracking)
	if err := tx.
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id = ? AND item_id = ?", request.StockID, request.ID).
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

	// 4. GET ALL TRACKING IN ORDER
	var tracks []entity.StockTracking
	if err := tx.
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("item_id = ?", item.ID).
		Order("created_at ASC, id ASC").
		Find(&tracks).Error; err != nil {
		return nil, exception.InternalServerError
	}

	// 5. REBUILD MAC
	var runningStock float64 = item.InitialStock
	var totalPrice float64 = util.Round2(item.InitialStock * item.UnitPrice)

	var updatedTarget *entity.StockTracking

	for i := range tracks {
		prev := runningStock
		tracks[i].PreviousStock = util.Round4(prev)

		switch tracks[i].Type {

		case "IN":
			added := util.Round2(tracks[i].Amount * tracks[i].UnitPrice)

			runningStock = util.Round4(prev + tracks[i].Amount)
			totalPrice = util.Round2(totalPrice + added)

			tracks[i].TotalPrice = util.Round2(tracks[i].Amount * tracks[i].UnitPrice)

		case "OUT":
			if prev < tracks[i].Amount {
				return nil, fiber.NewError(fiber.StatusBadRequest, "edit would result in invalid stock history")
			}

			avg := util.Round4(totalPrice / prev)

			deduction := util.Round2(avg * tracks[i].Amount)
			lineTotal := util.Round2(tracks[i].Amount * tracks[i].UnitPrice)

			runningStock = util.Round4(prev - tracks[i].Amount)
			totalPrice = util.Round2(totalPrice - deduction)

			tracks[i].TotalPrice = lineTotal
		}

		tracks[i].NewStock = util.Round4(runningStock)

		if runningStock < 0 {
			return nil, fiber.NewError(fiber.StatusBadRequest, "edit would cause stock to go negative")
		}

		if tracks[i].ID == uint(request.StockID) {
			updatedTarget = &tracks[i]
		}
	}

	// 6. UPDATE ALL TRACKING
	for i := range tracks {
		if err := tx.Model(&entity.StockTracking{}).
			Where("id = ?", tracks[i].ID).
			Updates(map[string]any{
				"previous_stock": tracks[i].PreviousStock,
				"new_stock":      tracks[i].NewStock,
				"total_price":    tracks[i].TotalPrice,
			}).Error; err != nil {
			return nil, exception.InternalServerError
		}
	}

	// 7. UPDATE ITEM
	item.Stock = util.Round4(runningStock)
	item.TotalPrice = util.Round2(totalPrice)

	if err := tx.Save(item).Error; err != nil {
		return nil, exception.InternalServerError
	}

	// 8. ACTIVITY LOG
	if err := tx.Create(&entity.ActivityLog{
		Type:  "EDIT-STOCK-" + stock.Type,
		Title: "Stock entry edited and chain recalculated",
		Description: fmt.Sprintf(
			"%s [%s]: %.2f @ %.2f → stock %.2f %s",
			item.Name,
			stock.Type,
			request.Amount,
			request.UnitPrice,
			item.Stock,
			item.MeasureUnit,
		),
		ActionBy: request.ModifiedBy,
	}).Error; err != nil {
		return nil, exception.InternalServerError
	}

	// 9. COMMIT
	if err := tx.Commit().Error; err != nil {
		return nil, exception.InternalServerError
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

	// 2. LOCK ITEM
	item := new(entity.Item)
	if err := tx.
		Clauses(clause.Locking{Strength: "UPDATE"}).
		First(item, "id = ?", request.ID).Error; err != nil {
		return false, exception.ItemNotFoundError
	}

	// 3. GET TARGET TRACKING
	stock := new(entity.StockTracking)
	if err := tx.
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id = ? AND item_id = ?", request.StockID, request.ID).
		First(stock).Error; err != nil {
		return false, exception.ItemNotFoundError
	}

	// 4. HARD DELETE
	if err := tx.
		Unscoped().
		Delete(&entity.StockTracking{}, "id = ?", stock.ID).Error; err != nil {
		return false, exception.InternalServerError
	}

	// 5. GET ALL TRACKING (ORDER FIX)
	var tracks []entity.StockTracking
	if err := tx.
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("item_id = ?", item.ID).
		Order("created_at ASC, id ASC"). // 🔥 penting
		Find(&tracks).Error; err != nil {
		return false, exception.InternalServerError
	}

	// 6. REBUILD MAC (seed from InitialStock/UnitPrice so items without IN-tracking don't break)
	var runningStock float64 = item.InitialStock
	var totalPrice float64 = util.Round2(item.InitialStock * item.UnitPrice)

	for i := range tracks {

		prev := runningStock
		tracks[i].PreviousStock = util.Round4(prev)

		switch tracks[i].Type {

		case "IN":
			added := util.Round2(tracks[i].Amount * tracks[i].UnitPrice)

			runningStock = util.Round4(prev + tracks[i].Amount)
			totalPrice = util.Round2(totalPrice + added)

			tracks[i].TotalPrice = util.Round2(tracks[i].Amount * tracks[i].UnitPrice)

		case "OUT":
			if prev < tracks[i].Amount {
				return false, fiber.NewError(fiber.StatusBadRequest, "invalid stock history")
			}

			avg := util.Round4(totalPrice / prev)

			deduction := util.Round2(avg * tracks[i].Amount)
			lineTotal := util.Round2(tracks[i].Amount * tracks[i].UnitPrice)

			runningStock = util.Round4(prev - tracks[i].Amount)
			totalPrice = util.Round2(totalPrice - deduction)

			tracks[i].TotalPrice = lineTotal
		}

		tracks[i].NewStock = util.Round4(runningStock)

		if runningStock < 0 {
			return false, fiber.NewError(fiber.StatusBadRequest, "stock became negative")
		}
	}

	// 7. UPDATE TRACKING (CHAIN FIX)
	for i := range tracks {
		if err := tx.Model(&entity.StockTracking{}).
			Where("id = ?", tracks[i].ID).
			Updates(map[string]any{
				"previous_stock": tracks[i].PreviousStock,
				"new_stock":      tracks[i].NewStock,
				"total_price":    tracks[i].TotalPrice,
			}).Error; err != nil {
			return false, exception.InternalServerError
		}
	}

	// 8. UPDATE ITEM
	item.Stock = util.Round4(runningStock)
	item.TotalPrice = util.Round2(totalPrice)

	if err := tx.Save(item).Error; err != nil {
		return false, exception.InternalServerError
	}

	// 9. ACTIVITY LOG
	action := "DELETE-STOCK"
	if stock.Type == "IN" {
		action = "DELETE-STOCK-IN"
	} else {
		action = "DELETE-STOCK-OUT"
	}

	if err := tx.Create(&entity.ActivityLog{
		Type:  action,
		Title: "Stock recalculated after delete",
		Description: fmt.Sprintf(
			"%s: %.2f %s",
			item.Name,
			item.Stock,
			item.MeasureUnit,
		),
		ActionBy: request.ModifiedBy,
	}).Error; err != nil {
		return false, exception.InternalServerError
	}

	// 10. COMMIT
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
