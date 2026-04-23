package route

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"

	"github.com/ridwanmuh3/simba-server/internal/delivery/handler"
	"github.com/ridwanmuh3/simba-server/internal/delivery/middleware"
	"github.com/ridwanmuh3/simba-server/internal/model"
)

type RouteConfig struct {
	App              *fiber.App
	UserHandler      *handler.UserHandler
	ItemHandler      *handler.ItemHandler
	FinanceHandler   *handler.FinanceHandler
	DashboardHandler *handler.DashboardHandler
	SettingHandler   *handler.SettingHandler
	AuthMiddleware   fiber.Handler
	Log              *zap.SugaredLogger
}

func (c *RouteConfig) Setup() {
	c.SetupPublicRoute()
	c.SetupUploadRoute()
	c.SetupAuthRoute()
	c.SetupUserRoute()
	c.SetupItemRoute()
	c.SetupFinanceRoute()
	c.SetupDashboardRoute()
	c.SetupSettingRoute()
}

func (c *RouteConfig) SetupPublicRoute() {
	c.App.Get("/", func(c *fiber.Ctx) error {
		return c.JSON(model.Response[any]{
			Status:  fiber.StatusOK,
			Message: "Welcome to SIMBA API!",
		})
	})

	c.App.Get("/health", func(c *fiber.Ctx) error {
		return c.SendString("API OK!")
	})
}

func (c *RouteConfig) SetupAuthRoute() {
	authRoute := c.App.Group("/api/auth")

	authRoute.Post("/login", c.UserHandler.Login)
	authRoute.Delete("/logout", c.AuthMiddleware, c.UserHandler.Logout)
	authRoute.Get("/_current", c.AuthMiddleware, c.UserHandler.Current)
	authRoute.Post("/refresh", c.UserHandler.Refresh)
	authRoute.Post("/reset-password", c.UserHandler.ResetPassword)
	authRoute.Post("/register", c.UserHandler.Register)
	authRoute.Get("/profile", c.AuthMiddleware, c.UserHandler.GetProfile)
	authRoute.Put("/profile", c.AuthMiddleware, c.UserHandler.UpdateProfile)
	authRoute.Post("/profile/avatar", c.AuthMiddleware, c.UserHandler.UpdateAvatar)
}

func (c *RouteConfig) SetupUserRoute() {
	userRoute := c.App.Group("/api/users")

	userRoute.Use(c.AuthMiddleware, middleware.NewRbacMiddleware(c.Log, "Super Admin"))

	userRoute.Post("/", c.UserHandler.Create)
	userRoute.Put("/:id", c.UserHandler.Update)
	userRoute.Delete("/:id", c.UserHandler.Delete)
	userRoute.Get("/stats", c.UserHandler.GetUsersStats)
	userRoute.Get("/:id", c.UserHandler.FindById)
	userRoute.Get("/", c.UserHandler.FindAll)

}

func (c *RouteConfig) SetupItemRoute() {
	itemRoute := c.App.Group("/api/items")

	itemRoute.Use(c.AuthMiddleware, middleware.NewRbacMiddleware(c.Log, "Admin", "Super Admin"))

	itemRoute.Post("/", c.ItemHandler.Add)
	itemRoute.Post("/import", c.ItemHandler.ImportItems)
	itemRoute.Post("/:id/stocks", c.ItemHandler.UpdateStock)
	itemRoute.Patch("/:id/stocks/:stock_id", c.ItemHandler.EditStock)
	itemRoute.Put("/:id", c.ItemHandler.Update)
	itemRoute.Delete("/:id/stocks/:stock_id", c.ItemHandler.DeleteStock)
	itemRoute.Delete("/:id", c.ItemHandler.Delete)
	itemRoute.Post("/invoice", c.ItemHandler.GetInvoiceItems)
	itemRoute.Get("/invoices", c.ItemHandler.GetInvoiceHistory)
	itemRoute.Delete("/invoices/:id", c.ItemHandler.DeleteInvoice)
	itemRoute.Get("/invoices/:id/pdf", c.ItemHandler.DownloadInvoicePDF)
	itemRoute.Get("/export", c.ItemHandler.ExportItems)
	itemRoute.Get("/stocks", c.ItemHandler.FindAllStocks)
	itemRoute.Get("/stocks/summary", c.ItemHandler.GetStocksFinanceSummary)
	itemRoute.Get("/stocks/opname", c.ItemHandler.GetItemStocksSummary)
	itemRoute.Get("/:id", c.ItemHandler.FindById)
	itemRoute.Get("/", c.ItemHandler.FindAll)
}

func (c *RouteConfig) SetupFinanceRoute() {
	financeRoute := c.App.Group("/api/finances")

	financeRoute.Use(c.AuthMiddleware, middleware.NewRbacMiddleware(c.Log, "Admin", "Super Admin"))

	financeRoute.Post("/", c.FinanceHandler.Add)
	// financeRoute.Post("/import", c.FinanceHandler.ImportItems)
	financeRoute.Put("/:id", c.FinanceHandler.Update)
	financeRoute.Delete("/:id", c.FinanceHandler.Delete)
	financeRoute.Get("/export", c.FinanceHandler.Export)
	financeRoute.Get("/:id", c.FinanceHandler.FindById)
	financeRoute.Get("/", c.FinanceHandler.FindAll)
}

func (c *RouteConfig) SetupDashboardRoute() {
	dashboardRoute := c.App.Group("/api/dashboard")
	dashboardRoute.Use(c.AuthMiddleware, middleware.NewRbacMiddleware(c.Log, "Admin", "Super Admin"))
	dashboardRoute.Get("/", c.DashboardHandler.GetDashboardStats)
}

func (c *RouteConfig) SetupUploadRoute() {
	c.App.Use("/uploads", c.AuthMiddleware, middleware.NewRbacMiddleware(c.Log, "Admin", "Super Admin"))
	c.App.Static("/uploads", "./uploads", fiber.Static{
		Browse:        false,
		CacheDuration: 24 * time.Hour,
	})
}

func (c *RouteConfig) SetupSettingRoute() {
	settingRoute := c.App.Group("/api/settings")

	settingRoute.Use(c.AuthMiddleware, middleware.NewRbacMiddleware(c.Log, "Admin", "Super Admin"))

	settingRoute.Get("/company", c.SettingHandler.GetCompanyProfile)
	settingRoute.Put("/company", c.SettingHandler.UpdateCompanyProfile)
	settingRoute.Get("/next-document-number", c.SettingHandler.GetNextDocumentNumbers)
}
