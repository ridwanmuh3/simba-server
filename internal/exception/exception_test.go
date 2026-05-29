package exception

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
)

func TestNewErrorHandler(t *testing.T) {
	app := fiber.New(fiber.Config{ErrorHandler: NewErrorHandler()})
	app.Get("/fiber-error", func(c *fiber.Ctx) error {
		return fiber.NewError(fiber.StatusNotFound, "missing")
	})
	app.Get("/validation-error", func(c *fiber.Ctx) error {
		validate := validator.New()
		var req struct {
			Name string `validate:"required"`
		}
		return validate.Struct(req)
	})

	t.Run("fiber error maps status and message", func(t *testing.T) {
		resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/fiber-error", nil), -1)
		if err != nil {
			t.Fatalf("app.Test error: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusNotFound {
			t.Fatalf("status = %d, want 404", resp.StatusCode)
		}

		var body map[string]any
		if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if body["error"] != "missing" {
			t.Fatalf("error = %v, want missing", body["error"])
		}
	})

	t.Run("validation error maps bad request", func(t *testing.T) {
		resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/validation-error", nil), -1)
		if err != nil {
			t.Fatalf("app.Test error: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", resp.StatusCode)
		}
	})
}
