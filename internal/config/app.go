package config

import (
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/ridwanmuh3/simba-server/internal/delivery/handler"
	"github.com/ridwanmuh3/simba-server/internal/delivery/middleware"
	"github.com/ridwanmuh3/simba-server/internal/delivery/route"
	"github.com/ridwanmuh3/simba-server/internal/repository"
	"github.com/ridwanmuh3/simba-server/internal/service"
)

type BootstrapConfig struct {
	DB       *gorm.DB
	App      *fiber.App
	Log      *zap.SugaredLogger
	Validate *validator.Validate
	Config   *viper.Viper
}

func Bootstrap(config *BootstrapConfig) {

	// global middlewares
	config.App.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowHeaders: "Cache-Control",
	}))

	userRepository := repository.NewUserRepository()

	userService := service.NewUserService(config.DB, config.Log, config.Validate, userRepository)

	userHandler := handler.NewUserHandler(config.Config, config.Log, userService)

	authMiddleware := middleware.NewAuthMiddleware(config.Log, userService)

	routeConfig := &route.RouteConfig{
		App:            config.App,
		UserHandler:    userHandler,
		AuthMiddleware: authMiddleware,
		Log:            config.Log,
	}

	routeConfig.Setup()
}
