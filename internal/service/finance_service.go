package service

import (
	"context"
	"fmt"
	"time"

	"github.com/go-playground/validator/v10"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/ridwanmuh3/simba-server/internal/entity"
	"github.com/ridwanmuh3/simba-server/internal/exception"
	"github.com/ridwanmuh3/simba-server/internal/model"
	"github.com/ridwanmuh3/simba-server/internal/model/converter"
)

type FinanceService struct {
	db       *gorm.DB
	log      *zap.SugaredLogger
	validate *validator.Validate
}

func NewFinanceService(db *gorm.DB, logger *zap.SugaredLogger, validate *validator.Validate) *FinanceService {
	return &FinanceService{
		db:       db,
		log:      logger,
		validate: validate,
	}
}

func (s *FinanceService) Add(ctx context.Context, request *model.AddFinanceRequest) (*model.FinanceResponse, error) {
	tx := s.db.WithContext(ctx).Begin()
	defer tx.Rollback()

	if err := s.validate.Struct(request); err != nil {
		s.log.Errorf("failed to validate request body: %v", err)
		return nil, err
	}

	finance := new(entity.Finance)
	finance.Category = request.Category
	finance.Description = request.Description
	finance.Amount = request.Amount
	finance.ExtraNote = request.ExtraNote
	finance.ProofImage = request.ProofImage

	if err := tx.Save(finance).Error; err != nil {
		s.log.Errorf("failed to save finance data: %v", err)
		return nil, exception.InternalServerError
	}

	if err := tx.Create(&entity.ActivityLog{
		Type:        "ADD-FINANCE",
		Title:       "Data keuangan ditambahkan",
		Description: fmt.Sprintf("%s - %s", finance.Category, finance.Description),
	}).Error; err != nil {
		s.log.Errorf("failed to save activity log to database: %v", err)
		return nil, exception.InternalServerError
	}

	if err := tx.Commit().Error; err != nil {
		s.log.Errorf("failed to commit transaction database: %v", err)
		return nil, exception.InternalServerError
	}

	return converter.FinanceToResponse(finance), nil
}

func (s *FinanceService) Update(ctx context.Context, request *model.UpdateFinanceRequest) (*model.FinanceResponse, error) {
	tx := s.db.WithContext(ctx).Begin()
	defer tx.Rollback()

	if err := s.validate.Struct(request); err != nil {
		s.log.Errorf("failed to validate request body: %v", err)
		return nil, err
	}

	finance := new(entity.Finance)
	if err := tx.Where("id = ?", request.ID).First(finance).Error; err != nil {
		s.log.Errorf("failed to find finance by id: %v", err)
		return nil, exception.FinanceNotFound
	}

	finance.Category = request.Category
	finance.Description = request.Description
	finance.Amount = request.Amount
	finance.ExtraNote = request.ExtraNote
	finance.ProofImage = request.ProofImage
	finance.ModifiedBy = request.ModifiedBy

	if err := tx.Save(finance).Error; err != nil {
		s.log.Errorf("failed to update finance data: %v", err)
		return nil, exception.InternalServerError
	}

	if err := tx.Commit().Error; err != nil {
		s.log.Errorf("failed to commit transaction database: %v", err)
		return nil, exception.InternalServerError
	}

	return converter.FinanceToResponse(finance), nil
}

func (s *FinanceService) Delete(ctx context.Context, request *model.DeleteFinanceRequest) (bool, error) {
	tx := s.db.WithContext(ctx).Begin()
	defer tx.Rollback()

	if err := s.validate.Struct(request); err != nil {
		s.log.Errorf("failed to validate request body: %v", err)
		return false, err
	}

	finance := new(entity.Finance)
	if err := tx.Where("id = ?", request.ID).First(finance).Error; err != nil {
		s.log.Errorf("failed to find finance by id: %v", err)
		return false, exception.FinanceNotFound
	}

	if err := tx.Delete(finance).Error; err != nil {
		s.log.Errorf("failed to delete finance data: %v", err)
		return false, exception.InternalServerError
	}

	if err := tx.Create(&entity.ActivityLog{
		Type:        "DELETE-FINANCE",
		Title:       "Data keuangan dihapus",
		Description: fmt.Sprintf("%s - %s", finance.Category, finance.Description),
	}).Error; err != nil {
		s.log.Errorf("failed to save activity log to database: %v", err)
		return false, exception.InternalServerError
	}

	if err := tx.Commit().Error; err != nil {
		s.log.Errorf("failed to commit transaction database: %v", err)
		return false, exception.InternalServerError
	}

	return true, nil
}

