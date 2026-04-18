package config

import (
	"github.com/bytedance/sonic"
	"github.com/gofiber/fiber/v2"
	"github.com/spf13/viper"

	"github.com/ridwanmuh3/simba-server/internal/exception"
)

func NewFiber(config *viper.Viper) *fiber.App {
	return fiber.New(fiber.Config{
		AppName:                 config.GetString("APP_NAME"),
		JSONEncoder:             sonic.Marshal,
		JSONDecoder:             sonic.Unmarshal,
		ErrorHandler:            exception.NewErrorHandler(),
		Prefork:                 config.GetBool("APP_PREFORK"),
		BodyLimit:               15 * 1024 * 1024, // 15MB max request body
		EnableTrustedProxyCheck: true,
		TrustedProxies: []string{
			"127.0.0.1",
			"::1",
			"10.0.0.0/8",
			"172.16.0.0/12",
			"192.168.0.0/16",
		},
	})
}
