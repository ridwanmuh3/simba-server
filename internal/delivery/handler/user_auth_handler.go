package handler

import (
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/ridwanmuh3/simba-server/internal/delivery/middleware"
	"github.com/ridwanmuh3/simba-server/internal/exception"
	"github.com/ridwanmuh3/simba-server/internal/model"
	"github.com/ridwanmuh3/simba-server/internal/service"
)

// setAuthCookies writes the access + refresh cookies with matching server-side
// lifetimes. Access cookie is available to the whole API; refresh cookie is
// path-scoped to the refresh endpoint so it is not sent on every request and
// cannot be grabbed from CSRF-vulnerable endpoints outside /api/auth.
func setAuthCookies(c *fiber.Ctx, accessToken, refreshToken string) {
	isSecureRequest := strings.EqualFold(c.Protocol(), "https")
	sameSite := "Lax"
	if isSecureRequest {
		sameSite = "Strict"
	}
	now := time.Now()

	c.Cookie(&fiber.Cookie{
		Name:     "token",
		Value:    accessToken,
		Path:     "/",
		Expires:  now.Add(service.AccessTokenTTL),
		HTTPOnly: true,
		Secure:   isSecureRequest,
		SameSite: sameSite,
	})

	c.Cookie(&fiber.Cookie{
		Name:     "refresh_token",
		Value:    refreshToken,
		Path:     "/api/auth",
		Expires:  now.Add(service.RefreshTokenTTL),
		HTTPOnly: true,
		Secure:   isSecureRequest,
		SameSite: sameSite,
	})

	// Non-HttpOnly cookie so the frontend can read the expiry timestamp
	// and schedule a proactive refresh before the JWT expires.
	c.Cookie(&fiber.Cookie{
		Name:     "token_exp",
		Value:    strconv.FormatInt(now.Add(service.AccessTokenTTL).Unix(), 10),
		Path:     "/",
		Expires:  now.Add(service.AccessTokenTTL),
		HTTPOnly: false,
		Secure:   isSecureRequest,
		SameSite: sameSite,
	})
}

func clearAuthCookies(c *fiber.Ctx) {
	isSecureRequest := strings.EqualFold(c.Protocol(), "https")
	sameSite := "Lax"
	if isSecureRequest {
		sameSite = "Strict"
	}

	c.Cookie(&fiber.Cookie{
		Name:     "token",
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		HTTPOnly: true,
		Secure:   isSecureRequest,
		SameSite: sameSite,
	})

	c.Cookie(&fiber.Cookie{
		Name:     "refresh_token",
		Value:    "",
		Path:     "/api/auth",
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		HTTPOnly: true,
		Secure:   isSecureRequest,
		SameSite: sameSite,
	})

	c.Cookie(&fiber.Cookie{
		Name:     "token_exp",
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		HTTPOnly: false,
		Secure:   isSecureRequest,
		SameSite: sameSite,
	})
}

func (h *UserHandler) Login(c *fiber.Ctx) error {
	request := new(model.LoginUserRequest)
	if err := c.BodyParser(request); err != nil {
		h.log.Warnf("failed to parse request body: %v", err)
		return err
	}

	response, err := h.userService.Login(c.Context(), request)
	if err != nil {
		h.log.Warnf("failed to login user: %v", err)
		return err
	}

	setAuthCookies(c, response.Token, response.RefreshToken)

	return c.JSON(model.Response[*model.Auth]{
		Status:  fiber.StatusOK,
		Message: "user login success",
		Data:    response,
	})
}

func (h *UserHandler) Refresh(c *fiber.Ctx) error {
	refresh := c.Cookies("refresh_token", "")
	if refresh == "" {
		return fiber.NewError(fiber.StatusUnauthorized, "missing refresh token")
	}

	response, err := h.userService.RefreshSession(c.Context(), &model.RefreshSessionRequest{
		RefreshToken: refresh,
	})
	if err != nil {
		clearAuthCookies(c)
		h.log.Warnf("failed to refresh session: %v", err)
		return err
	}

	setAuthCookies(c, response.Token, response.RefreshToken)

	return c.JSON(model.Response[*model.Auth]{
		Status:  fiber.StatusOK,
		Message: "session refresh success",
		Data:    response,
	})
}

func (h *UserHandler) Logout(c *fiber.Ctx) error {
	auth := middleware.GetAuthUser(c)
	request := &model.LogoutUserRequest{
		ID: auth.ID,
	}

	response, err := h.userService.Logout(c.Context(), request)
	if err != nil {
		h.log.Warnf("failed to logout user: %v", err)
		return err
	}

	clearAuthCookies(c)

	return c.JSON(model.Response[bool]{
		Status:  fiber.StatusOK,
		Message: "user logout success",
		Data:    response,
	})
}

