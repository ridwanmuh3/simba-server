package exception

import "github.com/gofiber/fiber/v2"

var (
	ItemNotFoundError = fiber.NewError(fiber.StatusNotFound, "item not found")
)
