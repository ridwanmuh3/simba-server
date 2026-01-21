package handler

import (
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/ridwanmuh3/simba-server/internal/delivery/middleware"
	"github.com/ridwanmuh3/simba-server/internal/model"
)

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

	expirationTimeEnv := h.config.GetInt("APP_COOKIE_EXPIRATION_TIME")
	if expirationTimeEnv == 0 {
		expirationTimeEnv = 30
	}

	var expirationTime time.Time
	if request.RememberMe {
		expirationTime = time.Now().Add(time.Duration(24) * time.Hour)
	} else {
		expirationTime = time.Now().Add(time.Duration(expirationTimeEnv) * time.Minute)
	}

	c.Cookie(&fiber.Cookie{
		Name:     "token",
		Value:    response.Token,
		Path:     "/",
		Expires:  expirationTime,
		HTTPOnly: true,
		// Secure: true,
	})

	return c.JSON(model.Response[bool]{
		Status:  fiber.StatusOK,
		Message: "user login success",
		Data:    true,
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

	c.Cookie(&fiber.Cookie{
		Name:     "token",
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		HTTPOnly: true,
		// Secure: true,
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
