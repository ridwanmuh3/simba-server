package exception

import "github.com/gofiber/fiber/v2"

var (
	InvoiceNotFoundError = fiber.NewError(fiber.StatusNotFound, "invoice tidak ditemukan")
)

// DuplicateDocumentError is a domain error for invoice/PO numbers that
// collide with an existing document. The message is user-facing.
func DuplicateDocumentError(msg string) error {
	return fiber.NewError(fiber.StatusConflict, msg)
}
