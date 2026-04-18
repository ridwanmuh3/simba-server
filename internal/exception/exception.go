package exception

import (
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"

	"github.com/ridwanmuh3/simba-server/internal/model"
)

var (
	InvalidUploadedFileError   = fiber.NewError(fiber.StatusBadRequest, "invalid uploaded file")
	InvalidFileFormatError     = fiber.NewError(fiber.StatusBadRequest, "invalid file format")
	InvalidCsvFormatError      = fiber.NewError(fiber.StatusUnprocessableEntity, "only accept csv format")
	ExceedMaximumFileSizeError = fiber.NewError(fiber.StatusUnprocessableEntity, "system only accept uploaded file size below 10MB")
	ForbiddenFilepathError     = fiber.NewError(fiber.StatusForbidden, "accessing path not allowed")
	FileNotFoundError          = fiber.NewError(fiber.StatusNotFound, "file not found")
	InternalServerError        = fiber.NewError(fiber.StatusInternalServerError, "internal server error")
)

func NewErrorHandler() fiber.ErrorHandler {
	return func(ctx *fiber.Ctx, err error) error {
		var errors any = fiber.ErrInternalServerError.Message

		code := fiber.StatusInternalServerError
		if e, ok := err.(*fiber.Error); ok {
			code = e.Code
			errors = e.Error()
		}

		if validationErrors, ok := err.(validator.ValidationErrors); ok {
			var msgs []string
			for _, validationError := range validationErrors {
				msgs = append(msgs, validationError.Error())
			}
			errors = msgs
			code = fiber.StatusBadRequest
		}

		return ctx.Status(code).JSON(model.Response[any]{
			Status: code,
			Error:  errors,
		})
	}
}
