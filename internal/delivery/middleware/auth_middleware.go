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
		logger.Debugf("token cookie: %s", request.Token)

		auth, err := userService.Verify(c.UserContext(), request)
		if err != nil {
			logger.Warnf("failed to find user by token: %v", err)
			return exception.UserUnauthorizedError
		}

		logger.Debugf("user: %v", auth)
		c.Locals("auth", auth)
		return c.Next()
	}
}

func GetAuthUser(c *fiber.Ctx) *model.Auth {
	return c.Locals("auth").(*model.Auth)
}
