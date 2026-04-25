package service

import (
	"context"

	"github.com/go-playground/validator/v10"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/ridwanmuh3/simba-server/internal/entity"
	"github.com/ridwanmuh3/simba-server/internal/exception"
	"github.com/ridwanmuh3/simba-server/internal/model"
	"github.com/ridwanmuh3/simba-server/internal/repository"
)

type DapurService struct {
	db              *gorm.DB
	log             *zap.SugaredLogger
	validate        *validator.Validate
	dapurRepository DapurRepository
	userRepository  UserRepository
}

type DapurRepository interface {
	Save(db *gorm.DB, entity *entity.Dapur) error
	FindById(db *gorm.DB, entity *entity.Dapur, id any) error
	Delete(db *gorm.DB, entity *entity.Dapur) error
	FindAll(db *gorm.DB) ([]entity.Dapur, error)
	CountByName(db *gorm.DB, name string) (int64, error)
}

var _ DapurRepository = (*repository.DapurRepository)(nil)

func NewDapurService(
	db *gorm.DB,
	log *zap.SugaredLogger,
	validate *validator.Validate,
	dapurRepository DapurRepository,
	userRepository UserRepository,
) *DapurService {
	return &DapurService{
		db:              db,
		log:             log,
		validate:        validate,
		dapurRepository: dapurRepository,
		userRepository:  userRepository,
	}
}

func (s *DapurService) FindAll(ctx context.Context) ([]model.DapurResponse, error) {
	db := s.db.WithContext(ctx)
	dapurs, err := s.dapurRepository.FindAll(db)
	if err != nil {
		s.log.Errorf("failed to list dapurs: %v", err)
		return nil, exception.InternalServerError
	}

	responses := make([]model.DapurResponse, len(dapurs))
	for i, d := range dapurs {
		responses[i] = model.DapurResponse{
			ID:          d.ID,
			Name:        d.Name,
			Description: d.Description,
			IsActive:    d.IsActive,
			CreatedAt:   d.CreatedAt,
		}
	}
	return responses, nil
}

func (s *DapurService) Create(ctx context.Context, request *model.CreateDapurRequest) (*model.DapurResponse, error) {
	tx := s.db.WithContext(ctx).Begin()
	defer tx.Rollback()

	if err := s.validate.Struct(request); err != nil {
		return nil, err
	}

	count, err := s.dapurRepository.CountByName(tx, request.Name)
	if err != nil {
		return nil, exception.InternalServerError
	}
	if count > 0 {
		return nil, exception.DapurAlreadyExists
	}

	dapur := &entity.Dapur{
		Name:        request.Name,
		Description: request.Description,
		IsActive:    true,
	}
	if err := s.dapurRepository.Save(tx, dapur); err != nil {
		s.log.Errorf("failed to save dapur: %v", err)
		return nil, exception.InternalServerError
	}

	if err := tx.Commit().Error; err != nil {
		return nil, exception.InternalServerError
	}

	return &model.DapurResponse{
		ID:          dapur.ID,
		Name:        dapur.Name,
		Description: dapur.Description,
		IsActive:    dapur.IsActive,
		CreatedAt:   dapur.CreatedAt,
	}, nil
}

func (s *DapurService) Update(ctx context.Context, request *model.UpdateDapurRequest) (*model.DapurResponse, error) {
	tx := s.db.WithContext(ctx).Begin()
	defer tx.Rollback()

	if err := s.validate.Struct(request); err != nil {
		return nil, err
	}

	dapur := new(entity.Dapur)
	if err := s.dapurRepository.FindById(tx, dapur, request.ID); err != nil {
		return nil, exception.DapurNotFoundError
	}

	if request.Name != "" {
		dapur.Name = request.Name
	}
	if request.Description != "" {
		dapur.Description = request.Description
	}
	if request.IsActive != nil {
		dapur.IsActive = *request.IsActive
	}

	if err := s.dapurRepository.Save(tx, dapur); err != nil {
		s.log.Errorf("failed to update dapur: %v", err)
		return nil, exception.InternalServerError
	}

	if err := tx.Commit().Error; err != nil {
		return nil, exception.InternalServerError
	}

	return &model.DapurResponse{
		ID:          dapur.ID,
		Name:        dapur.Name,
		Description: dapur.Description,
		IsActive:    dapur.IsActive,
		CreatedAt:   dapur.CreatedAt,
	}, nil
}

func (s *DapurService) Delete(ctx context.Context, request *model.DeleteDapurRequest) error {
	tx := s.db.WithContext(ctx).Begin()
	defer tx.Rollback()

	if err := s.validate.Struct(request); err != nil {
		return err
	}

	dapur := new(entity.Dapur)
	if err := s.dapurRepository.FindById(tx, dapur, request.ID); err != nil {
		return exception.DapurNotFoundError
	}

	if err := s.dapurRepository.Delete(tx, dapur); err != nil {
		s.log.Errorf("failed to delete dapur: %v", err)
		return exception.InternalServerError
	}

	return tx.Commit().Error
}

// SelectDapur validates the dapur exists and is active, then updates the user's current_dapur_id.
func (s *DapurService) SelectDapur(ctx context.Context, request *model.SelectDapurRequest) error {
	tx := s.db.WithContext(ctx).Begin()
	defer tx.Rollback()

	if err := s.validate.Struct(request); err != nil {
		return err
	}

	dapur := new(entity.Dapur)
	if err := s.dapurRepository.FindById(tx, dapur, request.DapurID); err != nil {
		return exception.DapurNotFoundError
	}
	if !dapur.IsActive {
		return exception.DapurNotActiveError
	}

	user := new(entity.User)
	if err := s.userRepository.FindById(tx, user, request.UserID); err != nil {
		return exception.UserNotFoundError
	}

	dapurID := dapur.ID
	user.CurrentDapurID = &dapurID
	if err := s.userRepository.Save(tx, user); err != nil {
		s.log.Errorf("failed to update user current_dapur_id: %v", err)
		return exception.InternalServerError
	}

	return tx.Commit().Error
}
