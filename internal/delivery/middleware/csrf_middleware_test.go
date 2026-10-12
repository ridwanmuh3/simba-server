package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

func newCSRFApp() *fiber.App {
	app := fiber.New()
	app.Use(NewCSRFMiddleware(zap.NewNop().Sugar()))
	app.Get("/safe", func(c *fiber.Ctx) error {
		return c.SendString(c.Locals(CSRFContextKey).(string))
	})
	app.Post("/write", func(c *fiber.Ctx) error {
		return c.SendStatus(http.StatusOK)
	})
	app.Put("/write", func(c *fiber.Ctx) error {
		return c.SendStatus(http.StatusOK)
	})
	app.Patch("/write", func(c *fiber.Ctx) error {
		return c.SendStatus(http.StatusOK)
	})
	app.Delete("/write", func(c *fiber.Ctx) error {
		return c.SendStatus(http.StatusOK)
	})
	return app
}

func csrfCookie(resp *http.Response) *http.Cookie {
	for _, c := range resp.Cookies() {
		if c.Name == CSRFCookieName {
			return c
		}
	}
	return nil
}

func TestCSRFMiddlewareGeneratesToken(t *testing.T) {
	app := newCSRFApp()

	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/safe", nil), -1)
	if err != nil {
		t.Fatalf("app.Test error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	cookie := csrfCookie(resp)
	if cookie == nil {
		t.Fatal("no csrf cookie set")
	}
	if cookie.Value == "" {
		t.Fatal("csrf cookie value is empty")
	}
	if cookie.Path != "/" {
		t.Fatalf("cookie path = %q, want /", cookie.Path)
	}
	if cookie.HttpOnly {
		t.Fatal("csrf cookie must be readable by JS (double-submit pattern)")
	}
	if cookie.SameSite != http.SameSiteLaxMode {
		t.Fatalf("sameSite = %v, want Lax", cookie.SameSite)
	}
}

func TestCSRFMiddlewareReusesExistingToken(t *testing.T) {
	app := newCSRFApp()

	req := httptest.NewRequest(http.MethodGet, "/safe", nil)
	req.AddCookie(&http.Cookie{Name: CSRFCookieName, Value: "existing-token"})
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("app.Test error: %v", err)
	}
	defer resp.Body.Close()

	cookie := csrfCookie(resp)
	if cookie == nil || cookie.Value != "existing-token" {
		t.Fatalf("token rotated unexpectedly: %+v", cookie)
	}
}

func TestCSRFMiddlewareProtectedMethods(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		cookie     string
		header     string
		wantStatus int
	}{
		{name: "post without any token", method: http.MethodPost, wantStatus: http.StatusForbidden},
		{name: "post cookie only", method: http.MethodPost, cookie: "abc", wantStatus: http.StatusForbidden},
		{name: "post header only", method: http.MethodPost, header: "abc", wantStatus: http.StatusForbidden},
		{name: "post mismatched", method: http.MethodPost, cookie: "abc", header: "def", wantStatus: http.StatusForbidden},
		{name: "post empty header", method: http.MethodPost, cookie: "abc", header: " ", wantStatus: http.StatusForbidden},
		{name: "post matching", method: http.MethodPost, cookie: "abc", header: "abc", wantStatus: http.StatusOK},
		{name: "put matching", method: http.MethodPut, cookie: "abc", header: "abc", wantStatus: http.StatusOK},
		{name: "patch mismatched", method: http.MethodPatch, cookie: "abc", header: "x", wantStatus: http.StatusForbidden},
		{name: "delete matching", method: http.MethodDelete, cookie: "abc", header: "abc", wantStatus: http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := newCSRFApp()
			req := httptest.NewRequest(tt.method, "/write", nil)
			if tt.cookie != "" {
				req.AddCookie(&http.Cookie{Name: CSRFCookieName, Value: tt.cookie})
			}
			if tt.header != "" {
				req.Header.Set(CSRFHeaderName, tt.header)
			}

			resp, err := app.Test(req, -1)
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

func TestCSRFMiddlewareSafeMethodsBypassValidation(t *testing.T) {
	app := newCSRFApp()

	// A GET with a forged header but no cookie must not be rejected.
	req := httptest.NewRequest(http.MethodGet, "/safe", nil)
	req.Header.Set(CSRFHeaderName, "forged")
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("app.Test error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
}
