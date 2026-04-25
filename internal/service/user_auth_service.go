package service

import (
	"context"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/ridwanmuh3/simba-server/internal/entity"
	"github.com/ridwanmuh3/simba-server/internal/exception"
	"github.com/ridwanmuh3/simba-server/internal/model"
	"github.com/ridwanmuh3/simba-server/internal/util"
)

// AccessTokenTTL and RefreshTokenTTL expose the default session lifetimes so
// the handler layer can set matching cookie Max-Age / Expires values.
const (
	AccessTokenTTL  = 30 * time.Minute
	RefreshTokenTTL = 7 * 24 * time.Hour
)

func (s *UserService) Login(ctx context.Context, request *model.LoginUserRequest) (*model.Auth, error) {
	tx := s.db.WithContext(ctx).Begin()
	defer tx.Rollback()

	if err := s.validate.Struct(request); err != nil {
		s.log.Errorf("failed to validate request body: %v", err)
		return nil, err
	}

	user := new(entity.User)
	if err := s.userRepository.FindByUsername(tx, user, request.Username); err != nil {
		s.log.Warnf("login attempt with invalid username")
		return nil, exception.UserInvalidCredentialsError
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(request.Password)); err != nil {
		s.log.Warnf("login attempt with invalid password")
		return nil, exception.UserInvalidCredentialsError
	}

	accessPlain := util.GenerateRandomString(64)
	refreshPlain := util.GenerateRandomString(64)
	now := time.Now()

	user.Token = util.HashToken(accessPlain)
	user.TokenExpiresAt = now.Add(AccessTokenTTL)
	user.RefreshToken = util.HashToken(refreshPlain)
	user.RefreshExpiresAt = now.Add(RefreshTokenTTL)
	user.IsActive = true
	user.LastActive = now

	if err := s.userRepository.Save(tx, user); err != nil {
		s.log.Errorf("failed to update user token: %v", err)
		return nil, exception.InternalServerError
	}

	if err := tx.Commit().Error; err != nil {
		s.log.Errorf("failed to commit database transaction: %v", err)
		return nil, exception.InternalServerError
	}

	return &model.Auth{
		ID:           int(user.ID),
		Fullname:     user.Fullname,
		Role:         user.Role,
		Token:        accessPlain,
		RefreshToken: refreshPlain,
	}, nil
}

// RefreshSession validates the presented refresh token against its stored
// hash + expiry and rotates BOTH tokens (one-time-use refresh). If the old
// refresh token is ever presented again after rotation, the DB lookup fails,
// forcing re-login — a basic reuse-detection boundary.
func (s *UserService) RefreshSession(ctx context.Context, request *model.RefreshSessionRequest) (*model.Auth, error) {
	tx := s.db.WithContext(ctx).Begin()
	defer tx.Rollback()

	if err := s.validate.Struct(request); err != nil {
		s.log.Errorf("failed to validate refresh request: %v", err)
		return nil, exception.UserUnauthorizedError
	}

	user := new(entity.User)
	hashed := util.HashToken(request.RefreshToken)
	if err := s.userRepository.FindByRefreshToken(tx, user, hashed); err != nil {
		s.log.Warnf("refresh attempt with invalid token")
		return nil, exception.UserUnauthorizedError
	}

	now := time.Now()
	if user.RefreshExpiresAt.IsZero() || now.After(user.RefreshExpiresAt) {
		// Clear the stale token to stop further use.
		user.RefreshToken = ""
		user.Token = ""
		user.IsActive = false
		_ = s.userRepository.Save(tx, user)
		_ = tx.Commit().Error
		s.log.Warnf("refresh attempt with expired token")
		return nil, exception.UserUnauthorizedError
	}

	accessPlain := util.GenerateRandomString(64)
	refreshPlain := util.GenerateRandomString(64)

	user.Token = util.HashToken(accessPlain)
	user.TokenExpiresAt = now.Add(AccessTokenTTL)
	user.RefreshToken = util.HashToken(refreshPlain)
	user.RefreshExpiresAt = now.Add(RefreshTokenTTL)
	user.IsActive = true
	user.LastActive = now

	if err := s.userRepository.Save(tx, user); err != nil {
		s.log.Errorf("failed to rotate tokens: %v", err)
		return nil, exception.InternalServerError
	}

	if err := tx.Commit().Error; err != nil {
		s.log.Errorf("failed to commit refresh transaction: %v", err)
		return nil, exception.InternalServerError
	}

	return &model.Auth{
		ID:           int(user.ID),
		Fullname:     user.Fullname,
		Role:         user.Role,
		Token:        accessPlain,
		RefreshToken: refreshPlain,
	}, nil
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
	user.TokenExpiresAt = time.Time{}
	user.RefreshToken = ""
	user.RefreshExpiresAt = time.Time{}
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
	db := s.db.WithContext(ctx)

	if err := s.validate.Struct(request); err != nil {
		s.log.Errorf("failed to validate request body: %v", err)
		return nil, err
	}

	user := new(entity.User)
	hashed := util.HashToken(request.Token)
	if err := s.userRepository.FindByToken(db, user, hashed); err != nil {
		s.log.Warnf("verify attempt with invalid token")
		return nil, exception.UserUnauthorizedError
	}

	if user.TokenExpiresAt.IsZero() || time.Now().After(user.TokenExpiresAt) {
		s.log.Warnf("verify attempt with expired token")
		return nil, exception.UserUnauthorizedError
	}

	return &model.Auth{
		ID:             int(user.ID),
		Fullname:       user.Fullname,
		Role:           user.Role,
		CurrentDapurID: user.CurrentDapurID,
	}, nil
}
