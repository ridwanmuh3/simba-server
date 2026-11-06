package exception

import "github.com/gofiber/fiber/v2"

var (
	ItemNotFoundError      = fiber.NewError(fiber.StatusNotFound, "item not found")
	InsufficientStockError = fiber.NewError(fiber.StatusBadRequest, "insufficient stock")
	// Domain validation errors returned by services. Services must return
	// these (or other exception.* values), never fiber.NewError directly;
	// the Fiber error handler in this package maps them to HTTP.
	InvalidAmountError    = fiber.NewError(fiber.StatusBadRequest, "amount must be greater than 0")
	InvalidStockTypeError = fiber.NewError(fiber.StatusBadRequest, "invalid stock type")
	InvalidRequestError   = fiber.NewError(fiber.StatusBadRequest, "invalid request")
)
