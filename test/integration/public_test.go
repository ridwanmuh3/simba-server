package integration_test

import (
	"encoding/json"
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

	t.Run("health reports status and database probe", func(t *testing.T) {
		resp := doJSON(http.MethodGet, "/health", nil, nil)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("status = %d, want 200", resp.StatusCode)
		}

		r := parseResp(resp)
		var data map[string]any
		_ = json.Unmarshal(r.Data, &data)

		if data["status"] != "healthy" {
			t.Errorf("status = %v, want healthy", data["status"])
		}
		if data["db"] != "connected" {
			t.Errorf("db = %v, want connected", data["db"])
		}
		for _, key := range []string{"uptime", "version"} {
			if v, ok := data[key].(string); !ok || v == "" {
				t.Errorf("%s missing or empty in health payload: %v", key, data[key])
			}
		}
	})
}
