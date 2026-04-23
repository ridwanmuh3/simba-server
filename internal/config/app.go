package config

import (
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/contrib/fiberzap/v2"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/helmet"
	"github.com/gofiber/fiber/v2/middleware/limiter"
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

	jwtSecret := []byte(config.Config.GetString("JWT_SECRET"))
	// RFC 7518 §3.2: HMAC-SHA256 key MUST be ≥ 256 bits (32 bytes).
	// An empty or short secret allows trivial token forgery.
	if len(jwtSecret) < 32 {
		panic("JWT_SECRET must be at least 32 bytes (256 bits) — set a strong random value in your environment")
	}

	// services
	userService := service.NewUserService(config.DB, config.Log, config.Validate, userRepository, jwtSecret)
	itemService := service.NewItemService(config.DB, config.Log, config.Validate, itemRepository)
	stockService := service.NewStockService(config.DB, config.Log, config.Validate, itemRepository)
	invoiceService := service.NewInvoiceService(config.DB, config.Log, config.Validate)
	financeService := service.NewFinanceService(config.DB, config.Log, config.Validate, financeRepository)
	dashboardService := service.NewDashboardService(config.DB, config.Log, config.Validate, dashboardRepository)
	settingService := service.NewSettingService(config.DB, settingRepository, config.Log, config.Validate)

	// handler
	userHandler := handler.NewUserHandler(config.Config, config.Log, userService)
	itemHandler := handler.NewItemHandler(config.Config, config.Log, config.Validate, itemService, stockService, invoiceService, settingService)
	financeHandler := handler.NewFinanceHandler(config.Config, config.Log, financeService)
	dashboardHandler := handler.NewDashboardHandler(config.Config, config.Log, dashboardService)
	settingHandler := handler.NewSettingHandler(config.Config, config.Log, config.Validate, settingService)

	authMiddleware := middleware.NewAuthMiddleware(config.Log, jwtSecret)

	routeConfig := &route.RouteConfig{
		App:              config.App,
		UserHandler:      userHandler,
		ItemHandler:      itemHandler,
		FinanceHandler:   financeHandler,
		DashboardHandler: dashboardHandler,
		SettingHandler:   settingHandler,
		AuthMiddleware:   authMiddleware,
		Log:              config.Log,
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

	// Global rate limit: 200 req/min per IP.
	config.App.Use(limiter.New(limiter.Config{
		Max:        200,
		Expiration: 1 * time.Minute,
		KeyGenerator: func(c *fiber.Ctx) string {
			return c.IP()
		},
		LimitReached: func(c *fiber.Ctx) error {
			return fiber.NewError(fiber.StatusTooManyRequests, "too many requests")
		},
	}))

	// Strict rate limit on auth endpoints: 10 req/min per IP (brute-force guard).
	config.App.Use("/api/auth", limiter.New(limiter.Config{
		Max:        10,
		Expiration: 1 * time.Minute,
		KeyGenerator: func(c *fiber.Ctx) string {
			return c.IP()
		},
		LimitReached: func(c *fiber.Ctx) error {
			return fiber.NewError(fiber.StatusTooManyRequests, "too many requests")
		},
	}))
}
