package config

import (
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/contrib/fiberzap/v2"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/recover"
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
	// repositories
	userRepository := repository.NewUserRepository()
	itemRepository := repository.NewItemRepository()

	// services
	userService := service.NewUserService(config.DB, config.Log, config.Validate, userRepository)
	itemService := service.NewItemService(config.DB, config.Log, config.Validate, itemRepository)
	// storageService := service.NewStorageService(config.Log, config.Validate)

	// handler
	userHandler := handler.NewUserHandler(config.Config, config.Log, userService)
	itemHandler := handler.NewItemHandler(config.Config, config.Log, itemService)
	// storageHandler := handler.NewStorageHandler(config.Config, config.Log, storageService)

	authMiddleware := middleware.NewAuthMiddleware(config.Log, userService)

	routeConfig := &route.RouteConfig{
		App:         config.App,
		UserHandler: userHandler,
		ItemHandler: itemHandler,
		// StorageHandler: storageHandler,
		AuthMiddleware: authMiddleware,
		Log:            config.Log,
	}

	SetupGlobalMiddlewares(config)
	routeConfig.Setup()
}

func SetupGlobalMiddlewares(config *BootstrapConfig) {
	config.App.Use(cors.New(cors.Config{
		AllowHeaders:     "Origin, Content-Type, Accept, Authorization",
		AllowCredentials: true,
		AllowMethods:     "GET, POST, HEAD, PUT, DELETE, PATCH",
		AllowOrigins:     "http://localhost:9091",
	}))

	config.App.Use(recover.New())

	config.App.Use(fiberzap.New(fiberzap.Config{
		Logger: config.Log.Desugar(),
	}))
}
