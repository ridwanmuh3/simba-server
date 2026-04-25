package exception

import "github.com/gofiber/fiber/v2"

var (
	DapurNotSelectedError = fiber.NewError(fiber.StatusForbidden, "dapur not selected")
	DapurNotFoundError    = fiber.NewError(fiber.StatusNotFound, "dapur not found")
	DapurNotActiveError   = fiber.NewError(fiber.StatusForbidden, "dapur is not active")
	DapurAlreadyExists    = fiber.NewError(fiber.StatusConflict, "dapur with that name already exists")
)
