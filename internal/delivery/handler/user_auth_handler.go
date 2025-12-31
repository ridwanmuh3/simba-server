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

	expirationTime := h.config.GetDuration("APP_COOKIE_EXPIRATION_TIME")

	c.Cookie(&fiber.Cookie{
		Name:    "token",
		Value:   response.Token,
		Expires: time.Now().Add(expirationTime),
		// HTTPOnly: true,
		// Secure: true,
	})

	return c.JSON(model.Response[bool]{
		Status:  fiber.StatusOK,
		Message: "user login success",
		Data:    true,
	})
}

func (h *UserHandler) Logout(c *fiber.Ctx) error {
	request := new(model.LogoutUserRequest)
	if err := c.BodyParser(request); err != nil {
		h.log.Warnf("failed to parse request body: %v", err)
		return err
	}

	response, err := h.userService.Logout(c.Context(), request)
	if err != nil {
		h.log.Warnf("failed to logout user: %v", err)
		return err
	}

	c.Cookie(&fiber.Cookie{
		Name:    "token",
		Value:   "",
		Expires: time.Now().Add(-1),
		// HTTPOnly: true,
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

	request := new(model.FindByIdUserRequest)
	if err := c.BodyParser(request); err != nil {
		h.log.Warnf("failed to parse request body: %v", err)
		return err
	}

	request.ID = auth.ID

	response, err := h.userService.FindById(c.Context(), request)
	if err != nil {
		h.log.Warnf("failed to login user: %v", err)
		return err
	}

	return c.JSON(model.Response[*model.UserResponse]{
		Status:  fiber.StatusOK,
		Message: "get current auth user success",
		Data:    response,
	})
}
