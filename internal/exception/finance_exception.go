package exception

import "github.com/gofiber/fiber/v2"

var (
	FinanceNotFound = fiber.NewError(fiber.StatusNotFound, "finance not found")
)
