package middleware

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"

	"github.com/ridwanmuh3/simba-server/internal/model"
)

type fakeAuthService struct {
	auth *model.Auth
	err  error
}

func (s fakeAuthService) Verify(ctx context.Context, request *model.VerifyUserRequest) (*model.Auth, error) {
	return s.auth, s.err
}

func TestNewAuthMiddleware(t *testing.T) {
	t.Run("stores auth on success", func(t *testing.T) {
		app := fiber.New()
		app.Get("/me", NewAuthMiddleware(zap.NewNop().Sugar(), fakeAuthService{
			auth: &model.Auth{ID: 7, Fullname: "Auth User", Role: "Admin"},
		}), func(c *fiber.Ctx) error {
			auth := GetAuthUser(c)
			return c.JSON(auth)
		})

		req := httptest.NewRequest(http.MethodGet, "/me", nil)
		req.AddCookie(&http.Cookie{Name: "token", Value: "valid-token"})

		resp, err := app.Test(req, -1)
		if err != nil {
			t.Fatalf("app.Test error: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("status = %d, want 200", resp.StatusCode)
		}

		var auth model.Auth
		if err := json.NewDecoder(resp.Body).Decode(&auth); err != nil {
			t.Fatalf("decode auth: %v", err)
		}
		if auth.ID != 7 || auth.Role != "Admin" {
			t.Fatalf("auth = %+v, want id=7 role=Admin", auth)
		}
	})

	t.Run("rejects invalid token", func(t *testing.T) {
		app := fiber.New()
		app.Get("/me", NewAuthMiddleware(zap.NewNop().Sugar(), fakeAuthService{
			err: errors.New("invalid token"),
		}), func(c *fiber.Ctx) error {
			return c.SendStatus(http.StatusOK)
		})

		resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/me", nil), -1)
		if err != nil {
			t.Fatalf("app.Test error: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("status = %d, want 401", resp.StatusCode)
		}
	})
}
