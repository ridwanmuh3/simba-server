package service

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/go-playground/validator/v10"
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

// itemIDLockKey is the PostgreSQL advisory lock key that serializes item-ID
// generation. pg_advisory_xact_lock holds it for the caller's transaction
// lifetime — auto-released on commit or rollback — so concurrent inserts
// across multiple app instances can never read the same max ID.
const itemIDLockKey = 7283910

// nextItemNumber acquires an exclusive advisory lock scoped to tx, then reads
// the current max numeric suffix from item IDs and returns the next value.
// Callers must be inside an open transaction.
func nextItemNumber(tx *gorm.DB) (int, error) {
	if err := tx.Exec("SELECT pg_advisory_xact_lock(?)", itemIDLockKey).Error; err != nil {
		return 0, err
	}

	var lastItem entity.Item
	err := tx.Unscoped().
		Where("id ~ ?", `^MBG-BHN-[0-9]+$`).
		Order("CAST(split_part(id, '-', 3) AS INTEGER) DESC").
		First(&lastItem).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 1, nil
		}
		return 0, err
	}

	parts := strings.Split(lastItem.ID, "-")
	if len(parts) == 3 {
		n, parseErr := strconv.Atoi(parts[2])
		if parseErr == nil {
			return n + 1, nil
		}
	}

	return 0, fmt.Errorf("malformed last item ID: %s", lastItem.ID)
}

type ItemService struct {
	db             *gorm.DB
	log            *zap.SugaredLogger
	validate       *validator.Validate
	itemRepository ItemRepository
}

type ItemRepository interface {
	Save(db *gorm.DB, entity *entity.Item) error
	Update(db *gorm.DB, entity *entity.Item, id any) error
	FindById(db *gorm.DB, entity *entity.Item, id any) error
	FindAll(db *gorm.DB, query *model.FindAllItemsRequest) ([]entity.Item, int64, error)
	AddBatches(db *gorm.DB, items []entity.Item) error
}

var _ ItemRepository = (*repository.ItemRepository)(nil)

func NewItemService(db *gorm.DB, logger *zap.SugaredLogger, validate *validator.Validate, itemRepository ItemRepository) *ItemService {
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

	nextNum, err := nextItemNumber(tx)
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

	if err := tx.Create(&entity.ActivityLog{
		Type:        "ADD-ITEM",
		Title:       "Bahan baru ditambahkan",
		Description: fmt.Sprintf("%s - %g %s", item.Name, item.InitialStock, item.MeasureUnit),
		ActionBy:    request.ModifiedBy,
		DapurID:     request.DapurID,
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

	nextNum, err := nextItemNumber(tx)
	if err != nil {
		s.log.Errorf("failed to generate item ID: %v", err)
		return exception.InternalServerError
	}

	var items []entity.Item

	for _, reqItem := range request.Items {
		newID := fmt.Sprintf("MBG-BHN-%04d", nextNum)
		nextNum++

		item := entity.Item{
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
		DapurID:     request.DapurID,
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
		Where("id = ? AND dapur_id = ?", request.ID, request.DapurID).
		First(item).Error; err != nil {
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

	var stockTracks []entity.StockTracking
	if err := tx.Where("item_id = ? AND dapur_id = ? AND deleted_at IS NULL", item.ID, request.DapurID).
		Order("created_at ASC").
		Find(&stockTracks).Error; err != nil {
		s.log.Errorf("failed to fetch stock tracking records: %v", err)
		return nil, exception.InternalServerError
	}

	runningStock := item.InitialStock
	totalPrice := util.Round2(item.InitialStock * request.UnitPrice)

	for i := range stockTracks {
		prevStock := runningStock
		stockTracks[i].PreviousStock = util.Round4(prevStock)

		switch stockTracks[i].Type {
		case "IN":
			runningStock = util.Round4(runningStock + stockTracks[i].Amount)
			totalPrice = util.Round2(totalPrice + stockTracks[i].Amount*stockTracks[i].UnitPrice)
		case "OUT":
			if prevStock <= 0 {
				s.log.Errorf("invalid state: stock below zero")
				return nil, exception.InternalServerError
			}

			avg := totalPrice / prevStock
			deduction := util.Round2(avg * stockTracks[i].Amount)
			runningStock = util.Round4(runningStock - stockTracks[i].Amount)
			totalPrice = util.Round2(totalPrice - deduction)
		}

		stockTracks[i].NewStock = util.Round4(runningStock)
	}

	for i := range stockTracks {
		if err := tx.Model(&entity.StockTracking{}).
			Where("id = ?", stockTracks[i].ID).
			UpdateColumns(map[string]any{
				"previous_stock": stockTracks[i].PreviousStock,
				"new_stock":      stockTracks[i].NewStock,
			}).Error; err != nil {
			s.log.Errorf("failed to update stock tracking id=%d: %v", stockTracks[i].ID, err)
			return nil, exception.InternalServerError
		}
	}

	item.Stock = util.Round4(runningStock)
	item.TotalPrice = util.Round2(totalPrice)
	item.UnitPrice = request.UnitPrice

	if err := tx.Model(item).
		Select("Name", "Category", "MeasureUnit", "Stock", "InitialStock", "UnitPrice", "TotalPrice", "ModifiedBy", "CreatedAt").
		Updates(item).Error; err != nil {
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
	tx := s.db.WithContext(ctx).Begin()
	defer tx.Rollback()

	if err := s.validate.Struct(request); err != nil {
		return false, err
	}

	item := new(entity.Item)
	if err := tx.
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id = ? AND dapur_id = ?", request.ID, request.DapurID).
		First(item).Error; err != nil {
		return false, exception.ItemNotFoundError
	}

	if err := tx.
		Unscoped().
		Where("item_id = ? AND dapur_id = ?", item.ID, request.DapurID).
		Delete(&entity.StockTracking{}).Error; err != nil {
		return false, exception.InternalServerError
	}

	if err := tx.Unscoped().Delete(item).Error; err != nil {
		return false, exception.InternalServerError
	}

	if err := tx.Create(&entity.ActivityLog{
		Type:        "DELETE-ITEM",
		Title:       "Data bahan dihapus permanen",
		Description: item.Name,
		ActionBy:    request.ModifiedBy,
		DapurID:     request.DapurID,
	}).Error; err != nil {
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

	item := new(entity.Item)
	if err := db.Where("id = ? AND dapur_id = ?", request.ID, request.DapurID).Take(item).Error; err != nil {
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

	var categories []string
	if err := db.Model(new(entity.Item)).
		Where("dapur_id = ? AND category != ''", dapurID).
		Distinct("category").
		Order("category ASC").
		Pluck("category", &categories).Error; err != nil {
		s.log.Errorf("failed to get item categories: %v", err)
		return nil, exception.InternalServerError
	}

	return categories, nil
}

func (s *ItemService) ExportItems(ctx context.Context, dapurID uint) ([]model.ItemResponse, int, error) {
	db := s.db.WithContext(ctx)

	var items []entity.Item
	if err := db.Model(new(entity.Item)).
		Where("dapur_id = ?", dapurID).
		Order("created_at DESC, id DESC").Find(&items).Error; err != nil {
		s.log.Errorf("failed to export all items: %v", err)
		return nil, 0, exception.InternalServerError
	}

	responses := make([]model.ItemResponse, len(items))
	for i, item := range items {
		responses[i] = *converter.ItemToResponse(&item)
	}

	return responses, len(responses), nil
}
