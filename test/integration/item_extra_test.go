package integration_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/ridwanmuh3/simba-server/internal/entity"
)

func TestItemImportExportCategoriesAndLastPrice(t *testing.T) {
	suffix := time.Now().UnixNano()
	name := fmt.Sprintf("Import Rice %d", suffix)
	category := fmt.Sprintf("ImportCat%d", suffix)
	csvBody := fmt.Sprintf("Nama,Kategori,Jumlah,Satuan,Harga Satuan\n%s,%s,2,Kg,1000\n", name, category)

	resp := doMultipart(http.MethodPost, "/api/items/import", nil, "import_file", "items.csv", []byte(csvBody), adminCookies)
	if resp.StatusCode != http.StatusOK {
		r := parseResp(resp)
		t.Fatalf("import status = %d: %v", resp.StatusCode, r.Error)
	}

	var item entity.Item
	if err := testDB.Where("name = ? AND dapur_id = ?", name, testDapurID).First(&item).Error; err != nil {
		t.Fatalf("query imported item: %v", err)
	}
	defer func() {
		testDB.Exec("DELETE FROM stock_tracks WHERE item_id = ?", item.ID)
		testDB.Exec("DELETE FROM items WHERE id = ?", item.ID)
	}()

	t.Run("categories include imported category", func(t *testing.T) {
		resp := doJSON(http.MethodGet, "/api/items/categories", nil, adminCookies)
		if resp.StatusCode != http.StatusOK {
			r := parseResp(resp)
			t.Fatalf("categories status = %d: %v", resp.StatusCode, r.Error)
		}

		r := parseResp(resp)
		var categories []string
		_ = json.Unmarshal(r.Data, &categories)
		found := false
		for _, got := range categories {
			if got == category {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("category %q not found in %+v", category, categories)
		}
	})

	t.Run("export includes imported item", func(t *testing.T) {
		resp := doJSON(http.MethodGet, "/api/items/export", nil, adminCookies)
		if resp.StatusCode != http.StatusOK {
			r := parseResp(resp)
			t.Fatalf("export status = %d: %v", resp.StatusCode, r.Error)
		}

		r := parseResp(resp)
		if !strings.Contains(string(r.Data), name) {
			t.Fatalf("export data missing item %q: %s", name, string(r.Data))
		}
	})

	t.Run("last stock price follows latest stock entry", func(t *testing.T) {
		stockResp := doJSON(http.MethodPost, fmt.Sprintf("/api/items/%s/stocks", item.ID), map[string]any{
			"type":       "IN",
			"amount":     1,
			"unit_price": 3210,
			"supplier":   "Import Supplier",
		}, adminCookies)
		if stockResp.StatusCode != http.StatusCreated {
			r := parseResp(stockResp)
			t.Fatalf("stock status = %d: %v", stockResp.StatusCode, r.Error)
		}

		resp := doJSON(http.MethodGet, fmt.Sprintf("/api/items/%s/stocks/last-price?type=IN", item.ID), nil, adminCookies)
		if resp.StatusCode != http.StatusOK {
			r := parseResp(resp)
			t.Fatalf("last-price status = %d: %v", resp.StatusCode, r.Error)
		}

		r := parseResp(resp)
		var price float64
		_ = json.Unmarshal(r.Data, &price)
		if price != 3210 {
			t.Fatalf("last price = %v, want 3210", price)
		}
	})
}

func TestImportItemsRejectsBadCsv(t *testing.T) {
	resp := doMultipart(http.MethodPost, "/api/items/import", nil, "import_file", "items.txt", []byte("bad"), adminCookies)
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422", resp.StatusCode)
	}
}
