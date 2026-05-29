package integration_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/ridwanmuh3/simba-server/internal/entity"
	"github.com/ridwanmuh3/simba-server/internal/service"
)

func TestSettingsRoutes(t *testing.T) {
	defer testDB.Exec("DELETE FROM app_settings WHERE dapur_id = ?", testDapurID)

	t.Run("default company profile returns default bank account", func(t *testing.T) {
		resp := doJSON(http.MethodGet, "/api/settings/company", nil, adminCookies)
		if resp.StatusCode != http.StatusOK {
			r := parseResp(resp)
			t.Fatalf("status = %d: %v", resp.StatusCode, r.Error)
		}

		r := parseResp(resp)
		var profile map[string]any
		_ = json.Unmarshal(r.Data, &profile)
		if profile["bank_account"] != service.DefaultBankAccount {
			t.Fatalf("bank_account = %v, want default", profile["bank_account"])
		}
	})

	t.Run("update and get company profile", func(t *testing.T) {
		body := map[string]any{
			"company_name":    "PT Integration",
			"company_address": "Jl Integration",
			"company_contact": "0800000000",
			"bank_account":    "BNI Test",
			"penanggungjawab": "PIC Integration",
			"jabatan":         "Manager",
		}
		resp := doJSON(http.MethodPut, "/api/settings/company", body, adminCookies)
		if resp.StatusCode != http.StatusOK {
			r := parseResp(resp)
			t.Fatalf("update status = %d: %v", resp.StatusCode, r.Error)
		}

		resp = doJSON(http.MethodGet, "/api/settings/company", nil, adminCookies)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("get status = %d", resp.StatusCode)
		}
		r := parseResp(resp)
		var profile map[string]any
		_ = json.Unmarshal(r.Data, &profile)
		if profile["company_name"] != body["company_name"] {
			t.Fatalf("company_name = %v, want %v", profile["company_name"], body["company_name"])
		}
	})

	t.Run("invalid company profile returns bad request", func(t *testing.T) {
		resp := doJSON(http.MethodPut, "/api/settings/company", map[string]any{
			"company_name": "",
		}, adminCookies)
		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", resp.StatusCode)
		}
	})

	t.Run("next document numbers follow invoice history", func(t *testing.T) {
		invoice := entity.Invoice{
			DapurID:         testDapurID,
			StockType:       "OUT",
			CompanyName:     "PT Sequence",
			CompanyContact:  "0800",
			CompanyAddress:  "Jl Sequence",
			InvoiceNumber:   "INV-009",
			PONumber:        "PO-007",
			ReceiverName:    "Client",
			ReceiverAddress: "Address",
			InvoiceDate:     "2026-05-27",
			Penanggungjawab: "PIC",
			Jabatan:         "Manager",
		}
		if err := testDB.Create(&invoice).Error; err != nil {
			t.Fatalf("create invoice sequence fixture: %v", err)
		}
		defer testDB.Delete(&invoice)

		resp := doJSON(http.MethodGet, "/api/settings/next-document-number", nil, adminCookies)
		if resp.StatusCode != http.StatusOK {
			r := parseResp(resp)
			t.Fatalf("status = %d: %v", resp.StatusCode, r.Error)
		}

		r := parseResp(resp)
		var numbers map[string]any
		_ = json.Unmarshal(r.Data, &numbers)
		if numbers["next_invoice_no"] != "INV-010" || numbers["next_po_no"] != "PO-008" {
			t.Fatalf("numbers = %+v, want INV-010/PO-008", numbers)
		}
	})
}

func TestDashboardRoute(t *testing.T) {
	suffix := time.Now().UnixNano()
	item := entity.Item{
		ID:           fmt.Sprintf("MBG-BHN-DASH-%d", suffix),
		Name:         "Dashboard Item",
		Category:     "Dashboard",
		InitialStock: 10,
		Stock:        10,
		MeasureUnit:  "Kg",
		UnitPrice:    1000,
		TotalPrice:   10000,
		ModifiedBy:   testAdminFullname,
		DapurID:      testDapurID,
	}
	if err := testDB.Create(&item).Error; err != nil {
		t.Fatalf("create dashboard item: %v", err)
	}
	defer func() {
		testDB.Exec("DELETE FROM stock_tracks WHERE item_id = ?", item.ID)
		testDB.Unscoped().Delete(&item)
	}()

	stock := entity.StockTracking{
		Type:          "IN",
		Amount:        5,
		PreviousStock: 10,
		NewStock:      15,
		UnitPrice:     1000,
		TotalPrice:    5000,
		Supplier:      "Dashboard Supplier",
		ModifiedBy:    testAdminFullname,
		DapurID:       testDapurID,
		ItemID:        item.ID,
	}
	if err := testDB.Create(&stock).Error; err != nil {
		t.Fatalf("create dashboard stock: %v", err)
	}

	finances := []entity.Finance{
		{Type: "PEMASUKAN", Category: "Dashboard", Description: "In", Amount: 10000, ProofImage: "/uploads/test.png", ModifiedBy: testAdminFullname, DapurID: testDapurID},
		{Type: "PENGELUARAN", Category: "Dashboard", Description: "Out", Amount: 3000, ProofImage: "/uploads/test.png", ModifiedBy: testAdminFullname, DapurID: testDapurID},
	}
	if err := testDB.Create(&finances).Error; err != nil {
		t.Fatalf("create dashboard finances: %v", err)
	}
	defer testDB.Exec("DELETE FROM finances WHERE category = ? AND dapur_id = ?", "Dashboard", testDapurID)

	resp := doJSON(http.MethodGet, "/api/dashboard", nil, adminCookies)
	if resp.StatusCode != http.StatusOK {
		r := parseResp(resp)
		t.Fatalf("status = %d: %v", resp.StatusCode, r.Error)
	}

	r := parseResp(resp)
	var dashboard map[string]any
	_ = json.Unmarshal(r.Data, &dashboard)
	if dashboard["total_items"] == nil || dashboard["total_budget"] == nil || dashboard["monthly_budget"] == nil {
		t.Fatalf("dashboard missing expected fields: %+v", dashboard)
	}

	unauth := doJSON(http.MethodGet, "/api/dashboard", nil, nil)
	if unauth.StatusCode != http.StatusUnauthorized {
		t.Fatalf("unauth status = %d, want 401", unauth.StatusCode)
	}
}
