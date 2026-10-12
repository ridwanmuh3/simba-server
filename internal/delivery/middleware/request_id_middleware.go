package middleware

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

const RequestIDHeader = "X-Request-ID"

// maxRequestIDLen bounds an inbound request id so a hostile client cannot
// push unbounded text into logs and response headers.
const maxRequestIDLen = 64

// isValidRequestID reports whether an inbound request id is safe to propagate
// verbatim. Values that fail this check are replaced with a generated UUID.
func isValidRequestID(id string) bool {
	if id == "" || len(id) > maxRequestIDLen {
		return false
	}
	for i := 0; i < len(id); i++ {
		ch := id[i]
		switch {
		case ch >= 'a' && ch <= 'z',
			ch >= 'A' && ch <= 'Z',
			ch >= '0' && ch <= '9',
			ch == '-', ch == '_', ch == '.':
		default:
			return false
		}
	}
	return true
}

func RequestID() fiber.Handler {
	return func(c *fiber.Ctx) error {
		id := c.Get(RequestIDHeader)
		if !isValidRequestID(id) {
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
