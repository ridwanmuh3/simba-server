package exception

import "github.com/gofiber/fiber/v2"

var (
	UserUnauthorizedError = fiber.NewError(fiber.StatusUnauthorized, "user unauthorized")
	UserForbiddenError    = fiber.NewError(fiber.StatusForbidden, "user forbidden")
)
