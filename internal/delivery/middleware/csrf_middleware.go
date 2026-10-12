package middleware

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

const (
	CSRFContextKey = "csrf_token"
	CSRFHeaderName = "X-Csrf-Token"
	CSRFCookieName = "csrf_"
)

const csrfTokenBytes = 32

func NewCSRFMiddleware(logger *zap.SugaredLogger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		token := c.Cookies(CSRFCookieName)

		if isCSRFProtectedMethod(c.Method()) {
			headerToken := c.Get(CSRFHeaderName)
			if token == "" || headerToken == "" || !sameToken(token, headerToken) {
				logger.Warnf("csrf validation failed for %s %s", c.Method(), c.Path())
				return fiber.NewError(fiber.StatusForbidden, "invalid csrf token")
			}
		}

		if token == "" {
			generated, err := generateCSRFToken()
			if err != nil {
				logger.Errorf("failed to generate csrf token: %v", err)
				return fiber.NewError(fiber.StatusInternalServerError, "csrf token unavailable")
			}
			token = generated
		}

		setCSRFCookie(c, token)
		c.Locals(CSRFContextKey, token)
		return c.Next()
	}
}

func isCSRFProtectedMethod(method string) bool {
	switch method {
	case fiber.MethodPost, fiber.MethodPut, fiber.MethodPatch, fiber.MethodDelete:
		return true
	default:
		return false
	}
}

func sameToken(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

func generateCSRFToken() (string, error) {
	raw := make([]byte, csrfTokenBytes)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}

func setCSRFCookie(c *fiber.Ctx, token string) {
	secure := strings.EqualFold(c.Protocol(), "https")
	c.Cookie(&fiber.Cookie{
		Name:     CSRFCookieName,
		Value:    token,
		Path:     "/",
		Expires:  time.Now().Add(time.Hour),
		HTTPOnly: false,
		Secure:   secure,
		SameSite: "Lax",
	})
}
