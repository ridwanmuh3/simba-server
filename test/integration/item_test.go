package integration_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
)

// createTestItem creates an item and returns its ID. Caller is responsible for cleanup.
func createTestItem(t *testing.T, nameSuffix string) string {
	t.Helper()
	body := map[string]any{
		"name":         "Test Item " + nameSuffix,
		"category":     "Test Category",
		"stock":        10.0,
		"measure_unit": "Kg",
		"unit_price":   5000.0,
	}
	resp := doJSON(http.MethodPost, "/api/items", body, adminCookies)
	if resp.StatusCode != http.StatusCreated {
		r := parseResp(resp)
		t.Fatalf("createTestItem: got %d: %v", resp.StatusCode, r.Error)
	}
	r := parseResp(resp)
	var data map[string]any
	_ = json.Unmarshal(r.Data, &data)
	return data["id"].(string)
}

func TestAddItem(t *testing.T) {
	tests := []struct {
		name       string
		cookies    []*http.Cookie
		body       map[string]any
		wantStatus int
	}{
		{
			name:    "admin adds item success",
			cookies: adminCookies,
			body:    map[string]any{"name": "Beras Premium", "category": "Bahan Pokok", "stock": 100.0, "measure_unit": "Kg", "unit_price": 12000.0},
			wantStatus: http.StatusCreated,
		},
		{
			name:    "super admin adds item success",
			cookies: superAdminCookies,
			body:    map[string]any{"name": "Gula Merah", "category": "Bahan Pokok", "stock": 50.0, "measure_unit": "Kg", "unit_price": 18000.0},
			wantStatus: http.StatusCreated,
		},
		{
			name:       "unauthenticated returns 401",
			cookies:    nil,
			body:       map[string]any{"name": "Minyak Zaitun", "category": "Bahan", "stock": 10.0, "measure_unit": "Liter", "unit_price": 50000.0},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "missing name returns 400",
			cookies:    adminCookies,
			body:       map[string]any{"category": "Bahan", "stock": 10.0, "measure_unit": "Kg", "unit_price": 5000.0},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "zero stock returns 400",
			cookies:    adminCookies,
			body:       map[string]any{"name": "Zero Stock Item", "category": "Bahan", "stock": 0.0, "measure_unit": "Kg", "unit_price": 5000.0},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "zero unit_price returns 400",
			cookies:    adminCookies,
			body:       map[string]any{"name": "Zero Price Item", "category": "Bahan", "stock": 10.0, "measure_unit": "Kg", "unit_price": 0.0},
			wantStatus: http.StatusBadRequest,
		},
	}

	var createdIDs []string
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resp := doJSON(http.MethodPost, "/api/items", tc.body, tc.cookies)
			if resp.StatusCode != tc.wantStatus {
				r := parseResp(resp)
				t.Errorf("got %d, want %d — error: %v", resp.StatusCode, tc.wantStatus, r.Error)
			}
			if tc.wantStatus == http.StatusCreated {
				r := parseResp(resp)
				var data map[string]any
				_ = json.Unmarshal(r.Data, &data)
				if id, ok := data["id"].(string); ok {
					createdIDs = append(createdIDs, id)
				}
			}
		})
	}

	for _, id := range createdIDs {
		testDB.Exec("DELETE FROM stock_tracks WHERE item_id = ?", id)
		testDB.Exec("DELETE FROM items WHERE id = ?", id)
	}
}

func TestUpdateItem(t *testing.T) {
	id := createTestItem(t, "update-target")
	defer func() {
		testDB.Exec("DELETE FROM stock_tracks WHERE item_id = ?", id)
		testDB.Exec("DELETE FROM items WHERE id = ?", id)
	}()

	t.Run("update name and unit_price", func(t *testing.T) {
		body := map[string]any{"name": "Updated Item Name", "unit_price": 9999.0}
		resp := doJSON(http.MethodPut, "/api/items/"+id, body, adminCookies)
		if resp.StatusCode != http.StatusOK {
			r := parseResp(resp)
			t.Errorf("got %d: %v", resp.StatusCode, r.Error)
		}
	})

	t.Run("update non-existent item returns 404", func(t *testing.T) {
		body := map[string]any{"name": "Ghost Item"}
		resp := doJSON(http.MethodPut, "/api/items/NONEXISTENT-ID-9999", body, adminCookies)
		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("got %d, want 404", resp.StatusCode)
		}
	})

	t.Run("unauthenticated returns 401", func(t *testing.T) {
		body := map[string]any{"name": "Unauth Update"}
		resp := doJSON(http.MethodPut, "/api/items/"+id, body, nil)
		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("got %d, want 401", resp.StatusCode)
		}
	})
}

