package service

import (
	"context"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/ridwanmuh3/simba-server/internal/entity"
	"github.com/ridwanmuh3/simba-server/internal/exception"
	"github.com/ridwanmuh3/simba-server/internal/model"
	"github.com/ridwanmuh3/simba-server/internal/util"
)

// AccessTokenTTL and RefreshTokenTTL expose session lifetimes for cookie Max-Age.
const (
	AccessTokenTTL  = 30 * time.Minute
	RefreshTokenTTL = 7 * 24 * time.Hour
)

// dummyHash is a pre-computed bcrypt hash used to normalise the Login response
// time when a username does not exist. Without it an attacker can enumerate
// valid usernames by measuring whether the response is fast (user not found)
// or slow (bcrypt comparison). Computed once at startup to match DefaultCost.
var (
	dummyHash     []byte
	dummyHashOnce sync.Once
)

func getDummyHash() []byte {
	dummyHashOnce.Do(func() {
		h, _ := bcrypt.GenerateFromPassword([]byte("__simba_dummy_sentinel__"), bcrypt.DefaultCost)
		dummyHash = h
	})
	return dummyHash
}

func (s *UserService) Login(ctx context.Context, request *model.LoginUserRequest) (*model.Auth, error) {
	tx := s.db.WithContext(ctx).Begin()
	defer tx.Rollback()

	if err := s.validate.Struct(request); err != nil {
		s.log.Errorf("failed to validate request body: %v", err)
		return nil, err
	}

	user := new(entity.User)
	userErr := s.userRepository.FindByUsername(tx, user, request.Username)
	if userErr != nil {
		// Run bcrypt against a dummy hash so the response time matches a
		// real wrong-password attempt — prevents username enumeration.
		_ = bcrypt.CompareHashAndPassword(getDummyHash(), []byte(request.Password))
		s.log.Warnf("login attempt with unknown username")
		return nil, exception.UserInvalidCredentialsError
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(request.Password)); err != nil {
		s.log.Warnf("login attempt with invalid password for user id %d", user.ID)
		return nil, exception.UserInvalidCredentialsError
	}

	// Stateless JWT access token — not stored in DB.
	accessToken, err := util.GenerateJWT(s.jwtSecret, int(user.ID), user.Fullname, user.Role, AccessTokenTTL)
	if err != nil {
		s.log.Errorf("failed to generate access JWT: %v", err)
		return nil, exception.InternalServerError
	}

	refreshPlain := util.GenerateRandomString(64)
	now := time.Now()

	user.RefreshToken = util.HashToken(refreshPlain)
	user.RefreshExpiresAt = now.Add(RefreshTokenTTL)
	user.IsActive = true
	user.LastActive = now

	if err := s.userRepository.Save(tx, user); err != nil {
		s.log.Errorf("failed to update user refresh token: %v", err)
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
		Token:        accessToken,
		RefreshToken: refreshPlain,
	}, nil
}

// RefreshSession validates the opaque refresh token, rotates both tokens.
// One-time-use refresh: re-use of a rotated refresh token fails immediately
// (the old hash is overwritten), providing basic refresh-token reuse detection.
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
		user.RefreshToken = ""
		user.IsActive = false
		_ = s.userRepository.Save(tx, user)
		_ = tx.Commit().Error
		s.log.Warnf("refresh attempt with expired token for user id %d", user.ID)
		return nil, exception.UserUnauthorizedError
	}

	accessToken, err := util.GenerateJWT(s.jwtSecret, int(user.ID), user.Fullname, user.Role, AccessTokenTTL)
	if err != nil {
		s.log.Errorf("failed to generate access JWT on refresh: %v", err)
		return nil, exception.InternalServerError
	}

	refreshPlain := util.GenerateRandomString(64)

	user.RefreshToken = util.HashToken(refreshPlain)
	user.RefreshExpiresAt = now.Add(RefreshTokenTTL)
	user.IsActive = true
	user.LastActive = now

	if err := s.userRepository.Save(tx, user); err != nil {
		s.log.Errorf("failed to rotate refresh token: %v", err)
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
		Token:        accessToken,
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
		s.log.Errorf("failed to find user by id: %v", err)
		return false, exception.UserNotFoundError
	}

	user.RefreshToken = ""
	user.RefreshExpiresAt = time.Time{}
	user.IsActive = false
	user.LastActive = time.Now()

	if err := s.userRepository.Save(tx, user); err != nil {
		s.log.Errorf("failed to clear user refresh token: %v", err)
		return false, exception.InternalServerError
	}

	if err := tx.Commit().Error; err != nil {
		s.log.Errorf("failed to commit database transaction: %v", err)
		return false, exception.InternalServerError
	}

	return true, nil
}
