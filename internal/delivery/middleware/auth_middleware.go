package middleware

import (
	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"

	"github.com/ridwanmuh3/simba-server/internal/exception"
	"github.com/ridwanmuh3/simba-server/internal/model"
	"github.com/ridwanmuh3/simba-server/internal/service"
)

func NewAuthMiddleware(logger *zap.SugaredLogger, userService *service.UserService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		request := &model.VerifyUserRequest{
			Token: c.Cookies("token", "NOT_FOUND"),
		}

		auth, err := userService.Verify(c.UserContext(), request)
		if err != nil {
			logger.Warnf("failed to find user by token: %v", err)
			return exception.UserUnauthorizedError
		}

		logger.Debugf("user: %v", auth.Fullname)
		c.Locals("auth", auth)
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

// NewDapurRequiredMiddleware rejects requests where user has not selected a dapur.
// Must run after NewAuthMiddleware.
func NewDapurRequiredMiddleware(logger *zap.SugaredLogger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		auth := GetAuthUser(c)
		if auth == nil || auth.CurrentDapurID == nil {
			logger.Warnf("dapur not selected for user request to %s", c.Path())
			return exception.DapurNotSelectedError
		}
		return c.Next()
	}
}