func TestDeleteItem(t *testing.T) {
	id := createTestItem(t, "delete-target")

	t.Run("delete item success", func(t *testing.T) {
		resp := doJSON(http.MethodDelete, "/api/items/"+id, nil, adminCookies)
		if resp.StatusCode != http.StatusOK {
			r := parseResp(resp)
			t.Errorf("got %d: %v", resp.StatusCode, r.Error)
		}
	})

	t.Run("delete already deleted returns 404", func(t *testing.T) {
		resp := doJSON(http.MethodDelete, "/api/items/"+id, nil, adminCookies)
		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("got %d, want 404", resp.StatusCode)
		}
	})

	t.Run("delete non-existent returns 404", func(t *testing.T) {
		resp := doJSON(http.MethodDelete, "/api/items/FAKEID-000", nil, adminCookies)
		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("got %d, want 404", resp.StatusCode)
		}
	})
}

func TestFindItemById(t *testing.T) {
	id := createTestItem(t, "findbyid")
	defer func() {
		testDB.Exec("DELETE FROM stock_tracks WHERE item_id = ?", id)
		testDB.Exec("DELETE FROM items WHERE id = ?", id)
	}()

	t.Run("find existing item", func(t *testing.T) {
		resp := doJSON(http.MethodGet, "/api/items/"+id, nil, adminCookies)
		if resp.StatusCode != http.StatusOK {
			r := parseResp(resp)
			t.Fatalf("got %d: %v", resp.StatusCode, r.Error)
		}

		r := parseResp(resp)
		var data map[string]any
		_ = json.Unmarshal(r.Data, &data)
		if data["id"] != id {
			t.Errorf("returned id %v, want %s", data["id"], id)
		}
	})

	t.Run("find non-existent item returns 404", func(t *testing.T) {
		resp := doJSON(http.MethodGet, "/api/items/NONEXISTENT-9999", nil, adminCookies)
		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("got %d, want 404", resp.StatusCode)
		}
	})

	t.Run("unauthenticated returns 401", func(t *testing.T) {
		resp := doJSON(http.MethodGet, "/api/items/"+id, nil, nil)
		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("got %d, want 401", resp.StatusCode)
		}
	})
}

func TestFindAllItems(t *testing.T) {
	id := createTestItem(t, "listtest")
	defer func() {
		testDB.Exec("DELETE FROM stock_tracks WHERE item_id = ?", id)
		testDB.Exec("DELETE FROM items WHERE id = ?", id)
	}()

	t.Run("list all items paginated", func(t *testing.T) {
		resp := doJSON(http.MethodGet, "/api/items?page=1&size=10", nil, adminCookies)
		if resp.StatusCode != http.StatusOK {
			rr := parseResp(resp)
			t.Fatalf("got %d: %v", resp.StatusCode, rr.Error)
		}
		r := parseResp(resp)
		if r.Paging == nil {
			t.Error("paging metadata missing")
		}
	})

	t.Run("search by name", func(t *testing.T) {
		resp := doJSON(http.MethodGet, "/api/items?search_query=Test+Item&page=1&size=10", nil, adminCookies)
		if resp.StatusCode != http.StatusOK {
			t.Errorf("got %d", resp.StatusCode)
		}
	})

	t.Run("unauthenticated returns 401", func(t *testing.T) {
		resp := doJSON(http.MethodGet, "/api/items", nil, nil)
		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("got %d, want 401", resp.StatusCode)
		}
	})
}