func (s *FinanceService) FindById(ctx context.Context, request *model.FindByIdFinanceRequest) (*model.FinanceResponse, error) {
	tx := s.db.WithContext(ctx).Begin()
	defer tx.Rollback()

	if err := s.validate.Struct(request); err != nil {
		s.log.Errorf("failed to validate request body: %v", err)
		return nil, err
	}

	finance := new(entity.Finance)
	if err := tx.Where("id = ?", request.ID).First(finance).Error; err != nil {
		s.log.Errorf("failed to find finance by id: %v", err)
		return nil, exception.FinanceNotFound
	}

	if err := tx.Commit().Error; err != nil {
		s.log.Errorf("failed to commit transaction database: %v", err)
		return nil, exception.InternalServerError
	}

	return converter.FinanceToResponse(finance), nil
}

func (s *FinanceService) FindAll(ctx context.Context, request *model.FindAllFinanceRequest) ([]model.FinanceResponse, int64, error) {
	tx := s.db.WithContext(ctx).Begin()
	defer tx.Rollback()

	if err := s.validate.Struct(request); err != nil {
		s.log.Errorf("failed to validate request body: %v", err)
		return nil, 0, err
	}

	var total int64
	if err := tx.Model(new(entity.Finance)).Count(&total).Error; err != nil {
		s.log.Errorf("failed to count finance data: %v", err)
		return nil, 0, exception.InternalServerError
	}

	var finances []entity.Finance
	if err := tx.Scopes(s.FilterFinance(request)).Offset((request.Page - 1) * request.Size).Limit(request.Size).Order("created_at DESC").Find(&finances).Error; err != nil {
		s.log.Errorf("failed to find all finances data: %v", err)
		return nil, 0, exception.InternalServerError
	}

	if err := tx.Commit().Error; err != nil {
		s.log.Errorf("failed to commit transaction database: %v", err)
		return nil, 0, exception.InternalServerError
	}

	responses := make([]model.FinanceResponse, total)
	for _, finance := range finances {
		responses = append(responses, *converter.FinanceToResponse(&finance))
	}

	return responses, total, nil
}

func (s *FinanceService) FilterFinance(query *model.FindAllFinanceRequest) func(tx *gorm.DB) *gorm.DB {
	return func(tx *gorm.DB) *gorm.DB {
		if searchQuery := query.SearchQuery; searchQuery != "" && len(searchQuery) > 0 {
			searchPattern := "%" + searchQuery + "%"
			tx = tx.Where(
				tx.Where("category ILIKE ?", searchPattern).
					Or("description ILIKE ?", searchPattern).
					Or("extra_note ILIKE ?", searchPattern).
					Or("type ILIKE ?"),
			)
		}

		const layout = "2006-01-02"

		if query.StartDate != "" && len(query.StartDate) > 0 {
			if parsedStart, err := time.Parse(layout, query.StartDate); err == nil {
				startOfDay := time.Date(
					parsedStart.Year(), parsedStart.Month(), parsedStart.Day(),
					0, 0, 0, 0, time.Local,
				)
				tx = tx.Where("created_at >= ?", startOfDay)
			}
		}

		if query.EndDate != "" && len(query.EndDate) > 0 {
			if parsedEnd, err := time.Parse(layout, query.EndDate); err == nil {
				endOfDay := time.Date(
					parsedEnd.Year(), parsedEnd.Month(), parsedEnd.Day(),
					23, 59, 59, 999999999, time.Local,
				)
				tx = tx.Where("created_at <= ?", endOfDay)
			}
		}

		return tx
	}
}
