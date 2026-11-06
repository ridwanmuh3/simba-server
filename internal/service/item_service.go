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

type ItemService struct {
	db             *gorm.DB
	log            *zap.SugaredLogger
	validate       *validator.Validate
	itemRepository ItemRepository
}

type ItemRepository interface {
	Save(db *gorm.DB, entity *entity.Item) error
	FindById(db *gorm.DB, entity *entity.Item, id any) error
	FindAll(db *gorm.DB, query *model.FindAllItemsRequest) ([]entity.Item, int64, error)
	AddBatches(db *gorm.DB, items []entity.Item) error
	NextItemNumber(db *gorm.DB) (int, error)
	FindItemForUpdate(db *gorm.DB, itemID string, dapurID uint) (*entity.Item, error)
	FindItemByIDTenant(db *gorm.DB, itemID string, dapurID uint) (*entity.Item, error)
	UpdateItemFields(db *gorm.DB, item *entity.Item) error
	FindCategories(db *gorm.DB, dapurID uint) ([]string, error)
	FindAllForExport(db *gorm.DB, dapurID uint) ([]entity.Item, error)
	DeleteTracksByItem(db *gorm.DB, itemID string, dapurID uint) error
	ListTracksChronological(db *gorm.DB, itemID string, dapurID uint) ([]entity.StockTracking, error)
	UpdateTracksMAC(db *gorm.DB, tracks []entity.StockTracking) error
	SaveActivityLog(db *gorm.DB, log *entity.ActivityLog) error
	DeleteItemHard(db *gorm.DB, item *entity.Item) error
}

var _ ItemRepository = (*repository.ItemRepository)(nil)

func NewItemService(db *gorm.DB, logger *zap.SugaredLogger, validate *validator.Validate, itemRepository ItemRepository) *ItemService {
	return &ItemService{db: db, log: logger, validate: validate, itemRepository: itemRepository}
}

