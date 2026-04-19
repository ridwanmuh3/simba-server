package integration_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
)

// createTestFinance creates a finance record and returns its ID.
func createTestFinance(t *testing.T) int {
	t.Helper()
	fields := map[string]string{
		"type":        "PEMASUKAN",
		"category":    "Penjualan",
		"description": "Test finance record",
		"amount":      "100000",
		"extra_note":  "test note",
	}
	resp := doMultipart(http.MethodPost, "/api/finances", fields, "proof_image", "proof.png", minimalPNG, adminCookies)
	if resp.StatusCode != http.StatusCreated {
		r := parseResp(resp)
		t.Fatalf("createTestFinance: got %d: %v", resp.StatusCode, r.Error)
	}
	r := parseResp(resp)
	var data map[string]any
	_ = json.Unmarshal(r.Data, &data)
	return int(data["id"].(float64))
}

func TestAddFinance(t *testing.T) {
	t.Run("admin adds finance success", func(t *testing.T) {
		fields := map[string]string{
			"type":        "PEMASUKAN",
			"category":    "Penjualan",
			"description": "Penjualan harian",
			"amount":      "500000",
		}
		resp := doMultipart(http.MethodPost, "/api/finances", fields, "proof_image", "bukti.png", minimalPNG, adminCookies)
		if resp.StatusCode != http.StatusCreated {
			r := parseResp(resp)
			t.Fatalf("got %d: %v", resp.StatusCode, r.Error)
		}

		r := parseResp(resp)
		var data map[string]any
		_ = json.Unmarshal(r.Data, &data)
		financeID := int(data["id"].(float64))
		defer testDB.Exec("DELETE FROM finances WHERE id = ?", financeID)

		if data["type"] != "PEMASUKAN" {
			t.Errorf("type got %v, want PEMASUKAN", data["type"])
		}
	})

	t.Run("pengeluaran type success", func(t *testing.T) {
		fields := map[string]string{
			"type":        "PENGELUARAN",
			"category":    "Operasional",
			"description": "Bayar listrik",
			"amount":      "300000",
		}
		resp := doMultipart(http.MethodPost, "/api/finances", fields, "proof_image", "bukti.png", minimalPNG, adminCookies)
		if resp.StatusCode != http.StatusCreated {
			r := parseResp(resp)
			t.Fatalf("got %d: %v", resp.StatusCode, r.Error)
		}
		r := parseResp(resp)
		var data map[string]any
		_ = json.Unmarshal(r.Data, &data)
		defer testDB.Exec("DELETE FROM finances WHERE id = ?", int(data["id"].(float64)))
	})

	t.Run("missing proof_image returns 400", func(t *testing.T) {
		fields := map[string]string{
			"type":        "PEMASUKAN",
			"category":    "Penjualan",
			"description": "No image finance",
			"amount":      "100000",
		}
		// No file attached
		resp := doMultipart(http.MethodPost, "/api/finances", fields, "", "", nil, adminCookies)
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("got %d, want 400", resp.StatusCode)
		}
	})

	t.Run("invalid file format returns 400", func(t *testing.T) {
		fields := map[string]string{
			"type":        "PEMASUKAN",
			"category":    "Penjualan",
			"description": "Bad file type",
			"amount":      "100000",
		}
		// Upload a .txt file — should be rejected
		resp := doMultipart(http.MethodPost, "/api/finances", fields, "proof_image", "proof.txt", []byte("not an image"), adminCookies)
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("got %d, want 400", resp.StatusCode)
		}
	})

	t.Run("unauthenticated returns 401", func(t *testing.T) {
		fields := map[string]string{
			"type":        "PEMASUKAN",
			"category":    "Penjualan",
			"description": "Unauth finance",
			"amount":      "100000",
		}
		resp := doMultipart(http.MethodPost, "/api/finances", fields, "proof_image", "proof.png", minimalPNG, nil)
		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("got %d, want 401", resp.StatusCode)
		}
	})
}

func TestFindFinanceById(t *testing.T) {
	id := createTestFinance(t)
	defer testDB.Exec("DELETE FROM finances WHERE id = ?", id)

	t.Run("find existing finance", func(t *testing.T) {
		resp := doJSON(http.MethodGet, fmt.Sprintf("/api/finances/%d", id), nil, adminCookies)
		if resp.StatusCode != http.StatusOK {
			r := parseResp(resp)
			t.Fatalf("got %d: %v", resp.StatusCode, r.Error)
		}

		r := parseResp(resp)
		var data map[string]any
		_ = json.Unmarshal(r.Data, &data)
		if int(data["id"].(float64)) != id {
			t.Errorf("returned id %v, want %d", data["id"], id)
		}
	})

	t.Run("find non-existent returns 404", func(t *testing.T) {
		resp := doJSON(http.MethodGet, "/api/finances/999999", nil, adminCookies)
		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("got %d, want 404", resp.StatusCode)
		}
	})

	t.Run("unauthenticated returns 401", func(t *testing.T) {
		resp := doJSON(http.MethodGet, fmt.Sprintf("/api/finances/%d", id), nil, nil)
		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("got %d, want 401", resp.StatusCode)
		}
	})
}

