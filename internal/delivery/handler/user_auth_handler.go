package handler

import (
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/spf13/viper"

	"github.com/ridwanmuh3/simba-server/internal/delivery/middleware"
	"github.com/ridwanmuh3/simba-server/internal/model"
	"github.com/ridwanmuh3/simba-server/internal/service"
)

// setAuthCookies writes the access + refresh cookies with matching server-side
// lifetimes. Access cookie is available to the whole API; refresh cookie is
// path-scoped to the refresh endpoint so it is not sent on every request and
// cannot be grabbed from CSRF-vulnerable endpoints outside /api/auth.
func setAuthCookies(c *fiber.Ctx, config *viper.Viper, accessToken, refreshToken string) {
	isProduction := strings.HasPrefix(config.GetString("APP_CORS_ALLOWED_ORIGINS"), "https")
	sameSite := "Lax"
	if isProduction {
		sameSite = "Strict"
	}
	now := time.Now()

	c.Cookie(&fiber.Cookie{
		Name:     "token",
		Value:    accessToken,
		Path:     "/",
		Expires:  now.Add(service.AccessTokenTTL),
		HTTPOnly: true,
		Secure:   isProduction,
		SameSite: sameSite,
	})

	c.Cookie(&fiber.Cookie{
		Name:     "refresh_token",
		Value:    refreshToken,
		Path:     "/api/auth",
		Expires:  now.Add(service.RefreshTokenTTL),
		HTTPOnly: true,
		Secure:   isProduction,
		SameSite: sameSite,
	})
}

func clearAuthCookies(c *fiber.Ctx, config *viper.Viper) {
	isProduction := strings.HasPrefix(config.GetString("APP_CORS_ALLOWED_ORIGINS"), "https")
	sameSite := "Lax"
	if isProduction {
		sameSite = "Strict"
	}

	c.Cookie(&fiber.Cookie{
		Name:     "token",
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		HTTPOnly: true,
		Secure:   isProduction,
		SameSite: sameSite,
	})

	c.Cookie(&fiber.Cookie{
		Name:     "refresh_token",
		Value:    "",
		Path:     "/api/auth",
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		HTTPOnly: true,
		Secure:   isProduction,
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

	setAuthCookies(c, h.config, response.Token, response.RefreshToken)

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
		clearAuthCookies(c, h.config)
		h.log.Warnf("failed to refresh session: %v", err)
		return err
	}

	setAuthCookies(c, h.config, response.Token, response.RefreshToken)

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

	isProduction := strings.HasPrefix(h.config.GetString("APP_CORS_ALLOWED_ORIGINS"), "https")
	sameSite := "Lax"
	if isProduction {
		sameSite = "Strict"
	}

	c.Cookie(&fiber.Cookie{
		Name:     "token",
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		HTTPOnly: true,
		Secure:   isProduction,
		SameSite: sameSite,
	})

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
	}

	return c.JSON(model.Response[model.Auth]{
		Status:  fiber.StatusOK,
		Message: "get current auth user success",
		Data:    authData,
	})
}