func TestUpdateItemStock(t *testing.T) {
	id := createTestItem(t, "stocktest")
	defer func() {
		testDB.Exec("DELETE FROM stock_tracks WHERE item_id = ?", id)
		testDB.Exec("DELETE FROM items WHERE id = ?", id)
	}()

	var stockID int

	t.Run("stock IN success", func(t *testing.T) {
		body := map[string]any{
			"type":       "IN",
			"amount":     20.0,
			"unit_price": 6000.0,
			"supplier":   "PT Test Supplier",
		}
		resp := doJSON(http.MethodPost, fmt.Sprintf("/api/items/%s/stocks", id), body, adminCookies)
		if resp.StatusCode != http.StatusCreated {
			r := parseResp(resp)
			t.Fatalf("stock IN got %d: %v", resp.StatusCode, r.Error)
		}

		r := parseResp(resp)
		var data map[string]any
		_ = json.Unmarshal(r.Data, &data)
		stockID = int(data["id"].(float64))
	})

	t.Run("stock OUT success", func(t *testing.T) {
		body := map[string]any{
			"type":       "OUT",
			"amount":     5.0,
			"unit_price": 7000.0,
		}
		resp := doJSON(http.MethodPost, fmt.Sprintf("/api/items/%s/stocks", id), body, adminCookies)
		if resp.StatusCode != http.StatusCreated {
			r := parseResp(resp)
			t.Fatalf("stock OUT got %d: %v", resp.StatusCode, r.Error)
		}
	})

	t.Run("stock OUT exceeding available stock returns 400", func(t *testing.T) {
		// Item created with 10 stock, IN +20 = 30, OUT -5 = 25; try to take out 9999
		body := map[string]any{
			"type":       "OUT",
			"amount":     9999.0,
			"unit_price": 7000.0,
		}
		resp := doJSON(http.MethodPost, fmt.Sprintf("/api/items/%s/stocks", id), body, adminCookies)
		if resp.StatusCode != http.StatusBadRequest && resp.StatusCode != http.StatusUnprocessableEntity {
			t.Errorf("got %d, want 400 or 422 for insufficient stock", resp.StatusCode)
		}
	})

	t.Run("stock IN with zero amount returns 400", func(t *testing.T) {
		body := map[string]any{
			"type":       "IN",
			"amount":     0.0,
			"unit_price": 5000.0,
		}
		resp := doJSON(http.MethodPost, fmt.Sprintf("/api/items/%s/stocks", id), body, adminCookies)
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("got %d, want 400", resp.StatusCode)
		}
	})

	t.Run("stock on non-existent item returns 404", func(t *testing.T) {
		body := map[string]any{"type": "IN", "amount": 5.0, "unit_price": 5000.0}
		resp := doJSON(http.MethodPost, "/api/items/FAKEID-000/stocks", body, adminCookies)
		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("got %d, want 404", resp.StatusCode)
		}
	})

	t.Run("unauthenticated returns 401", func(t *testing.T) {
		body := map[string]any{"type": "IN", "amount": 5.0, "unit_price": 5000.0}
		resp := doJSON(http.MethodPost, fmt.Sprintf("/api/items/%s/stocks", id), body, nil)
		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("got %d, want 401", resp.StatusCode)
		}
	})

	// Edit stock — depends on stockID from stock IN above
	t.Run("edit stock success", func(t *testing.T) {
		if stockID == 0 {
			t.Skip("stockID not set, skipping edit test")
		}
		body := map[string]any{
			"amount":     25.0,
			"unit_price": 6500.0,
			"supplier":   "PT Updated Supplier",
		}
		resp := doJSON(http.MethodPatch, fmt.Sprintf("/api/items/%s/stocks/%d", id, stockID), body, adminCookies)
		if resp.StatusCode != http.StatusOK {
			r := parseResp(resp)
			t.Errorf("edit stock got %d: %v", resp.StatusCode, r.Error)
		}
	})

	t.Run("edit non-existent stock returns 404", func(t *testing.T) {
		body := map[string]any{"amount": 5.0, "unit_price": 5000.0}
		resp := doJSON(http.MethodPatch, fmt.Sprintf("/api/items/%s/stocks/999999", id), body, adminCookies)
		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("got %d, want 404", resp.StatusCode)
		}
	})

	t.Run("delete stock success", func(t *testing.T) {
		if stockID == 0 {
			t.Skip("stockID not set, skipping delete test")
		}
		resp := doJSON(http.MethodDelete, fmt.Sprintf("/api/items/%s/stocks/%d", id, stockID), nil, adminCookies)
		if resp.StatusCode != http.StatusOK {
			r := parseResp(resp)
			t.Errorf("delete stock got %d: %v", resp.StatusCode, r.Error)
		}
	})
}

func TestFindAllStocks(t *testing.T) {
	t.Run("list all stocks", func(t *testing.T) {
		resp := doJSON(http.MethodGet, "/api/items/stocks?page=1&size=10", nil, adminCookies)
		if resp.StatusCode != http.StatusOK {
			rr := parseResp(resp)
			t.Fatalf("got %d: %v", resp.StatusCode, rr.Error)
		}
	})

	t.Run("filter by type IN", func(t *testing.T) {
		resp := doJSON(http.MethodGet, "/api/items/stocks?type=IN&page=1&size=10", nil, adminCookies)
		if resp.StatusCode != http.StatusOK {
			t.Errorf("got %d", resp.StatusCode)
		}
	})

	t.Run("filter by type OUT", func(t *testing.T) {
		resp := doJSON(http.MethodGet, "/api/items/stocks?type=OUT&page=1&size=10", nil, adminCookies)
		if resp.StatusCode != http.StatusOK {
			t.Errorf("got %d", resp.StatusCode)
		}
	})

	t.Run("unauthenticated returns 401", func(t *testing.T) {
		resp := doJSON(http.MethodGet, "/api/items/stocks", nil, nil)
		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("got %d, want 401", resp.StatusCode)
		}
	})
}

func TestStockFinanceSummary(t *testing.T) {
	t.Run("get summary authenticated", func(t *testing.T) {
		resp := doJSON(http.MethodGet, "/api/items/stocks/summary", nil, adminCookies)
		if resp.StatusCode != http.StatusOK {
			rr := parseResp(resp)
			t.Fatalf("got %d: %v", resp.StatusCode, rr.Error)
		}
	})

	t.Run("unauthenticated returns 401", func(t *testing.T) {
		resp := doJSON(http.MethodGet, "/api/items/stocks/summary", nil, nil)
		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("got %d, want 401", resp.StatusCode)
		}
	})
}

func TestItemStockOpname(t *testing.T) {
	t.Run("get opname summary", func(t *testing.T) {
		resp := doJSON(http.MethodGet, "/api/items/stocks/opname?page=1&size=10", nil, adminCookies)
		if resp.StatusCode != http.StatusOK {
			rr := parseResp(resp)
			t.Fatalf("got %d: %v", resp.StatusCode, rr.Error)
		}
	})
}
