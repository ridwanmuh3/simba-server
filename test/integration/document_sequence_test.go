package integration_test

import (
	"context"
	"sync"
	"testing"

	"github.com/go-playground/validator/v10"

	"github.com/ridwanmuh3/simba-server/internal/entity"
	"github.com/ridwanmuh3/simba-server/internal/model"
	"github.com/ridwanmuh3/simba-server/internal/repository"
	"github.com/ridwanmuh3/simba-server/internal/service"
)

func newSeqServices(t *testing.T) (*service.SettingService, *service.InvoiceService) {
	t.Helper()
	validate := validator.New()
	settingSvc := service.NewSettingService(testDB, repository.NewSettingRepository(), testLog, validate)
	invoiceSvc := service.NewInvoiceService(testDB, testLog, validate, repository.NewInvoiceRepository())
	return settingSvc, invoiceSvc
}

func cleanupSeqState(t *testing.T) {
	t.Helper()
	testDB.Exec("DELETE FROM invoice_items WHERE invoice_id IN (SELECT id FROM invoices WHERE dapur_id = ?)", testDapurID)
	testDB.Exec("DELETE FROM invoices WHERE dapur_id = ?", testDapurID)
	testDB.Exec("DELETE FROM app_settings WHERE dapur_id = ? AND key IN ?", testDapurID, []string{"seq_invoice", "seq_po"})
}

func seqRequest() *model.GenerateInvoiceRequest {
	return &model.GenerateInvoiceRequest{
		CompanyName:     "PT Seq Test",
		CompanyAddress:  "Jl Seq Test",
		CompanyContact:  "0800123456",
		Date:            "2026-05-27",
		ReceiverName:    "Client Seq",
		ReceiverAddress: "Jl Client Seq",
		Penanggungjawab: "PIC Seq",
		Jabatan:         "Manager",
		StockType:       "OUT",
		DapurID:         testDapurID,
	}
}

func TestInvoiceSequence_ConcurrentSaveWithoutExplicitNumbers(t *testing.T) {
	cleanupSeqState(t)
	_, invoiceSvc := newSeqServices(t)
	ctx := context.Background()

	var wg sync.WaitGroup
	errs := make([]error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			errs[idx] = invoiceSvc.SaveInvoice(ctx, seqRequest(), nil)
		}(i)
	}
	wg.Wait()

	for i, err := range errs {
		if err != nil {
			t.Fatalf("SaveInvoice goroutine %d: %v", i, err)
		}
	}

	var numbers []string
	if err := testDB.Model(&entity.Invoice{}).
		Where("dapur_id = ?", testDapurID).
		Order("invoice_number ASC").
		Pluck("invoice_number", &numbers).Error; err != nil {
		t.Fatalf("query invoice numbers: %v", err)
	}
	if len(numbers) != 2 {
		t.Fatalf("invoice numbers = %v, want 2 rows", numbers)
	}
	got := map[string]bool{numbers[0]: true, numbers[1]: true}
	if !got["INV-001"] || !got["INV-002"] {
		t.Fatalf("invoice numbers = %v, want INV-001/INV-002", numbers)
	}

	var poNumbers []string
	if err := testDB.Model(&entity.Invoice{}).
		Where("dapur_id = ?", testDapurID).
		Order("po_number ASC").
		Pluck("po_number", &poNumbers).Error; err != nil {
		t.Fatalf("query po numbers: %v", err)
	}
	gotPo := map[string]bool{poNumbers[0]: true, poNumbers[1]: true}
	if !gotPo["PO-001"] || !gotPo["PO-002"] {
		t.Fatalf("po numbers = %v, want PO-001/PO-002", poNumbers)
	}

	assertSeq(t, "seq_invoice", "2")
	assertSeq(t, "seq_po", "2")

	cleanupSeqState(t)
}

func TestSettingGetNext_RepeatWithoutSaveSameNumber(t *testing.T) {
	cleanupSeqState(t)
	settingSvc, _ := newSeqServices(t)
	ctx := context.Background()

	first, err := settingSvc.GetNextDocumentNumbers(ctx, testDapurID)
	if err != nil {
		t.Fatalf("GetNext first: %v", err)
	}
	second, err := settingSvc.GetNextDocumentNumbers(ctx, testDapurID)
	if err != nil {
		t.Fatalf("GetNext second: %v", err)
	}
	if first.NextInvoiceNo != second.NextInvoiceNo || first.NextPONumber != second.NextPONumber {
		t.Fatalf("GetNext twice = %+v / %+v, want same number (no burn)", first, second)
	}
	if first.NextInvoiceNo != "INV-001" || first.NextPONumber != "PO-001" {
		t.Fatalf("GetNext = %+v, want INV-001/PO-001", first)
	}

	assertSeq(t, "seq_invoice", "0")
	assertSeq(t, "seq_po", "0")

	cleanupSeqState(t)
}

func assertSeq(t *testing.T, key, want string) {
	t.Helper()
	var setting entity.AppSetting
	if err := testDB.Where("key = ? AND dapur_id = ?", key, testDapurID).First(&setting).Error; err != nil {
		t.Fatalf("read %s: %v", key, err)
	}
	if setting.Value != want {
		t.Fatalf("%s = %q, want %q", key, setting.Value, want)
	}
}
