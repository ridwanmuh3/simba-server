package service

import (
	"context"

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

	item := &entity.Item{
		Name:        request.Name,
		Category:    request.Category,
		Quantity:    request.Quantity,
		UnitPrice:   request.UnitPrice,
		MeasureUnit: request.MeasureUnit,
		TotalPrice:  request.UnitPrice * float64(request.Quantity),
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

	var items []entity.Item

	for _, reqItem := range request.Items {
		item := entity.Item{
			Name:        reqItem.Name,
			Category:    reqItem.Category,
			Quantity:    reqItem.Quantity,
			UnitPrice:   reqItem.UnitPrice,
			MeasureUnit: reqItem.MeasureUnit,
			TotalPrice:  reqItem.UnitPrice * float64(reqItem.Quantity),
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
	item.Quantity = request.Quantity
	item.MeasureUnit = request.MeasureUnit
	item.UnitPrice = request.UnitPrice
	item.TotalPrice = request.UnitPrice * float64(request.Quantity)
	item.ModifiedBy = request.ModifiedBy

	if err := s.itemRepository.Save(tx, item); err != nil {
		s.log.Errorf("failed to update item to database: %v", err)
		return nil, exception.InternalServerError
	}

	if err := tx.Commit(); err != nil {
		s.log.Errorf("failed to commit database transaction: %v", err)
		return nil, exception.InternalServerError
	}

	return converter.ItemToResponse(item), nil
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

	if err := tx.Commit(); err != nil {
		s.log.Errorf("failed to commit database transaction: %v", err)
		return nil, 0, exception.InternalServerError
	}

	responses := make([]model.ItemResponse, total)
	for i, item := range items {
		responses[i] = *converter.ItemToResponse(&item)
	}

	return responses, total, nil
}
