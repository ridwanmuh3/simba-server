package integration_test

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/ridwanmuh3/simba-server/internal/entity"
)

func TestInvoiceDownloadUpdateFlatAndDelete(t *testing.T) {
	invoiceID, _, _, cleanup := createTestInvoice(t, "extra-flow")
	defer cleanup()

	t.Run("download invoice pdf inline", func(t *testing.T) {
		resp := doJSON(http.MethodGet, fmt.Sprintf("/api/items/invoices/%d/pdf?mode=view", invoiceID), nil, adminCookies)
		if resp.StatusCode != http.StatusOK {
			r := parseResp(resp)
			t.Fatalf("download status = %d: %v", resp.StatusCode, r.Error)
		}
		defer resp.Body.Close()

		if !strings.Contains(resp.Header.Get("Content-Type"), "application/pdf") {
			t.Fatalf("content type = %q, want application/pdf", resp.Header.Get("Content-Type"))
		}
		if !strings.Contains(resp.Header.Get("Content-Disposition"), "inline") {
			t.Fatalf("content disposition = %q, want inline", resp.Header.Get("Content-Disposition"))
		}
		_, _ = io.Copy(io.Discard, resp.Body)
	})

	t.Run("flat invoice item list returns rows", func(t *testing.T) {
		resp := doJSON(http.MethodGet, "/api/items/invoices/items-flat?stock_type=OUT&page=1&size=10", nil, adminCookies)
		if resp.StatusCode != http.StatusOK {
			r := parseResp(resp)
			t.Fatalf("flat status = %d: %v", resp.StatusCode, r.Error)
		}

		r := parseResp(resp)
		var rows []map[string]any
		_ = json.Unmarshal(r.Data, &rows)
		if len(rows) == 0 {
			t.Fatal("flat invoice list returned no rows")
		}
		if r.Paging == nil {
			t.Fatal("flat invoice paging missing")
		}
	})

	t.Run("update invoice metadata", func(t *testing.T) {
		resp := doJSON(http.MethodPatch, fmt.Sprintf("/api/items/invoices/%d", invoiceID), map[string]any{
			"company_name":     "PT Updated Invoice",
			"company_address":  "Jl Updated Invoice",
			"company_contact":  "081111111",
			"receiver_name":    "Updated Receiver",
			"receiver_address": "Updated Address",
			"invoice_date":     "2026-05-27",
			"keterangan":       "Updated Note",
			"penanggungjawab":  "Updated PIC",
			"jabatan":          "Director",
			"bank_account":     "Updated Bank",
		}, adminCookies)
		if resp.StatusCode != http.StatusOK {
			r := parseResp(resp)
			t.Fatalf("update status = %d: %v", resp.StatusCode, r.Error)
		}

		detail := doJSON(http.MethodGet, fmt.Sprintf("/api/items/invoices/%d", invoiceID), nil, adminCookies)
		if detail.StatusCode != http.StatusOK {
			t.Fatalf("detail status = %d", detail.StatusCode)
		}
		r := parseResp(detail)
		var data map[string]any
		_ = json.Unmarshal(r.Data, &data)
		if data["company_name"] != "PT Updated Invoice" {
			t.Fatalf("company_name = %v, want PT Updated Invoice", data["company_name"])
		}
	})

	t.Run("delete invoice", func(t *testing.T) {
		resp := doJSON(http.MethodDelete, fmt.Sprintf("/api/items/invoices/%d", invoiceID), nil, adminCookies)
		if resp.StatusCode != http.StatusOK {
			r := parseResp(resp)
			t.Fatalf("delete status = %d: %v", resp.StatusCode, r.Error)
		}

		resp = doJSON(http.MethodGet, fmt.Sprintf("/api/items/invoices/%d", invoiceID), nil, adminCookies)
		if resp.StatusCode != http.StatusNotFound {
			t.Fatalf("detail after delete status = %d, want 404", resp.StatusCode)
		}
	})
}

// TestInvoiceSchemaCascadesToItems guards the ON DELETE CASCADE that
// entity.Invoice declares for its line items. Without it, deleting an invoice
// row while items still reference it fails at the database level, so any
// caller that does not clear the children first (raw SQL, maintenance scripts,
// a future repository method) breaks.
func TestInvoiceSchemaCascadesToItems(t *testing.T) {
	invoice := entity.Invoice{
		DapurID:         testDapurID,
		StockType:       "OUT",
		CompanyName:     "PT Cascade",
		CompanyContact:  "0800",
		CompanyAddress:  "Jl Cascade",
		InvoiceNumber:   fmt.Sprintf("INV-CASCADE-%d", time.Now().UnixNano()),
		ReceiverName:    "Client",
		ReceiverAddress: "Address",
		InvoiceDate:     "2026-05-27",
		Penanggungjawab: "PIC",
		Jabatan:         "Manager",
	}
	if err := testDB.Create(&invoice).Error; err != nil {
		t.Fatalf("create invoice fixture: %v", err)
	}
	defer func() {
		testDB.Exec("DELETE FROM invoice_items WHERE invoice_id = ?", invoice.ID)
		testDB.Exec("DELETE FROM invoices WHERE id = ?", invoice.ID)
	}()

	items := []entity.InvoiceItem{
		{InvoiceID: invoice.ID, ItemName: "Cascade Item A", Amount: 1, UnitPrice: 1000, TotalPrice: 1000},
		{InvoiceID: invoice.ID, ItemName: "Cascade Item B", Amount: 2, UnitPrice: 1500, TotalPrice: 3000},
	}
	if err := testDB.Create(&items).Error; err != nil {
		t.Fatalf("create invoice items fixture: %v", err)
	}

	// Delete the parent alone; the children must go with it.
	if err := testDB.Exec("DELETE FROM invoices WHERE id = ?", invoice.ID).Error; err != nil {
		t.Fatalf("deleting an invoice with line items failed: %v", err)
	}

	var remaining int64
	if err := testDB.Model(&entity.InvoiceItem{}).
		Where("invoice_id = ?", invoice.ID).
		Count(&remaining).Error; err != nil {
		t.Fatalf("count orphaned items: %v", err)
	}
	if remaining != 0 {
		t.Errorf("orphaned invoice items after parent delete = %d, want 0", remaining)
	}
}
