package service

import (
	"context"
	"fmt"
	"strconv"
	"strings"
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

type ItemService struct {
	db             *gorm.DB
	log            *zap.SugaredLogger
	validate       *validator.Validate
	itemRepository *repository.ItemRepository
}

func NewItemService(db *gorm.DB, logger *zap.SugaredLogger, validate *validator.Validate, itemRepository *repository.ItemRepository) *ItemService {
	return &ItemService{
		db:             db,
		log:            logger,
		validate:       validate,
		itemRepository: itemRepository,
	}
}

func (s *ItemService) Add(ctx context.Context, request *model.AddItemRequest) (*model.ItemResponse, error) {
	tx := s.db.WithContext(ctx).Begin()
	defer tx.Rollback()

	if err := s.validate.Struct(request); err != nil {
		s.log.Errorf("failed to validate request body: %v", err)
		return nil, err
	}

	count, err := s.itemRepository.CountByName(tx, request.Name)
	if err != nil {
		s.log.Errorf("failed to count item by name: %v", err)
		return nil, exception.InternalServerError
	}

	if count > 0 {
		s.log.Errorf("item already exists")
		return nil, exception.ItemAlreadyExistsError
	}

	var lastItem entity.Item
	err = tx.WithContext(ctx).Unscoped().Order("id DESC").First(&lastItem).Error

	currentNumber := 0
	if err == nil {
		parts := strings.Split(lastItem.ID, "-")
		if len(parts) == 3 {
			currentNumber, _ = strconv.Atoi(parts[2])
		}
	}

	currentNumber++
	newID := fmt.Sprintf("MBG-BHN-%04d", currentNumber)

	item := &entity.Item{
		ID:           newID,
		Name:         request.Name,
		Category:     request.Category,
		InitialStock: request.Stock,
		Stock:        request.Stock,
		UnitPrice:    request.UnitPrice,
		MeasureUnit:  request.MeasureUnit,
		TotalPrice:   request.UnitPrice * float64(request.Stock),
		ModifiedBy:   request.ModifiedBy,
	}

	if err := s.itemRepository.Save(tx, item); err != nil {
		s.log.Errorf("failed to save item to database: %v", err)
		return nil, exception.InternalServerError
	}

	if err := tx.Create(&entity.ActivityLog{
		Type:        "ADD-ITEM",
		Title:       "Bahan baru ditambahkan",
		Description: fmt.Sprintf("%s - %d %s", item.Name, item.InitialStock, item.MeasureUnit),
		ActionBy:    request.ModifiedBy,
	}).Error; err != nil {
		s.log.Errorf("failed to save activity log to database: %v", err)
		return nil, exception.InternalServerError
	}

	if err := tx.Commit().Error; err != nil {
		s.log.Errorf("failed to commit database transaction: %v", err)
		return nil, exception.InternalServerError
	}

	return converter.ItemToResponse(item), nil
}

func (s *ItemService) AddBatches(ctx context.Context, request *model.AddItemBatchRequest) error {
	tx := s.db.WithContext(ctx).Begin()
	defer tx.Rollback()

	if err := s.validate.Struct(request); err != nil {
		s.log.Errorf("failed to validate request body: %v", err)
		return err
	}

	if err := util.ValidateSliceOfStruct(s.validate, request.Items); err != nil {
		s.log.Errorf("failed to validate items: %v", err)
		return err
	}

	var lastItem entity.Item
	err := tx.WithContext(ctx).Unscoped().Order("id DESC").First(&lastItem).Error

	currentNumber := 0
	if err == nil {
		parts := strings.Split(lastItem.ID, "-")
		if len(parts) == 3 {
			currentNumber, _ = strconv.Atoi(parts[2])
		}
	}

	var items []entity.Item

	for _, reqItem := range request.Items {
		currentNumber++

		newID := fmt.Sprintf("MBG-BHN-%04d", currentNumber)

		item := entity.Item{
			ID:           newID,
			Name:         reqItem.Name,
			Category:     reqItem.Category,
			Stock:        reqItem.Stock,
			InitialStock: reqItem.Stock,
			MeasureUnit:  reqItem.MeasureUnit,
			UnitPrice:    reqItem.UnitPrice,
			TotalPrice:   reqItem.UnitPrice * float64(reqItem.Stock),
			ModifiedBy:   reqItem.ModifiedBy,
		}
		items = append(items, item)
	}

	if len(items) > 0 {
		if err := s.itemRepository.AddBatches(tx, items); err != nil {
			s.log.Errorf("failed to bulk insert items: %v", err)
			return exception.InternalServerError
		}
	}

	if err := tx.Create(&entity.ActivityLog{
		Type:        "ADD-ITEM-BATCHES",
		Title:       "Import bahan baru ditambahkan",
		Description: "Format CSV",
		ActionBy:    request.Items[0].ModifiedBy,
	}).Error; err != nil {
		s.log.Errorf("failed to save activity log to database: %v", err)
		return exception.InternalServerError
	}

	if err := tx.Commit().Error; err != nil {
		s.log.Errorf("failed to commit database transaction: %v", err)
		return err
	}

	return nil
}

func (s *ItemService) Update(ctx context.Context, request *model.UpdateItemRequest) (*model.ItemResponse, error) {
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

	item.Name = request.Name
	item.Category = request.Category
	item.MeasureUnit = request.MeasureUnit
	item.UnitPrice = request.UnitPrice
	item.TotalPrice = request.UnitPrice * float64(item.Stock)
	item.ModifiedBy = request.ModifiedBy

	if err := s.itemRepository.Update(tx, item, item.ID); err != nil {
		s.log.Errorf("failed to update item to database: %v", err)
		return nil, exception.InternalServerError
	}

	if err := tx.Commit().Error; err != nil {
		s.log.Errorf("failed to commit database transaction: %v", err)
		return nil, exception.InternalServerError
	}

	return converter.ItemToResponse(item), nil
}

func (s *ItemService) UpdateStock(ctx context.Context, request *model.UpdateItemStockRequest) (*model.StockResponse, error) {
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
		stockTracking.NewStock = stockTracking.PreviousStock + request.Amount
	case "OUT":
		if stockTracking.PreviousStock < request.Amount {
			s.log.Errorf("failed to decrease item stock: insufficient stock")
			return nil, fiber.NewError(fiber.StatusBadRequest, "stock must be sufficient enough to decreased")
		}
		stockTracking.NewStock = stockTracking.PreviousStock - request.Amount
	default:
		s.log.Errorf("invalid stock type '%s'", request.Type)
		return nil, fiber.NewError(fiber.StatusBadRequest, "invalid update stock type")
	}

	stockTracking.TotalPrice = request.UnitPrice * float64(request.Amount)
	item.Stock = stockTracking.NewStock
	item.TotalPrice = float64(item.Stock) * item.UnitPrice

	if err := s.itemRepository.Update(tx, item, item.ID); err != nil {
		s.log.Errorf("failed to update item stock to database: %v", err)
		return nil, exception.InternalServerError
	}

	if err := tx.Create(stockTracking).Error; err != nil {
		s.log.Errorf("failed to create stock tracking to database: %v", err)
		return nil, exception.InternalServerError
	}

	if err := tx.Create(&entity.ActivityLog{
		Type:        "UPDATE-STOCK",
		Title:       "Stok bahan diperbarui",
		Description: fmt.Sprintf("%s - %d %s", item.Name, item.Stock, item.MeasureUnit),
		ActionBy:    request.ModifiedBy,
	}).Error; err != nil {
		s.log.Errorf("failed to save activity log to database: %v", err)
		return nil, exception.InternalServerError
	}

	if err := tx.Commit().Error; err != nil {
		s.log.Errorf("failed to commit database transaction: %v", err)
		return nil, exception.InternalServerError
	}

	return converter.StockToResponse(stockTracking), nil
}

