package service

import (
	"context"

	"github.com/go-playground/validator/v10"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

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
		return nil, exception.InternalServerError
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
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		First(user, "id = ?", request.ID).Error; err != nil {
		s.log.Errorf("failed to find user by id: %v", err)
		return nil, exception.UserNotFoundError
	}

	if request.Password != "" {
		hashPassword, err := bcrypt.GenerateFromPassword([]byte(request.Password), bcrypt.DefaultCost)
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

	if request.AuthID == request.ID {
		s.log.Errorf("you are not allowed to self delete account")
		return false, exception.UserCannotSelfDeleteError
	}

	if user.Role == "Super Admin" {
		var count int64
		tx.Model(user).Where("role = ?", user.Role).Count(&count)
		if count <= 1 {
			s.log.Errorf("failed to delete last super admin user. system must have one super admin user")
			return false, exception.UserDeleteSingleSuperAdminError
		}
	}

	// if user.IsActive {
	// 	s.log.Errorf("failed to delete currently active user")
	// 	return false, exception.UserDeleteActiveUserError
	// }

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
	db := s.db.WithContext(ctx)

	if err := s.validate.Struct(request); err != nil {
		s.log.Errorf("failed to validate request body: %v", err)
		return nil, err
	}

	user := new(entity.User)
	if err := s.userRepository.FindById(db, user, request.ID); err != nil {
		s.log.Errorf("failed to find user by id: %v", err)
		return nil, exception.UserNotFoundError
	}

	return converter.UserToResponse(user), nil
}

func (s *UserService) FindAll(ctx context.Context, request *model.FindAllUserRequest) ([]model.UserResponse, int64, error) {
	db := s.db.WithContext(ctx)

	if err := s.validate.Struct(request); err != nil {
		s.log.Errorf("failed to validate request body: %v", err)
		return nil, 0, err
	}

	users, total, err := s.userRepository.FindAll(db, request)
	if err != nil {
		s.log.Errorf("failed to find all users: %v", err)
		return nil, 0, exception.UserNotFoundError
	}

	responses := make([]model.UserResponse, len(users))
	for i, user := range users {
		responses[i] = *converter.UserToResponse(&user)
	}

	return responses, total, nil
}

func (s *UserService) GetUsersStats(ctx context.Context) (*model.UsersStatsResponse, error) {
	db := s.db.WithContext(ctx)

	stats, err := s.userRepository.GetUsersStats(db)
	if err != nil {
		s.log.Errorf("failed to get users statistics: %v", err)
		return nil, exception.InternalServerError
	}

	return &model.UsersStatsResponse{
		UsersTotal:           stats.Total,
		UsersSuperAdminTotal: stats.SuperAdmin,
		UsersAdminTotal:      stats.Admin,
		UsersActiveTotal:     stats.Active,
	}, nil
}
