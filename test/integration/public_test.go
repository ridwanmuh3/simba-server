package integration_test

import (
	"io"
	"net/http"
	"testing"
)

func TestPublicRoutes(t *testing.T) {
	t.Run("root returns welcome response", func(t *testing.T) {
		resp := doJSON(http.MethodGet, "/", nil, nil)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("status = %d, want 200", resp.StatusCode)
		}
		r := parseResp(resp)
		if r.Status != http.StatusOK {
			t.Fatalf("response status = %d, want 200", r.Status)
		}
	})

	t.Run("health returns plain ok", func(t *testing.T) {
		resp := doJSON(http.MethodGet, "/health", nil, nil)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("status = %d, want 200", resp.StatusCode)
		}
		body, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if string(body) != "API OK!" {
			t.Fatalf("body = %q, want API OK!", body)
		}
	})
}
