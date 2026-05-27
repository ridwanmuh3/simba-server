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
)

type FinanceService struct {
	db                *gorm.DB
	log               *zap.SugaredLogger
	validate          *validator.Validate
	financeRepository FinanceRepository
}

type FinanceRepository interface {
	Save(db *gorm.DB, entity *entity.Finance) error
	Delete(db *gorm.DB, entity *entity.Finance) error
	FindByIdForUpdate(db *gorm.DB, id any, dapurID uint) (*entity.Finance, error)
	FindById(db *gorm.DB, id any, dapurID uint) (*entity.Finance, error)
	FindAll(db *gorm.DB, query *model.FindAllFinanceRequest) ([]entity.Finance, int64, error)
	FindAllUnpaginated(db *gorm.DB, dapurID uint) ([]entity.Finance, int64, error)
}

var _ FinanceRepository = (*repository.FinanceRepository)(nil)

func NewFinanceService(db *gorm.DB, logger *zap.SugaredLogger, validate *validator.Validate, financeRepository FinanceRepository) *FinanceService {
	return &FinanceService{
		db:                db,
		log:               logger,
		validate:          validate,
		financeRepository: financeRepository,
	}
}

func (s *FinanceService) Add(ctx context.Context, request *model.AddFinanceRequest) (*model.FinanceResponse, error) {
	tx := s.db.WithContext(ctx).Begin()
	defer tx.Rollback()

	if err := s.validate.Struct(request); err != nil {
		s.log.Errorf("failed to validate request body: %v", err)
		return nil, err
	}

	finance := &entity.Finance{
		Type:        request.Type,
		Category:    request.Category,
		Description: request.Description,
		Amount:      request.Amount,
		ExtraNote:   request.ExtraNote,
		ProofImage:  request.ProofImage,
		ModifiedBy:  request.ModifiedBy,
		DapurID:     request.DapurID,
	}

	if err := s.financeRepository.Save(tx, finance); err != nil {
		s.log.Errorf("failed to save finance data: %v", err)
		return nil, exception.InternalServerError
	}

	if err := tx.Create(&entity.ActivityLog{
		Type:        "ADD-FINANCE",
		Title:       "Data keuangan ditambahkan",
		Description: fmt.Sprintf("%s - %s", finance.Category, finance.Description),
		ActionBy:    request.ModifiedBy,
		DapurID:     request.DapurID,
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

	finance, err := s.financeRepository.FindByIdForUpdate(tx, request.ID, request.DapurID)
	if err != nil {
		s.log.Errorf("failed to find finance by id: %v", err)
		return nil, exception.FinanceNotFound
	}

	finance.Category = request.Category
	finance.Description = request.Description
	finance.Amount = request.Amount
	finance.ExtraNote = request.ExtraNote
	finance.ModifiedBy = request.ModifiedBy
	if request.ProofImage != "" {
		finance.ProofImage = request.ProofImage
	}

	if err := s.financeRepository.Save(tx, finance); err != nil {
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

	finance, err := s.financeRepository.FindByIdForUpdate(tx, request.ID, request.DapurID)
	if err != nil {
		s.log.Errorf("failed to find finance by id: %v", err)
		return false, exception.FinanceNotFound
	}

	if err := s.financeRepository.Delete(tx, finance); err != nil {
		s.log.Errorf("failed to delete finance data: %v", err)
		return false, exception.InternalServerError
	}

	if err := tx.Create(&entity.ActivityLog{
		Type:        "DELETE-FINANCE",
		Title:       "Data keuangan dihapus",
		Description: fmt.Sprintf("%s - %s", finance.Category, finance.Description),
		ActionBy:    finance.ModifiedBy,
		DapurID:     request.DapurID,
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
	db := s.db.WithContext(ctx)

	if err := s.validate.Struct(request); err != nil {
		s.log.Errorf("failed to validate request body: %v", err)
		return nil, err
	}

	finance, err := s.financeRepository.FindById(db, request.ID, request.DapurID)
	if err != nil {
		s.log.Errorf("failed to find finance by id: %v", err)
		return nil, exception.FinanceNotFound
	}

	return converter.FinanceToResponse(finance), nil
}

func (s *FinanceService) FindAll(
	ctx context.Context,
	request *model.FindAllFinanceRequest,
) ([]model.FinanceResponse, int64, error) {
	db := s.db.WithContext(ctx)

	if err := s.validate.Struct(request); err != nil {
		return nil, 0, err
	}

	finances, total, err := s.financeRepository.FindAll(db, request)
	if err != nil {
		s.log.Errorf("failed to find all finances: %v", err)
		return nil, 0, exception.InternalServerError
	}

	responses := make([]model.FinanceResponse, 0, len(finances))
	for _, finance := range finances {
		responses = append(responses, *converter.FinanceToResponse(&finance))
	}

	return responses, total, nil
}

func (s *FinanceService) Export(ctx context.Context, dapurID uint) ([]model.FinanceResponse, int64, error) {
	db := s.db.WithContext(ctx)

	finances, total, err := s.financeRepository.FindAllUnpaginated(db, dapurID)
	if err != nil {
		s.log.Errorf("failed to find all finances data: %v", err)
		return nil, 0, exception.InternalServerError
	}

	responses := make([]model.FinanceResponse, len(finances))
	for i, finance := range finances {
		responses[i] = *converter.FinanceToResponse(&finance)
	}

	return responses, total, nil
}
