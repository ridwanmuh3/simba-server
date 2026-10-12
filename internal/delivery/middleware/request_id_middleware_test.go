package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

func TestIsValidRequestID(t *testing.T) {
	tests := []struct {
		name string
		id   string
		want bool
	}{
		{name: "uuid", id: uuid.NewString(), want: true},
		{name: "traceparent style", id: "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01", want: true},
		{name: "dashes underscores dots", id: "trace-1.a_b", want: true},
		{name: "empty", id: "", want: false},
		{name: "CRLF injection", id: "abc\r\nX-Evil: 1", want: false},
		{name: "LF only", id: "abc\nX-Evil: 1", want: false},
		{name: "CR only", id: "abc\rX-Evil", want: false},
		{name: "space", id: "abc def", want: false},
		{name: "tab", id: "abc\tdef", want: false},
		{name: "colon (header syntax)", id: "a: b", want: false},
		{name: "quote", id: `a"b`, want: false},
		{name: "NUL", id: "abc\x00def", want: false},
		{name: "non-ascii", id: "abc-ñ", want: false},
		{name: "exactly at length limit", id: strings.Repeat("a", maxRequestIDLen), want: true},
		{name: "over length limit", id: strings.Repeat("a", maxRequestIDLen+1), want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isValidRequestID(tt.id); got != tt.want {
				t.Errorf("isValidRequestID(%q) = %v, want %v", tt.id, got, tt.want)
			}
		})
	}
}

// A request id that fails validation must be replaced by a generated UUID, so
// hostile input can never reach the response header or the logs.
func TestRequestIDRejectsInvalid(t *testing.T) {
	app := fiber.New()
	app.Use(RequestID())
	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString(GetRequestID(c))
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set(RequestIDHeader, "trace-1 with spaces")
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("app.Test error: %v", err)
	}
	defer resp.Body.Close()

	got := resp.Header.Get(RequestIDHeader)
	if got == "trace-1 with spaces" {
		t.Fatal("middleware propagated an unvalidated request id")
	}
	if _, err := uuid.Parse(got); err != nil {
		t.Fatalf("rejected id was not replaced with a uuid, got %q", got)
	}
}

func TestRequestIDGeneratesWhenMissing(t *testing.T) {
	app := fiber.New()
	app.Use(RequestID())
	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString(GetRequestID(c))
	})

	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/", nil), -1)
	if err != nil {
		t.Fatalf("app.Test error: %v", err)
	}
	defer resp.Body.Close()

	id := resp.Header.Get(RequestIDHeader)
	if id == "" {
		t.Fatal("response missing X-Request-ID header")
	}
	if _, err := uuid.Parse(id); err != nil {
		t.Fatalf("generated id %q is not a uuid: %v", id, err)
	}
}

func TestRequestIDPropagatesIncoming(t *testing.T) {
	app := fiber.New()
	app.Use(RequestID())
	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString(GetRequestID(c))
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set(RequestIDHeader, "trace-123")
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("app.Test error: %v", err)
	}
	defer resp.Body.Close()

	if got := resp.Header.Get(RequestIDHeader); got != "trace-123" {
		t.Fatalf("header = %q, want trace-123", got)
	}
}

// A client-supplied request id must never be reflected verbatim when it
// contains characters that could break response framing (header injection).
// NOTE: fasthttp already drops such headers, so this exercises the middleware
// as a regression guard; TestIsValidRequestID covers our own validation.
func TestRequestIDSanitizesUntrustedInput(t *testing.T) {
	app := fiber.New()
	app.Use(RequestID())
	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString(GetRequestID(c))
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set(RequestIDHeader, "abc\r\nX-Evil: 1")
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("app.Test error: %v", err)
	}
	defer resp.Body.Close()

	if resp.Header.Get("X-Evil") != "" {
		t.Fatal("request id injected an extra response header")
	}

	got := resp.Header.Get(RequestIDHeader)
	if strings.ContainsAny(got, "\r\n") {
		t.Fatalf("request id still contains CR/LF: %q", got)
	}
}

func TestGetRequestIDWithoutMiddleware(t *testing.T) {
	app := fiber.New()
	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString(GetRequestID(c))
	})

	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/", nil), -1)
	if err != nil {
		t.Fatalf("app.Test error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
}
