package middleware

import (
	"slices"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"

	"github.com/ridwanmuh3/simba-server/internal/exception"
)

func NewRbacMiddleware(logger *zap.SugaredLogger, roles ...string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		auth := GetAuthUser(c)
		if auth == nil {
			logger.Warnf("failed to get user auth")
			return exception.UserUnauthorizedError
		}

		if !slices.Contains(roles, auth.Role) {
			logger.Warnf("invalid user role")
			return exception.UserForbiddenError
		}

		return c.Next()
	}
}
