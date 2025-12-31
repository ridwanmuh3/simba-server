package route

import (
	"bufio"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/valyala/fasthttp"
	"go.uber.org/zap"

	"github.com/ridwanmuh3/simba-server/internal/delivery/handler"
	"github.com/ridwanmuh3/simba-server/internal/delivery/middleware"
	"github.com/ridwanmuh3/simba-server/internal/model"
)

type RouteConfig struct {
	App            *fiber.App
	UserHandler    *handler.UserHandler
	AuthMiddleware fiber.Handler
	Log            *zap.SugaredLogger
}

func (c *RouteConfig) Setup() {
	c.SetupPublicRoute()
	c.SetupAuthRoute()
	c.SetupUserRoute()
}

func (c *RouteConfig) SetupPublicRoute() {
	c.App.Get("/", func(c *fiber.Ctx) error {
		return c.JSON(model.Response[any]{
			Status:  fiber.StatusOK,
			Message: "welcome to simba api",
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
	userRoute.Get("/:id", c.UserHandler.FindById)
	userRoute.Get("/", c.UserHandler.FindAll)
}

func (c *RouteConfig) SetupSSE() {
	var sseQueue []*model.EventPayload[any]

	// global route
	c.App.Get("/api/notifications", func(c *fiber.Ctx) error {
		c.Set("Content-Type", "text/event-stream")
		c.Set("Cache-Control", "no-cache")
		c.Set("Connection", "keep-alive")
		c.Set("Transfer-Encoding", "chunked")

		c.Status(fiber.StatusOK).Context().SetBodyStreamWriter(fasthttp.StreamWriter(func(w *bufio.Writer) {
			fmt.Println("WRITER")
			var i int
			for {
				i++

				var msg string

				// if there are messages that have been sent to the `/publish` endpoint
				// then use these first, otherwise just send the current time
				if len(sseQueue) > 0 {
					msg = fmt.Sprintf("%d - message recieved: %s", i, sseQueue[0])
					// remove the message from the buffer
					sseQueue = sseQueue[1:]
				} else {
					msg = fmt.Sprintf("%d - the time is %v", i, time.Now())
				}

				fmt.Fprintf(w, "data: Message: %s\n\n", msg)
				fmt.Println(msg)

				err := w.Flush()
				if err != nil {
					// Refreshing page in web browser will establish a new
					// SSE connection, but only (the last) one is alive, so
					// dead connections must be closed here.
					fmt.Printf("Error while flushing: %v. Closing http connection.\n", err)

					break
				}
				time.Sleep(2 * time.Second)
			}
		}))

		return nil
	})

	c.App.Post("/api/notifications/publish", func(c *fiber.Ctx) error {
		payload := new(model.EventPayload[any])

		if err := c.BodyParser(payload); err != nil {
			return c.Status(fiber.StatusBadRequest).SendString(err.Error())
		}

		sseQueue = append(sseQueue, payload)

		return c.SendString("Message added to queue\n")
	})
}
