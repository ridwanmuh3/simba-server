package route

import (
	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"

	"github.com/ridwanmuh3/simba-server/internal/delivery/handler"
	"github.com/ridwanmuh3/simba-server/internal/delivery/middleware"
	"github.com/ridwanmuh3/simba-server/internal/model"
)

type RouteConfig struct {
	App            *fiber.App
	UserHandler    *handler.UserHandler
	ItemHandler    *handler.ItemHandler
	AuthMiddleware fiber.Handler
	Log            *zap.SugaredLogger
}

func (c *RouteConfig) Setup() {
	c.SetupPublicRoute()
	c.SetupAuthRoute()
	c.SetupUserRoute()
	c.SetupItemRoute()
}

func (c *RouteConfig) SetupPublicRoute() {
	c.App.Get("/", func(c *fiber.Ctx) error {
		return c.JSON(model.Response[any]{
			Status:  fiber.StatusOK,
			Message: "Welcome to  SIMBA API!",
		})
	})
}

func (c *RouteConfig) SetupAuthRoute() {
	authRoute := c.App.Group("/api/auth")

	authRoute.Post("/login", c.UserHandler.Login)
	authRoute.Delete("/logout", c.AuthMiddleware, c.UserHandler.Logout)
	authRoute.Get("/_current", c.AuthMiddleware, c.UserHandler.Current)
}

func (c *RouteConfig) SetupUserRoute() {
	userRoute := c.App.Group("/api/users")

	// setup scope middleware
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

	// setup scope middleware
	itemRoute.Use(c.AuthMiddleware, middleware.NewRbacMiddleware(c.Log, "Admin", "Super Admin"))
	itemRoute.Post("/", c.ItemHandler.Add)
	itemRoute.Post("/import", c.ItemHandler.ImportItems)
	itemRoute.Put("/:id", c.ItemHandler.Update)
	itemRoute.Delete("/:id", c.ItemHandler.Delete)
	itemRoute.Get("/export", c.ItemHandler.ExportItems)
	itemRoute.Get("/:id", c.ItemHandler.FindById)
	itemRoute.Get("/", c.ItemHandler.FindAll)
}
