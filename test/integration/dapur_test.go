package integration_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"
)

func TestDapurLifecycleAndSelection(t *testing.T) {
	name := fmt.Sprintf("Dapur IT %d", time.Now().UnixNano())
	body := map[string]any{
		"name":        name,
		"description": "Integration dapur",
	}

	createResp := doJSON(http.MethodPost, "/api/dapurs", body, superAdminCookies)
	if createResp.StatusCode != http.StatusCreated {
		r := parseResp(createResp)
		t.Fatalf("create dapur status = %d: %v", createResp.StatusCode, r.Error)
	}

	r := parseResp(createResp)
	var created map[string]any
	_ = json.Unmarshal(r.Data, &created)
	dapurID := uint(created["id"].(float64))
	defer testDB.Exec("DELETE FROM dapurs WHERE id = ?", dapurID)

	t.Run("list dapurs authenticated", func(t *testing.T) {
		resp := doJSON(http.MethodGet, "/api/dapurs", nil, adminCookies)
		if resp.StatusCode != http.StatusOK {
			rr := parseResp(resp)
			t.Fatalf("status = %d: %v", resp.StatusCode, rr.Error)
		}
	})

	t.Run("duplicate dapur returns conflict", func(t *testing.T) {
		resp := doJSON(http.MethodPost, "/api/dapurs", body, superAdminCookies)
		if resp.StatusCode != http.StatusConflict {
			t.Fatalf("status = %d, want 409", resp.StatusCode)
		}
	})

	t.Run("admin cannot create dapur", func(t *testing.T) {
		resp := doJSON(http.MethodPost, "/api/dapurs", map[string]any{
			"name":        name + " Admin",
			"description": "forbidden",
		}, adminCookies)
		if resp.StatusCode != http.StatusForbidden {
			t.Fatalf("status = %d, want 403", resp.StatusCode)
		}
	})

	t.Run("select inactive dapur is rejected", func(t *testing.T) {
		inactive := false
		resp := doJSON(http.MethodPut, fmt.Sprintf("/api/dapurs/%d", dapurID), map[string]any{
			"is_active": inactive,
		}, superAdminCookies)
		if resp.StatusCode != http.StatusOK {
			rr := parseResp(resp)
			t.Fatalf("deactivate status = %d: %v", resp.StatusCode, rr.Error)
		}

		resp = doJSON(http.MethodPost, "/api/auth/select-dapur", map[string]any{
			"dapur_id": dapurID,
		}, adminCookies)
		if resp.StatusCode != http.StatusForbidden {
			t.Fatalf("status = %d, want 403", resp.StatusCode)
		}
	})

	t.Run("select active dapur updates current auth user", func(t *testing.T) {
		active := true
		resp := doJSON(http.MethodPut, fmt.Sprintf("/api/dapurs/%d", dapurID), map[string]any{
			"is_active": active,
		}, superAdminCookies)
		if resp.StatusCode != http.StatusOK {
			rr := parseResp(resp)
			t.Fatalf("activate status = %d: %v", resp.StatusCode, rr.Error)
		}

		resp = doJSON(http.MethodPost, "/api/auth/select-dapur", map[string]any{
			"dapur_id": dapurID,
		}, adminCookies)
		if resp.StatusCode != http.StatusOK {
			rr := parseResp(resp)
			t.Fatalf("select status = %d: %v", resp.StatusCode, rr.Error)
		}

		current := doJSON(http.MethodGet, "/api/auth/_current", nil, adminCookies)
		if current.StatusCode != http.StatusOK {
			t.Fatalf("current status = %d", current.StatusCode)
		}
		cr := parseResp(current)
		var auth map[string]any
		_ = json.Unmarshal(cr.Data, &auth)
		if uint(auth["current_dapur_id"].(float64)) != dapurID {
			t.Fatalf("current_dapur_id = %v, want %d", auth["current_dapur_id"], dapurID)
		}

		resp = doJSON(http.MethodPost, "/api/auth/select-dapur", map[string]any{
			"dapur_id": testDapurID,
		}, adminCookies)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("restore default dapur status = %d", resp.StatusCode)
		}
	})

	t.Run("delete dapur success", func(t *testing.T) {
		resp := doJSON(http.MethodDelete, fmt.Sprintf("/api/dapurs/%d", dapurID), nil, superAdminCookies)
		if resp.StatusCode != http.StatusOK {
			rr := parseResp(resp)
			t.Fatalf("delete status = %d: %v", resp.StatusCode, rr.Error)
		}
	})
}

func TestDapurRequiredMiddlewareIntegration(t *testing.T) {
	username := fmt.Sprintf("nodapur%d", time.Now().UnixNano()%1_000_000_000)
	createResp := doJSON(http.MethodPost, "/api/users", map[string]any{
		"username": username,
		"fullname": "No Dapur User",
		"role":     "Admin",
		"password": "nodapurpass",
	}, superAdminCookies)
	if createResp.StatusCode != http.StatusCreated {
		r := parseResp(createResp)
		t.Fatalf("create user status = %d: %v", createResp.StatusCode, r.Error)
	}
	defer testDB.Exec("DELETE FROM users WHERE username = ?", username)

	cookies := mustLoginUser(username, "nodapurpass")
	resp := doJSON(http.MethodGet, "/api/items", nil, cookies)
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", resp.StatusCode)
	}
}