func (s *ItemService) Delete(ctx context.Context, request *model.DeleteItemRequest) (bool, error) {
	tx := s.db.WithContext(ctx).Begin()
	defer tx.Rollback()

	if err := s.validate.Struct(request); err != nil {
		s.log.Errorf("failed to validate request body: %v", err)
		return false, err
	}

	item := new(entity.Item)
	if err := s.itemRepository.FindById(tx, item, request.ID); err != nil {
		s.log.Errorf("failed to find item by id: %v", err)
		return false, exception.UserNotFoundError
	}

	if err := s.itemRepository.Delete(tx, item); err != nil {
		s.log.Errorf("failed to delete item by id: %v", err)
		return false, exception.InternalServerError
	}

	if err := tx.Create(&entity.ActivityLog{
		Type:        "DELETE-ITEM",
		Title:       "Data bahan dihapus",
		Description: item.Name,
		ActionBy:    item.ModifiedBy,
	}).Error; err != nil {
		s.log.Errorf("failed to save activity log to database: %v", err)
		return false, exception.InternalServerError
	}

	if err := tx.Commit().Error; err != nil {
		s.log.Errorf("failed to commit database transaction: %v", err)
		return false, exception.InternalServerError
	}

	return true, nil
}

func (s *ItemService) DeleteStock(ctx context.Context, request *model.DeleteStockRequest) (bool, error) {
	tx := s.db.WithContext(ctx).Begin()
	defer tx.Rollback()

	if err := s.validate.Struct(request); err != nil {
		s.log.Errorf("failed to validate request body: %v", err)
		return false, err
	}

	item := new(entity.Item)
	if err := s.itemRepository.FindById(tx, item, request.ID); err != nil {
		s.log.Errorf("failed to find item by id: %v", err)
		return false, exception.ItemNotFoundError
	}

	stock := new(entity.StockTracking)
	if err := tx.Where("id = ? AND item_id = ?", request.StockID, request.ID).First(stock).Error; err != nil {
		s.log.Errorf("failed to find stock item by id: %v", err)
		return false, exception.ItemNotFoundError
	}

	switch stock.Type {
	case "IN":
		item.Stock = item.Stock - stock.Amount
	case "OUT":
		item.Stock = item.Stock + stock.Amount
	}

	if item.Stock < 0 {
		s.log.Errorf("deletion resulted in negative stock")
		return false, fiber.NewError(fiber.StatusBadRequest, "reducing this stock item will resulting negative stock")
	}

	if err := tx.Where("id = ? AND item_id = ?", request.StockID, item.ID).Delete(stock).Error; err != nil {
		s.log.Errorf("failed to delete stock item by id: %v", err)
		return false, exception.InternalServerError
	}

	if err := s.itemRepository.Save(tx, item); err != nil {
		s.log.Errorf("failed to save stock item: %v", err)
		return false, exception.InternalServerError
	}

	if err := tx.Create(&entity.ActivityLog{
		Type:        "REDUCE-STOCK",
		Title:       "Data stock bahan diperbarui",
		Description: fmt.Sprintf("%s - %d %s", item.Name, item.Stock, item.MeasureUnit),
		ActionBy:    item.ModifiedBy,
	}).Error; err != nil {
		s.log.Errorf("failed to save activity log to database: %v", err)
		return false, exception.InternalServerError
	}

	if err := tx.Commit().Error; err != nil {
		s.log.Errorf("failed to commit database transaction: %v", err)
		return false, exception.InternalServerError
	}

	return true, nil
}

