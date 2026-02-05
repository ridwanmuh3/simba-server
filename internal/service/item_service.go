package service

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
	"gorm.io/gorm"

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
		ID:          newID,
		Name:        request.Name,
		Category:    request.Category,
		Stock:       request.Stock,
		UnitPrice:   request.UnitPrice,
		MeasureUnit: request.MeasureUnit,
		TotalPrice:  request.UnitPrice * float64(request.Stock),
		ModifiedBy:  request.ModifiedBy,
	}

	if err := s.itemRepository.Save(tx, item); err != nil {
		s.log.Errorf("failed to save item to database: %v", err)
		return nil, err
	}

	if err := tx.Commit().Error; err != nil {
		s.log.Errorf("failed to commit database transaction: %v", err)
		return nil, err
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
			ID:          newID,
			Name:        reqItem.Name,
			Category:    reqItem.Category,
			Stock:       reqItem.Stock,
			MeasureUnit: reqItem.MeasureUnit,
			UnitPrice:   reqItem.UnitPrice,
			TotalPrice:  reqItem.UnitPrice * float64(reqItem.Stock),
			ModifiedBy:  reqItem.ModifiedBy,
		}
		items = append(items, item)
	}

	if len(items) > 0 {
		if err := s.itemRepository.AddBatches(tx, items); err != nil {
			s.log.Errorf("failed to bulk insert items: %v", err)
			return err
		}
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
	if err := s.itemRepository.FindById(tx, item, request.ID); err != nil {
		s.log.Errorf("failed to find item by code: %v", err)
		return nil, exception.ItemNotFoundError
	}

	item.Name = request.Name
	item.Category = request.Category
	item.MeasureUnit = request.MeasureUnit
	item.UnitPrice = request.UnitPrice
	item.TotalPrice = request.UnitPrice * float64(item.Stock)
	item.ModifiedBy = request.ModifiedBy

	if err := s.itemRepository.Save(tx, item); err != nil {
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
	if err := s.itemRepository.FindById(tx, item, request.ID); err != nil {
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
	stockTracking.Supplier = request.Supplier

	switch request.Type {
	case "IN":
		stockTracking.NewStock = stockTracking.PreviousStock + request.Amount
	case "OUT":
		if stockTracking.PreviousStock < request.Amount {
			s.log.Errorf("failed to decrease item stock")
			return nil, fiber.NewError(fiber.StatusBadRequest, "stock must be sufficient enough to decreased")
		}
		stockTracking.NewStock = stockTracking.PreviousStock - request.Amount
	default:
		s.log.Errorf("invalid stock type '%s'", request.Type)
		return nil, fiber.NewError(fiber.StatusBadRequest, "invalid update stock type")
	}

	item.Stock = stockTracking.NewStock

	if err := s.itemRepository.Save(tx, item); err != nil {
		s.log.Errorf("failed to update item stock to database: %v", err)
		return nil, exception.InternalServerError
	}

	if err := tx.Save(stockTracking).Error; err != nil {
		s.log.Errorf("failed to create stock tracking to database: %v", err)
		return nil, exception.InternalServerError
	}

	if err := tx.Commit().Error; err != nil {
		s.log.Errorf("failed to commit database transaction: %v", err)
		return nil, exception.InternalServerError
	}

	return nil, nil
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
	if err := tx.Where("id = ? AND item_id = ?", request.ID, request.StockID).First(stock).Error; err != nil {
		s.log.Errorf("failed to find stock item by id: %v", err)
		return false, exception.ItemNotFoundError
	}

	item.Stock = stock.PreviousStock

	if err := tx.Where("id = ? AND item_id = ?", request.ID, item.ID).Delete(stock).Error; err != nil {
		s.log.Errorf("failed to delete stock item by id: %v", err)
		return false, exception.InternalServerError
	}

	if err := tx.Commit().Error; err != nil {
		s.log.Errorf("failed to commit database transaction: %v", err)
		return false, exception.InternalServerError
	}

	return true, nil
}

func (s *ItemService) FindById(ctx context.Context, request *model.FindByIdItemRequest) (*model.ItemResponse, error) {
	tx := s.db.WithContext(ctx).Begin()
	defer tx.Rollback()

	if err := s.validate.Struct(request); err != nil {
		s.log.Errorf("failed to validate request body: %v", err)
		return nil, err
	}

	item := new(entity.Item)
	if err := s.itemRepository.FindById(tx, item, request.ID); err != nil {
		s.log.Errorf("failed to find item by id: %v", err)
		return nil, exception.ItemNotFoundError
	}

	if err := tx.Commit().Error; err != nil {
		s.log.Errorf("failed to commit database transaction: %v", err)
		return nil, exception.InternalServerError
	}

	return converter.ItemToResponse(item), nil
}

func (s *ItemService) FindAll(ctx context.Context, request *model.FindAllItemsRequest) ([]model.ItemResponse, int64, error) {
	tx := s.db.WithContext(ctx).Begin()
	defer tx.Rollback()

	if err := s.validate.Struct(request); err != nil {
		s.log.Errorf("failed to validate request body: %v", err)
		return nil, 0, err
	}

	items, total, err := s.itemRepository.FindAll(tx, request)
	if err != nil {
		s.log.Errorf("failed to find all items: %v", err)
		return nil, 0, exception.InternalServerError
	}

	if err := tx.Commit().Error; err != nil {
		s.log.Errorf("failed to commit database transaction: %v", err)
		return nil, 0, exception.InternalServerError
	}

	responses := make([]model.ItemResponse, len(items))
	for i, item := range items {
		responses[i] = *converter.ItemToResponse(&item)
	}

	return responses, total, nil
}

func (s *ItemService) FindAllStocks(ctx context.Context, request *model.FindAllStocksRequest) ([]model.StockResponse, int64, error) {
	tx := s.db.WithContext(ctx).Begin()
	defer tx.Rollback()

	stocksTracking, total, err := s.itemRepository.FindAllStocks(tx, request)
	if err != nil {
		s.log.Errorf("failed to find all items stocks: %v", err)
		return nil, 0, exception.InternalServerError
	}

	if err := tx.Commit().Error; err != nil {
		s.log.Errorf("failed to commit database transaction: %v", err)
		return nil, 0, exception.InternalServerError
	}

	responses := make([]model.StockResponse, len(stocksTracking))
	for i, stock := range stocksTracking {
		responses[i] = *converter.StockToResponse(&stock)
	}

	return responses, total, nil
}

func (s *ItemService) ExportItems(ctx context.Context) ([]model.ItemResponse, int, error) {
	tx := s.db.WithContext(ctx).Begin()
	defer tx.Rollback()

	var items []entity.Item
	err := tx.Model(new(entity.Item)).Order("id ASC").Find(&items).Error
	if err != nil {
		s.log.Errorf("failed to export all items: %v", err)
		return nil, 0, exception.InternalServerError
	}

	if err := tx.Commit().Error; err != nil {
		s.log.Errorf("failed to commit database transaction: %v", err)
		return nil, 0, exception.InternalServerError
	}

	responses := make([]model.ItemResponse, len(items))
	for i, item := range items {
		responses[i] = *converter.ItemToResponse(&item)
	}

	return responses, len(responses), nil
}
