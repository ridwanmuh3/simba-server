package service

import (
	"context"
	"crypto/sha3"
	"encoding/base64"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/ridwanmuh3/simba-server/internal/entity"
	"github.com/ridwanmuh3/simba-server/internal/exception"
	"github.com/ridwanmuh3/simba-server/internal/model"
	"github.com/ridwanmuh3/simba-server/internal/model/converter"
)

func (s *UserService) Login(ctx context.Context, request *model.LoginUserRequest) (*model.UserResponse, error) {
	tx := s.db.WithContext(ctx).Begin()
	defer tx.Rollback()

	if err := s.validate.Struct(request); err != nil {
		s.log.Errorf("failed to validate request body: %v", err)
		return nil, err
	}

	user := new(entity.User)
	if err := s.userRepository.FindByUsername(tx, user, request.Username); err != nil {
		s.log.Errorf("failed to find user by username: %v", err)
		return nil, exception.UserUnauthorizedError
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(request.Password)); err != nil {
		s.log.Errorf("failed to compare password: %v", err)
		return nil, exception.UserUnauthorizedError
	}

	token, err := s.GenerateToken()
	if err != nil {
		s.log.Errorf("failed to generate token: %v", err)
		return nil, exception.InternalServerError
	}

	user.Token = token
	user.IsActive = true
	user.LastActive = time.Now()

	if err := s.userRepository.Save(tx, user); err != nil {
		s.log.Errorf("failed to update user token: %v", err)
		return nil, exception.InternalServerError
	}

	if err := tx.Commit().Error; err != nil {
		s.log.Errorf("failed to commit database transaction: %v", err)
		return nil, exception.InternalServerError
	}

	return converter.UserToTokenResponse(user), nil
}

func (s *UserService) Logout(ctx context.Context, request *model.LogoutUserRequest) (bool, error) {
	tx := s.db.WithContext(ctx).Begin()
	defer tx.Rollback()

	if err := s.validate.Struct(request); err != nil {
		s.log.Errorf("failed to validate request body: %v", err)
		return false, err
	}

	user := new(entity.User)
	if err := s.userRepository.FindById(tx, user, request.ID); err != nil {
		s.log.Errorf("failed to find user by username: %v", err)
		return false, exception.UserNotFoundError
	}

	user.Token = ""
	user.IsActive = false
	user.LastActive = time.Now()

	if err := s.userRepository.Save(tx, user); err != nil {
		s.log.Errorf("failed to delete user token: %v", err)
		return false, exception.InternalServerError
	}

	if err := tx.Commit().Error; err != nil {
		s.log.Errorf("failed to commit database transaction: %v", err)
		return false, exception.InternalServerError
	}

	return true, nil
}

func (s *UserService) Verify(ctx context.Context, request *model.VerifyUserRequest) (*model.Auth, error) {
	tx := s.db.WithContext(ctx).Begin()
	defer tx.Rollback()

	if err := s.validate.Struct(request); err != nil {
		s.log.Errorf("failed to validate request body: %v", err)
		return nil, err
	}

	user := new(entity.User)
	if err := s.userRepository.FindByToken(tx, user, request.Token); err != nil {
		s.log.Errorf("failed to find user by token: %v", err)
		return nil, exception.UserNotFoundError
	}

	if err := tx.Commit().Error; err != nil {
		s.log.Errorf("failed to commit database transaction: %v", err)
		return nil, exception.InternalServerError
	}

	return &model.Auth{
		ID:       int(user.ID),
		Fullname: user.Fullname,
		Role:     user.Role,
	}, nil
}

func (s *UserService) GenerateToken() (string, error) {
	rawToken := uuid.New()

	bcryptHashToken, err := bcrypt.GenerateFromPassword(rawToken[:], bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}

	hashToken := sha3.Sum256(bcryptHashToken)

	return base64.RawURLEncoding.EncodeToString(hashToken[:]), nil
}
