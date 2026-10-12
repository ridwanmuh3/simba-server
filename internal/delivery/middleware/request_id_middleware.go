package middleware

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

const RequestIDHeader = "X-Request-ID"

func RequestID() fiber.Handler {
	return func(c *fiber.Ctx) error {
		id := c.Get(RequestIDHeader)
		if id == "" {
			id = uuid.NewString()
		}
		c.Locals(RequestIDHeader, id)
		c.Set(RequestIDHeader, id)
		return c.Next()
	}
}

func GetRequestID(c *fiber.Ctx) string {
	id := c.Locals(RequestIDHeader)
	if id == nil {
		return ""
	}
	return id.(string)
}