func (s *ItemService) FindById(ctx context.Context, request *model.FindByIdItemRequest) (*model.ItemResponse, error) {
	db := s.db.WithContext(ctx)

	if err := s.validate.Struct(request); err != nil {
		s.log.Errorf("failed to validate request body: %v", err)
		return nil, err
	}

	item := new(entity.Item)
	if err := s.itemRepository.FindById(db, item, request.ID); err != nil {
		s.log.Errorf("failed to find item by id: %v", err)
		return nil, exception.ItemNotFoundError
	}

	return converter.ItemToResponse(item), nil
}

func (s *ItemService) FindAll(ctx context.Context, request *model.FindAllItemsRequest) ([]model.ItemResponse, int64, error) {
	db := s.db.WithContext(ctx)

	if err := s.validate.Struct(request); err != nil {
		s.log.Errorf("failed to validate request body: %v", err)
		return nil, 0, err
	}

	items, total, err := s.itemRepository.FindAll(db, request)
	if err != nil {
		s.log.Errorf("failed to find all items: %v", err)
		return nil, 0, exception.InternalServerError
	}

	responses := make([]model.ItemResponse, len(items))
	for i, item := range items {
		responses[i] = *converter.ItemToResponse(&item)
	}

	return responses, total, nil
}

func (s *ItemService) FindAllStocks(ctx context.Context, request *model.FindAllStocksRequest) ([]model.StockResponse, int64, error) {
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

func (s *ItemService) GetStocksFinanceSummary(ctx context.Context) (*model.StocksFinanceSummaryResponse, error) {
	db := s.db.WithContext(ctx)

	var (
		masterItemsTotalBudget,
		budgetIn,
		budgetOut,
		profit,
		currentBudget float64
	)

	db.
		Model(new(entity.Item)).
		Select("COALESCE(SUM(total_price),0)").
		Scan(&masterItemsTotalBudget)

	db.
		Model(new(entity.StockTracking)).
		Select("COALESCE(SUM(amount * unit_price),0)").
		Where("type = ?", "IN").
		Scan(&budgetIn)

	db.
		Model(new(entity.StockTracking)).
		Select("COALESCE(SUM(amount * unit_price),0)").
		Where("type = ?", "OUT").
		Scan(&budgetOut)

	profit = budgetOut - budgetIn
	currentBudget = masterItemsTotalBudget + profit

	return &model.StocksFinanceSummaryResponse{
		MasterItemsTotalBudget: masterItemsTotalBudget,
		BudgetIn:               budgetIn,
		BudgetOut:              budgetOut,
		Profit:                 profit,
		CurrentBudget:          currentBudget,
	}, nil
}

func (s *ItemService) ExportItems(ctx context.Context) ([]model.ItemResponse, int, error) {
	db := s.db.WithContext(ctx)

	var items []entity.Item
	err := db.Model(new(entity.Item)).Order("created_at DESC").Find(&items).Error
	if err != nil {
		s.log.Errorf("failed to export all items: %v", err)
		return nil, 0, exception.InternalServerError
	}

	responses := make([]model.ItemResponse, len(items))
	for i, item := range items {
		responses[i] = *converter.ItemToResponse(&item)
	}

	return responses, len(responses), nil
}

func (s *ItemService) GetInvoiceItems(ctx context.Context, request *model.GetInvoiceItemsRequest) ([]model.StockResponse, error) {
	db := s.db.WithContext(ctx)

	if err := s.validate.Struct(request); err != nil {
		s.log.Errorf("failed to validate request body: %v", err)
		return nil, err
	}

	var stocks []entity.StockTracking
	query := db.Model(new(entity.StockTracking)).Preload("Item").Where("type = ?", "OUT")

	if request.DateFrom != "" {
		parsedFrom, err := time.Parse(time.RFC3339, request.DateFrom)
		if err != nil {
			s.log.Errorf("failed to parse date from: %v", err)
			return nil, exception.InternalServerError
		}
		query = query.Where("created_at >= ?", parsedFrom)
	}

	if request.DateTo != "" {
		parsedTo, err := time.Parse(time.RFC3339, request.DateTo)
		if err != nil {
			s.log.Errorf("failed to parse date to: %v", err)
			return nil, exception.InternalServerError
		}
		query = query.Where("created_at <= ?", parsedTo)
	}

	if err := query.Order("item_id ASC").Limit(500).Find(&stocks).Error; err != nil {
		s.log.Errorf("failed to get invoice items: %v", err)
		return nil, exception.InternalServerError
	}

	responses := make([]model.StockResponse, len(stocks))
	for i, stock := range stocks {
		responses[i] = *converter.StockToResponse(&stock)
	}

	return responses, nil
}

func (s *ItemService) GetItemStocksSummary(ctx context.Context, request *model.GetItemStockSummaryRequest) ([]model.ItemStocksSummaryResponse, int64, error) {
	db := s.db.WithContext(ctx)

	if err := s.validate.Struct(request); err != nil {
		s.log.Errorf("failed to validate request body: %v", err)
		return nil, 0, err
	}

	var responses []model.ItemStocksSummaryResponse
	var total int64
	offset := (request.Page - 1) * request.Size

	err := db.Table("items").
		Joins("JOIN stock_tracks st ON st.item_id = items.id").
		Distinct("items.id").
		Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	err = db.Table("items").
		Select(`
			items.id AS item_id,
			items.name AS name,
			items.category AS category,
			items.measure_unit AS measure_unit,
			items.stock AS current_stock,
			(items.stock * items.unit_price) AS stock_value,
			COALESCE(SUM(CASE WHEN st.type = 'IN' THEN st.amount ELSE 0 END), 0) AS total_in,
			COALESCE(SUM(CASE WHEN st.type = 'OUT' THEN st.amount ELSE 0 END), 0) AS total_out,
			(items.stock
			 - COALESCE(SUM(CASE WHEN st.type = 'IN' THEN st.amount ELSE 0 END), 0)
			 + COALESCE(SUM(CASE WHEN st.type = 'OUT' THEN st.amount ELSE 0 END), 0)) AS initial_stock
		`).
		Joins("JOIN stock_tracks st ON st.item_id = items.id").
		Group("items.id, items.name, items.category, items.measure_unit, items.stock, items.unit_price").
		Order("items.updated_at DESC").
		Limit(request.Size).
		Offset(offset).
		Find(&responses).Error

	if err != nil {
		return nil, 0, err
	}

	return responses, total, nil
}
