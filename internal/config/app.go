package config

import (
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/contrib/fiberzap/v2"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/helmet"
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
	financeRepository := repository.NewFinanceRepository()
	dashboardRepository := repository.NewDashboardRepository()
	settingRepository := repository.NewSettingRepository()
	dapurRepository := repository.NewDapurRepository()

	// services
	userService := service.NewUserService(config.DB, config.Log, config.Validate, userRepository)
	itemService := service.NewItemService(config.DB, config.Log, config.Validate, itemRepository)
	stockService := service.NewStockService(config.DB, config.Log, config.Validate, itemRepository)
	invoiceService := service.NewInvoiceService(config.DB, config.Log, config.Validate)
	financeService := service.NewFinanceService(config.DB, config.Log, config.Validate, financeRepository)
	dashboardService := service.NewDashboardService(config.DB, config.Log, config.Validate, dashboardRepository)
	settingService := service.NewSettingService(config.DB, settingRepository, config.Log, config.Validate)
	dapurService := service.NewDapurService(config.DB, config.Log, config.Validate, dapurRepository, userRepository)

	// handler
	userHandler := handler.NewUserHandler(config.Config.GetString("APP_RESET_SECRET"), config.Log, userService)
	itemHandler := handler.NewItemHandler(config.Log, itemService)
	stockHandler := handler.NewStockHandler(config.Log, stockService)
	invoiceHandler := handler.NewInvoiceHandler(config.Log, config.Validate, invoiceService, settingService)
	financeHandler := handler.NewFinanceHandler(config.Log, financeService)
	dashboardHandler := handler.NewDashboardHandler(config.Log, dashboardService)
	settingHandler := handler.NewSettingHandler(config.Log, settingService)
	dapurHandler := handler.NewDapurHandler(config.Log, dapurService)

	authMiddleware := middleware.NewAuthMiddleware(config.Log, userService)
	dapurRequiredMiddleware := middleware.NewDapurRequiredMiddleware(config.Log)

	routeConfig := &route.RouteConfig{
		App:                     config.App,
		UserHandler:             userHandler,
		ItemHandler:             itemHandler,
		StockHandler:            stockHandler,
		InvoiceHandler:          invoiceHandler,
		FinanceHandler:          financeHandler,
		DashboardHandler:        dashboardHandler,
		SettingHandler:          settingHandler,
		DapurHandler:            dapurHandler,
		AuthMiddleware:          authMiddleware,
		DapurRequiredMiddleware: dapurRequiredMiddleware,
		Log:                     config.Log,
	}

	SetupGlobalMiddlewares(config)
	routeConfig.Setup()
}

func SetupGlobalMiddlewares(config *BootstrapConfig) {
	allowedOrigins := config.Config.GetString("APP_CORS_ALLOWED_ORIGINS")

	config.App.Use(recover.New())
	config.App.Use(fiberzap.New(fiberzap.Config{
		Logger: config.Log.Desugar(),
	}))
	config.App.Use(helmet.New())
	config.App.Use(cors.New(cors.Config{
		AllowHeaders:     "Origin, Content-Type, Accept, Authorization",
		AllowCredentials: true,
		AllowMethods:     "GET, POST, HEAD, PUT, DELETE, PATCH",
		AllowOrigins:     allowedOrigins,
	}))

}
