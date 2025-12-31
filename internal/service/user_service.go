package service

import (
	"context"

	"github.com/go-playground/validator/v10"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"github.com/ridwanmuh3/simba-server/internal/entity"
	"github.com/ridwanmuh3/simba-server/internal/exception"
	"github.com/ridwanmuh3/simba-server/internal/model"
	"github.com/ridwanmuh3/simba-server/internal/model/converter"
	"github.com/ridwanmuh3/simba-server/internal/repository"
)

type UserService struct {
	db             *gorm.DB
	log            *zap.SugaredLogger
	validate       *validator.Validate
	userRepository *repository.UserRepository
}

func NewUserService(db *gorm.DB, logger *zap.SugaredLogger, validate *validator.Validate, userRepository *repository.UserRepository) *UserService {
	return &UserService{
		db:             db,
		log:            logger,
		validate:       validate,
		userRepository: userRepository,
	}
}

func (s *UserService) Create(ctx context.Context, request *model.CreateUserRequest) (*model.UserResponse, error) {
	tx := s.db.WithContext(ctx).Begin()
	defer tx.Rollback()

	if err := s.validate.Struct(request); err != nil {
		s.log.Errorf("failed to validate request body: %v", err)
		return nil, err
	}

	count, err := s.userRepository.CountByUsername(tx, request.Username)
	if err != nil {
		s.log.Errorf("failed to count user by username: %v", err)
		return nil, exception.UserNotFoundError
	}

	if count > 0 {
		s.log.Errorf("user already exists")
		return nil, exception.UserAlreadyExistsError
	}

	hashPassword, err := bcrypt.GenerateFromPassword([]byte(request.Password), bcrypt.DefaultCost)
	if err != nil {
		s.log.Errorf("failed to hash password: %v", err)
		return nil, exception.InternalServerError
	}

	user := &entity.User{
		Username: request.Username,
		Fullname: request.Fullname,
		Role:     request.Role,
		Password: string(hashPassword),
	}

	if err := s.userRepository.Save(tx, user); err != nil {
		s.log.Errorf("failed to save new user: %v", err)
		return nil, exception.InternalServerError
	}

	if err := tx.Commit().Error; err != nil {
		s.log.Errorf("failed to commit database transaction: %v", err)
		return nil, exception.InternalServerError
	}

	return converter.UserToResponse(user), nil
}

func (s *UserService) Update(ctx context.Context, request *model.UpdateUserRequest) (*model.UserResponse, error) {
	tx := s.db.WithContext(ctx).Begin()
	defer tx.Rollback()

	if err := s.validate.Struct(request); err != nil {
		s.log.Errorf("failed to validate request body: %v", err)
		return nil, err
	}

	user := new(entity.User)
	if err := s.userRepository.FindById(tx, user, request.ID); err != nil {
		s.log.Errorf("failed to find user by id: %v", err)
		return nil, exception.UserNotFoundError
	}

	if request.OldPassword != "" {
		if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(request.OldPassword)); err != nil {
			s.log.Errorf("failed to compare user password: %v", err)
			return nil, exception.UserPasswordNotMatch
		}
	}

	if request.NewPassword != "" {
		hashPassword, err := bcrypt.GenerateFromPassword([]byte(request.NewPassword), bcrypt.DefaultCost)
		if err != nil {
			s.log.Errorf("failed to hash new password: %v", err)
			return nil, exception.InternalServerError
		}
		user.Password = string(hashPassword)
	}

	user.Fullname = request.Fullname

	if err := s.userRepository.Save(tx, user); err != nil {
		s.log.Errorf("failed to save updated user: %v", err)
		return nil, err
	}

	if err := tx.Commit().Error; err != nil {
		s.log.Errorf("failed to commit database transaction: %v", err)
		return nil, exception.InternalServerError
	}

	return converter.UserToResponse(user), nil
}

func (s *UserService) Delete(ctx context.Context, request *model.DeleteUserRequest) (bool, error) {
	tx := s.db.WithContext(ctx).Begin()
	defer tx.Rollback()

	if err := s.validate.Struct(request); err != nil {
		s.log.Errorf("failed to validate request body: %v", err)
		return false, err
	}

	user := new(entity.User)
	if err := s.userRepository.FindById(tx, user, request.ID); err != nil {
		s.log.Errorf("failed to find user by id: %v", err)
		return false, exception.UserNotFoundError
	}

	if err := s.userRepository.Delete(tx, user); err != nil {
		s.log.Errorf("failed to delete user by id: %v", err)
		return false, exception.InternalServerError
	}

	if err := tx.Commit().Error; err != nil {
		s.log.Errorf("failed to commit database transaction: %v", err)
		return false, exception.InternalServerError
	}

	return true, nil
}

func (s *UserService) FindById(ctx context.Context, request *model.FindByIdUserRequest) (*model.UserResponse, error) {
	tx := s.db.WithContext(ctx).Begin()
	defer tx.Rollback()

	if err := s.validate.Struct(request); err != nil {
		s.log.Errorf("failed to validate request body: %v", err)
		return nil, err
	}

	user := new(entity.User)
	if err := s.userRepository.FindById(tx, user, request.ID); err != nil {
		s.log.Errorf("failed to find user by id: %v", err)
		return nil, exception.UserNotFoundError
	}

	if err := tx.Commit().Error; err != nil {
		s.log.Errorf("failed to commit database transaction: %v", err)
		return nil, exception.InternalServerError
	}

	return converter.UserToResponse(user), nil
}

func (s *UserService) FindAll(ctx context.Context, request *model.FindAllUserRequest) ([]model.UserResponse, int64, error) {
	tx := s.db.WithContext(ctx).Begin()
	defer tx.Rollback()

	if err := s.validate.Struct(request); err != nil {
		s.log.Errorf("failed to validate request body: %v", err)
		return nil, 0, err
	}

	users, total, err := s.userRepository.FindAll(tx, request)
	if err != nil {
		s.log.Errorf("failed to find all users: %v", err)
		return nil, 0, exception.UserNotFoundError
	}

	if err := tx.Commit().Error; err != nil {
		s.log.Errorf("failed to commit database transaction: %v", err)
		return nil, 0, exception.InternalServerError
	}

	responses := make([]model.UserResponse, len(users))
	for i, user := range users {
		responses[i] = *converter.UserToResponse(&user)
	}

	return responses, total, nil
}
