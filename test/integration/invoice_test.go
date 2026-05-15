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

type invoiceHistoryEntry struct {
	ID            uint   `json:"id"`
	InvoiceNumber string `json:"invoice_number"`
	PONumber      string `json:"po_number"`
	Kebutuhan     string `json:"kebutuhan"`
}

type invoiceDetailEntry struct {
	ID            uint   `json:"id"`
	InvoiceNumber string `json:"invoice_number"`
	PONumber      string `json:"po_number"`
	Kebutuhan     string `json:"kebutuhan"`
	Items         []struct {
		ID uint `json:"id"`
	} `json:"items"`
}

func createTestInvoice(t *testing.T, suffix string) (uint, string, string, func()) {
	t.Helper()

	itemID := createTestItem(t, suffix)

	stockBody := map[string]any{
		"type":       "OUT",
		"amount":     2.0,
		"unit_price": 7000.0,
		"supplier":   "Supplier " + suffix,
	}
	stockResp := doJSON(http.MethodPost, fmt.Sprintf("/api/items/%s/stocks", itemID), stockBody, adminCookies)
	if stockResp.StatusCode != http.StatusCreated {
		r := parseResp(stockResp)
		t.Fatalf("createTestInvoice stock create got %d: %v", stockResp.StatusCode, r.Error)
	}

	stockResult := parseResp(stockResp)
	var stockData map[string]any
	_ = json.Unmarshal(stockResult.Data, &stockData)
	stockID := int(stockData["id"].(float64))

	sequence := time.Now().UnixNano()
	invoiceNo := fmt.Sprintf("INV-%d", sequence)
	poNo := fmt.Sprintf("PO-%d", sequence)
	kebutuhan := "Kebutuhan " + suffix

	invoiceBody := map[string]any{
		"company_name":     "PT Test Invoice",
		"company_address":  "Jl Test Invoice",
		"company_contact":  "0800123456",
		"invoice_no":       invoiceNo,
		"po_no":            poNo,
		"kebutuhan":        kebutuhan,
		"receiver_name":    "Client Test",
		"receiver_address": "Jl Client Test",
		"date":             "2026-05-14 09:00:00",
		"stock_type":       "OUT",
		"stock_ids":        []int{stockID},
		"keterangan":       "Catatan test",
		"penanggungjawab":  "PIC Test",
		"jabatan":          "Manager",
		"bank_account":     "BNI Test",
	}

	invoiceResp := doJSON(http.MethodPost, "/api/items/invoices", invoiceBody, adminCookies)
	if invoiceResp.StatusCode != http.StatusOK {
		r := parseResp(invoiceResp)
		t.Fatalf("createTestInvoice invoice create got %d: %v", invoiceResp.StatusCode, r.Error)
	}
	if contentType := invoiceResp.Header.Get("Content-Type"); !strings.Contains(contentType, "application/pdf") {
		t.Fatalf("createTestInvoice unexpected content type: %q", contentType)
	}
	_, _ = io.Copy(io.Discard, invoiceResp.Body)
	_ = invoiceResp.Body.Close()

	var invoice entity.Invoice
	if err := testDB.Where("invoice_number = ?", invoiceNo).First(&invoice).Error; err != nil {
		t.Fatalf("createTestInvoice query invoice failed: %v", err)
	}

	cleanup := func() {
		testDB.Exec("DELETE FROM invoices WHERE id = ?", invoice.ID)
		testDB.Exec("DELETE FROM stock_tracks WHERE id = ?", stockID)
		testDB.Exec("DELETE FROM stock_tracks WHERE item_id = ?", itemID)
		testDB.Exec("DELETE FROM items WHERE id = ?", itemID)
	}

	return invoice.ID, invoiceNo, kebutuhan, cleanup
}

func TestInvoiceKebutuhanFlow(t *testing.T) {
	invoiceID, invoiceNo, kebutuhan, cleanup := createTestInvoice(t, "kebutuhan-flow")
	defer cleanup()

	t.Run("persist kebutuhan on create", func(t *testing.T) {
		var invoice entity.Invoice
		if err := testDB.First(&invoice, invoiceID).Error; err != nil {
			t.Fatalf("query invoice failed: %v", err)
		}
		if invoice.Kebutuhan != kebutuhan {
			t.Fatalf("stored kebutuhan = %q, want %q", invoice.Kebutuhan, kebutuhan)
		}
	})

	t.Run("history returns kebutuhan", func(t *testing.T) {
		resp := doJSON(
			http.MethodGet,
			fmt.Sprintf("/api/items/invoices?search_query=%s&page=1&size=10", kebutuhan),
			nil,
			adminCookies,
		)
		if resp.StatusCode != http.StatusOK {
			r := parseResp(resp)
			t.Fatalf("history got %d: %v", resp.StatusCode, r.Error)
		}

		r := parseResp(resp)
		var invoices []invoiceHistoryEntry
		_ = json.Unmarshal(r.Data, &invoices)

		if len(invoices) == 0 {
			t.Fatal("history returned no invoices")
		}
		if invoices[0].InvoiceNumber != invoiceNo {
			t.Fatalf("history invoice_number = %q, want %q", invoices[0].InvoiceNumber, invoiceNo)
		}
		if invoices[0].Kebutuhan != kebutuhan {
			t.Fatalf("history kebutuhan = %q, want %q", invoices[0].Kebutuhan, kebutuhan)
		}
	})

	t.Run("detail returns kebutuhan", func(t *testing.T) {
		resp := doJSON(http.MethodGet, fmt.Sprintf("/api/items/invoices/%d", invoiceID), nil, adminCookies)
		if resp.StatusCode != http.StatusOK {
			r := parseResp(resp)
			t.Fatalf("detail got %d: %v", resp.StatusCode, r.Error)
		}

		r := parseResp(resp)
		var invoice invoiceDetailEntry
		_ = json.Unmarshal(r.Data, &invoice)

		if invoice.InvoiceNumber != invoiceNo {
			t.Fatalf("detail invoice_number = %q, want %q", invoice.InvoiceNumber, invoiceNo)
		}
		if invoice.Kebutuhan != kebutuhan {
			t.Fatalf("detail kebutuhan = %q, want %q", invoice.Kebutuhan, kebutuhan)
		}
		if len(invoice.Items) == 0 {
			t.Fatal("detail returned no items")
		}
	})

	t.Run("search by kebutuhan works", func(t *testing.T) {
		resp := doJSON(
			http.MethodGet,
			fmt.Sprintf("/api/items/invoices?search_query=%s&page=1&size=10", "flow"),
			nil,
			adminCookies,
		)
		if resp.StatusCode != http.StatusOK {
			r := parseResp(resp)
			t.Fatalf("search got %d: %v", resp.StatusCode, r.Error)
		}

		r := parseResp(resp)
		var invoices []invoiceHistoryEntry
		_ = json.Unmarshal(r.Data, &invoices)

		found := false
		for _, invoice := range invoices {
			if invoice.ID == invoiceID && invoice.Kebutuhan == kebutuhan {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("invoice id %d with kebutuhan %q not found in search results", invoiceID, kebutuhan)
		}
	})
}