func (h *UserHandler) Current(c *fiber.Ctx) error {
	auth := middleware.GetAuthUser(c)
	request := &model.FindByIdUserRequest{
		ID: auth.ID,
	}

	response, err := h.userService.FindById(c.Context(), request)
	if err != nil {
		h.log.Warnf("failed to login user: %v", err)
		return err
	}

	authData := model.Auth{
		ID:       response.ID,
		Fullname: response.Fullname,
		Role:     response.Role,
		Avatar:   response.Avatar,
	}

	return c.JSON(model.Response[model.Auth]{
		Status:  fiber.StatusOK,
		Message: "get current auth user success",
		Data:    authData,
	})
}

func (h *UserHandler) GetProfile(c *fiber.Ctx) error {
	auth := middleware.GetAuthUser(c)

	response, err := h.userService.FindById(c.Context(), &model.FindByIdUserRequest{ID: auth.ID})
	if err != nil {
		h.log.Warnf("failed to get profile: %v", err)
		return err
	}

	return c.JSON(model.Response[*model.UserResponse]{
		Status:  fiber.StatusOK,
		Message: "get profile success",
		Data:    response,
	})
}

func (h *UserHandler) UpdateProfile(c *fiber.Ctx) error {
	auth := middleware.GetAuthUser(c)

	request := new(model.UpdateProfileRequest)
	if err := c.BodyParser(request); err != nil {
		h.log.Warnf("failed to parse profile update body: %v", err)
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	request.AuthID = auth.ID

	response, err := h.userService.UpdateProfile(c.Context(), request)
	if err != nil {
		h.log.Warnf("failed to update profile: %v", err)
		return err
	}

	return c.JSON(model.Response[*model.UserResponse]{
		Status:  fiber.StatusOK,
		Message: "profile updated",
		Data:    response,
	})
}

func (h *UserHandler) UpdateAvatar(c *fiber.Ctx) error {
	auth := middleware.GetAuthUser(c)

	currentUser, err := h.userService.FindById(c.Context(), &model.FindByIdUserRequest{ID: auth.ID})
	if err != nil {
		h.log.Warnf("failed to get current user for avatar update: %v", err)
		return err
	}

	avatarFile, err := c.FormFile("avatar")
	if err != nil {
		h.log.Warnf("failed to get avatar file: %v", err)
		return exception.InvalidUploadedFileError
	}

	ext := strings.ToLower(filepath.Ext(avatarFile.Filename))
	if !slices.Contains([]string{".png", ".jpg", ".jpeg"}, ext) {
		h.log.Warnf("invalid avatar file format: %s", ext)
		return exception.InvalidFileFormatError
	}

	const maxSize = int64(5 * 1024 * 1024)
	if avatarFile.Size > maxSize {
		h.log.Warnf("avatar file too large: %d bytes", avatarFile.Size)
		return exception.ExceedMaximumFileSizeError
	}

	uploadDir := filepath.Join("uploads", "avatars")
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		h.log.Warnf("failed to create avatars directory: %v", err)
		return exception.InternalServerError
	}

	fileName := uuid.New().String() + ext
	filePath := filepath.Join(uploadDir, fileName)

	if err := c.SaveFile(avatarFile, filePath); err != nil {
		h.log.Warnf("failed to save avatar file: %v", err)
		return exception.InternalServerError
	}

	newAvatarPath := "/uploads/avatars/" + fileName

	response, err := h.userService.UpdateAvatar(c.Context(), auth.ID, newAvatarPath)
	if err != nil {
		h.log.Warnf("failed to update avatar in db: %v", err)
		if removeErr := os.Remove(filePath); removeErr != nil {
			h.log.Warnf("failed to cleanup orphaned avatar: %v", removeErr)
		}
		return err
	}

	if currentUser.Avatar != "" {
		oldFileName := filepath.Base(currentUser.Avatar)
		oldFilePath := filepath.Join("uploads", "avatars", oldFileName)
		if removeErr := os.Remove(oldFilePath); removeErr != nil && !os.IsNotExist(removeErr) {
			h.log.Warnf("failed to remove old avatar: %v", removeErr)
		}
	}

	return c.JSON(model.Response[*model.UserResponse]{
		Status:  fiber.StatusOK,
		Message: "avatar updated",
		Data:    response,
	})
}