func (s *ItemService) Add(ctx context.Context, request *model.AddItemRequest) (*model.ItemResponse, error) {
	// Validate before Begin: don't hold a connection for invalid input.
	if err := s.validate.Struct(request); err != nil {
		s.log.Errorf("failed to validate request body: %v", err)
		return nil, err
	}

	tx := s.db.WithContext(ctx).Begin()
	defer tx.Rollback()

	nextNum, err := s.itemRepository.NextItemNumber(tx)
	if err != nil {
		s.log.Errorf("failed to generate item ID: %v", err)
		return nil, exception.InternalServerError
	}
	newID := fmt.Sprintf("MBG-BHN-%04d", nextNum)

	stock := util.Round4(request.Stock)
	item := &entity.Item{
		ID:           newID,
		Name:         request.Name,
		Category:     request.Category,
		InitialStock: stock,
		Stock:        stock,
		UnitPrice:    request.UnitPrice,
		MeasureUnit:  request.MeasureUnit,
		TotalPrice:   util.GetTotalPrice(stock, request.UnitPrice),
		ModifiedBy:   request.ModifiedBy,
		DapurID:      request.DapurID,
	}
	if request.DateAdded != nil {
		item.CreatedAt = *request.DateAdded
	}

	if err := s.itemRepository.Save(tx, item); err != nil {
		s.log.Errorf("failed to save item to database: %v", err)
		return nil, exception.InternalServerError
	}
	if err := s.itemRepository.SaveActivityLog(tx, &entity.ActivityLog{
		Type:        "ADD-ITEM",
		Title:       "Bahan baru ditambahkan",
		Description: fmt.Sprintf("%s - %g %s", item.Name, item.InitialStock, item.MeasureUnit),
		ActionBy:    request.ModifiedBy,
		DapurID:     request.DapurID,
	}); err != nil {
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
	if err := s.validate.Struct(request); err != nil {
		s.log.Errorf("failed to validate request body: %v", err)
		return err
	}
	if err := util.ValidateSliceOfStruct(s.validate, request.Items); err != nil {
		s.log.Errorf("failed to validate items: %v", err)
		return err
	}

	tx := s.db.WithContext(ctx).Begin()
	defer tx.Rollback()

	nextNum, err := s.itemRepository.NextItemNumber(tx)
	if err != nil {
		s.log.Errorf("failed to generate item ID: %v", err)
		return exception.InternalServerError
	}

	var items []entity.Item
	for _, reqItem := range request.Items {
		newID := fmt.Sprintf("MBG-BHN-%04d", nextNum)
		nextNum++
		items = append(items, entity.Item{
			ID:           newID,
			Name:         reqItem.Name,
			Category:     reqItem.Category,
			Stock:        reqItem.Stock,
			InitialStock: reqItem.Stock,
			MeasureUnit:  reqItem.MeasureUnit,
			UnitPrice:    reqItem.UnitPrice,
			TotalPrice:   util.GetTotalPrice(reqItem.Stock, reqItem.UnitPrice),
			ModifiedBy:   reqItem.ModifiedBy,
			DapurID:      request.DapurID,
		})
	}

	if len(items) > 0 {
		if err := s.itemRepository.AddBatches(tx, items); err != nil {
			s.log.Errorf("failed to bulk insert items: %v", err)
			return exception.InternalServerError
		}
	}
	if err := s.itemRepository.SaveActivityLog(tx, &entity.ActivityLog{
		Type:        "ADD-ITEM-BATCHES",
		Title:       "Import bahan baru ditambahkan",
		Description: "Format CSV",
		ActionBy:    request.Items[0].ModifiedBy,
		DapurID:     request.DapurID,
	}); err != nil {
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
	if err := s.validate.Struct(request); err != nil {
		s.log.Errorf("failed to validate request body: %v", err)
		return nil, err
	}

	tx := s.db.WithContext(ctx).Begin()
	defer tx.Rollback()

	item, err := s.itemRepository.FindItemForUpdate(tx, request.ID, request.DapurID)
	if err != nil {
		s.log.Errorf("failed to find item by code: %v", err)
		return nil, exception.ItemNotFoundError
	}

	item.Name = request.Name
	item.Category = request.Category
	item.MeasureUnit = request.MeasureUnit
	item.ModifiedBy = request.ModifiedBy
	if request.DateAdded != nil {
		item.CreatedAt = *request.DateAdded
	}
	if request.InitialStock != nil {
		item.InitialStock = util.Round4(*request.InitialStock)
	}

	_, finalStock, finalTotalPrice, err := recalculateMAC(s.itemRepository, tx, item, request.DapurID)
	if err != nil {
		s.log.Errorf("failed to recalculate MAC: %v", err)
		if err == exception.InsufficientStockError {
			return nil, err
		}
		return nil, exception.InternalServerError
	}

	item.Stock = finalStock
	item.TotalPrice = finalTotalPrice
	item.UnitPrice = request.UnitPrice

	if err := s.itemRepository.UpdateItemFields(tx, item); err != nil {
		s.log.Errorf("failed to update item: %v", err)
		return nil, exception.InternalServerError
	}
	if err := tx.Commit().Error; err != nil {
		s.log.Errorf("failed to commit transaction: %v", err)
		return nil, exception.InternalServerError
	}
	return converter.ItemToResponse(item), nil
}

func (s *ItemService) Delete(ctx context.Context, request *model.DeleteItemRequest) (bool, error) {
	if err := s.validate.Struct(request); err != nil {
		return false, err
	}

	tx := s.db.WithContext(ctx).Begin()
	defer tx.Rollback()

	item, err := s.itemRepository.FindItemForUpdate(tx, request.ID, request.DapurID)
	if err != nil {
		return false, exception.ItemNotFoundError
	}
	if err := s.itemRepository.DeleteTracksByItem(tx, item.ID, request.DapurID); err != nil {
		return false, exception.InternalServerError
	}
	if err := s.itemRepository.DeleteItemHard(tx, item); err != nil {
		return false, exception.InternalServerError
	}
	if err := s.itemRepository.SaveActivityLog(tx, &entity.ActivityLog{
		Type:        "DELETE-ITEM",
		Title:       "Data bahan dihapus permanen",
		Description: item.Name,
		ActionBy:    request.ModifiedBy,
		DapurID:     request.DapurID,
	}); err != nil {
		return false, exception.InternalServerError
	}
	if err := tx.Commit().Error; err != nil {
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
	item, err := s.itemRepository.FindItemByIDTenant(db, request.ID, request.DapurID)
	if err != nil {
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

func (s *ItemService) GetItemCategories(ctx context.Context, dapurID uint) ([]string, error) {
	db := s.db.WithContext(ctx)
	categories, err := s.itemRepository.FindCategories(db, dapurID)
	if err != nil {
		s.log.Errorf("failed to get item categories: %v", err)
		return nil, exception.InternalServerError
	}
	return categories, nil
}

func (s *ItemService) ExportItems(ctx context.Context, dapurID uint) ([]model.ItemResponse, int, error) {
	db := s.db.WithContext(ctx)
	items, err := s.itemRepository.FindAllForExport(db, dapurID)
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
