package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"

	"github.com/ridwanmuh3/simba-server/internal/model"
)

func TestNewRbacMiddleware(t *testing.T) {
	tests := []struct {
		name       string
		auth       *model.Auth
		roles      []string
		wantStatus int
	}{
		{
			name:       "allowed role",
			auth:       &model.Auth{ID: 1, Role: "Admin"},
			roles:      []string{"Admin", "Super Admin"},
			wantStatus: http.StatusOK,
		},
		{
			name:       "missing auth",
			auth:       nil,
			roles:      []string{"Admin"},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "wrong role",
			auth:       &model.Auth{ID: 1, Role: "Viewer"},
			roles:      []string{"Admin"},
			wantStatus: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New()
			app.Get("/secure", func(c *fiber.Ctx) error {
				if tt.auth != nil {
					c.Locals("auth", tt.auth)
				}
				return c.Next()
			}, NewRbacMiddleware(zap.NewNop().Sugar(), tt.roles...), func(c *fiber.Ctx) error {
				return c.SendStatus(http.StatusOK)
			})

			resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/secure", nil), -1)
			if err != nil {
				t.Fatalf("app.Test error: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tt.wantStatus {
				t.Fatalf("status = %d, want %d", resp.StatusCode, tt.wantStatus)
			}
		})
	}
}