func TestFindAllFinances(t *testing.T) {
	id := createTestFinance(t)
	defer testDB.Exec("DELETE FROM finances WHERE id = ?", id)

	t.Run("list all finances paginated", func(t *testing.T) {
		resp := doJSON(http.MethodGet, "/api/finances?page=1&size=10", nil, adminCookies)
		if resp.StatusCode != http.StatusOK {
			rr := parseResp(resp)
			t.Fatalf("got %d: %v", resp.StatusCode, rr.Error)
		}
		r := parseResp(resp)
		if r.Paging == nil {
			t.Error("paging metadata missing")
		}
	})

	t.Run("filter by search query", func(t *testing.T) {
		resp := doJSON(http.MethodGet, "/api/finances?search_query=Test&page=1&size=10", nil, adminCookies)
		if resp.StatusCode != http.StatusOK {
			t.Errorf("got %d", resp.StatusCode)
		}
	})

	t.Run("filter by date range", func(t *testing.T) {
		resp := doJSON(http.MethodGet, "/api/finances?start_date=2020-01-01&end_date=2099-12-31&page=1&size=10", nil, adminCookies)
		if resp.StatusCode != http.StatusOK {
			t.Errorf("got %d", resp.StatusCode)
		}
	})

	t.Run("unauthenticated returns 401", func(t *testing.T) {
		resp := doJSON(http.MethodGet, "/api/finances", nil, nil)
		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("got %d, want 401", resp.StatusCode)
		}
	})
}

func TestUpdateFinance(t *testing.T) {
	id := createTestFinance(t)
	defer testDB.Exec("DELETE FROM finances WHERE id = ?", id)

	t.Run("update category and amount", func(t *testing.T) {
		body := map[string]any{
			"type":        "PENGELUARAN",
			"category":    "Operasional",
			"description": "Updated description",
			"amount":      250000,
		}
		resp := doJSON(http.MethodPut, fmt.Sprintf("/api/finances/%d", id), body, adminCookies)
		if resp.StatusCode != http.StatusOK {
			r := parseResp(resp)
			t.Errorf("got %d: %v", resp.StatusCode, r.Error)
		}
	})

	t.Run("update non-existent returns 404", func(t *testing.T) {
		body := map[string]any{"type": "PEMASUKAN", "category": "Penjualan", "amount": 100000}
		resp := doJSON(http.MethodPut, "/api/finances/999999", body, adminCookies)
		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("got %d, want 404", resp.StatusCode)
		}
	})

	t.Run("unauthenticated returns 401", func(t *testing.T) {
		body := map[string]any{"type": "PEMASUKAN", "amount": 100000}
		resp := doJSON(http.MethodPut, fmt.Sprintf("/api/finances/%d", id), body, nil)
		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("got %d, want 401", resp.StatusCode)
		}
	})
}

func TestDeleteFinance(t *testing.T) {
	id := createTestFinance(t)

	t.Run("delete success", func(t *testing.T) {
		resp := doJSON(http.MethodDelete, fmt.Sprintf("/api/finances/%d", id), nil, adminCookies)
		if resp.StatusCode != http.StatusOK {
			r := parseResp(resp)
			t.Fatalf("got %d: %v", resp.StatusCode, r.Error)
		}
	})

	t.Run("delete already deleted returns 404", func(t *testing.T) {
		resp := doJSON(http.MethodDelete, fmt.Sprintf("/api/finances/%d", id), nil, adminCookies)
		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("got %d, want 404", resp.StatusCode)
		}
	})

	t.Run("delete non-existent returns 404", func(t *testing.T) {
		resp := doJSON(http.MethodDelete, "/api/finances/999999", nil, adminCookies)
		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("got %d, want 404", resp.StatusCode)
		}
	})

	t.Run("unauthenticated returns 401", func(t *testing.T) {
		resp := doJSON(http.MethodDelete, fmt.Sprintf("/api/finances/%d", id), nil, nil)
		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("got %d, want 401", resp.StatusCode)
		}
	})
}

func TestExportFinances(t *testing.T) {
	t.Run("export authenticated returns data", func(t *testing.T) {
		resp := doJSON(http.MethodGet, "/api/finances/export", nil, adminCookies)
		if resp.StatusCode != http.StatusOK {
			rr := parseResp(resp)
			t.Fatalf("got %d: %v", resp.StatusCode, rr.Error)
		}
	})

	t.Run("unauthenticated returns 401", func(t *testing.T) {
		resp := doJSON(http.MethodGet, "/api/finances/export", nil, nil)
		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("got %d, want 401", resp.StatusCode)
		}
	})
}
