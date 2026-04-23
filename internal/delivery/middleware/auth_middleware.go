package middleware

import (
	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"

	"github.com/ridwanmuh3/simba-server/internal/exception"
	"github.com/ridwanmuh3/simba-server/internal/model"
	"github.com/ridwanmuh3/simba-server/internal/util"
)

// NewAuthMiddleware validates the JWT access token from the HttpOnly cookie.
// No database roundtrip — claims are verified cryptographically.
func NewAuthMiddleware(logger *zap.SugaredLogger, jwtSecret []byte) fiber.Handler {
	return func(c *fiber.Ctx) error {
		token := c.Cookies("token", "")
		if token == "" {
			return exception.UserUnauthorizedError
		}

		// Reject oversized tokens before any parsing work — prevents an attacker
		// from forcing repeated HMAC work with artificially large cookie values.
		if len(token) > util.MaxTokenBytes {
			logger.Warnf("rejected oversized token cookie (%d bytes)", len(token))
			return exception.UserUnauthorizedError
		}

		claims, err := util.ParseJWT(jwtSecret, token)
		if err != nil {
			logger.Warnf("JWT validation failed: %v", err)
			return exception.UserUnauthorizedError
		}

		logger.Debugf("authenticated user id %d", claims.UserID)
		c.Locals("auth", &model.Auth{
			ID:       claims.UserID,
			Fullname: claims.Fullname,
			Role:     claims.Role,
		})
		return c.Next()
	}
}

func GetAuthUser(c *fiber.Ctx) *model.Auth {
	auth := c.Locals("auth")
	if auth == nil {
		return nil
	}
	return auth.(*model.Auth)
}
