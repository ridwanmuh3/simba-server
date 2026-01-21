package exception

import "github.com/gofiber/fiber/v2"

var (
	ItemAlreadyExistsError    = fiber.NewError(fiber.StatusConflict, "item already exists")
	ItemNotFoundError         = fiber.NewError(fiber.StatusNotFound, "item not found")
	ItemCsvTooMuchColumnError = fiber.NewError(fiber.StatusBadRequest, "only accept item csv column: 'Nama', 'Kategori', 'Jumlah', 'Satuan', dan 'Harga Satuan'")
)
