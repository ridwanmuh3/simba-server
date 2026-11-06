package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/ridwanmuh3/simba-server/internal/entity"
	"github.com/ridwanmuh3/simba-server/internal/exception"
	"github.com/ridwanmuh3/simba-server/internal/model"
	"github.com/ridwanmuh3/simba-server/internal/model/converter"
	"github.com/ridwanmuh3/simba-server/internal/repository"
	"github.com/ridwanmuh3/simba-server/internal/util"
)

func isDuplicateKeyError(err error) (bool, string) {
	msg := err.Error()
	if !strings.Contains(msg, "23505") && !strings.Contains(msg, "duplicate key") {
		return false, ""
	}
	switch {
	case strings.Contains(msg, "invoice_number"):
		return true, "nomor invoice sudah digunakan"
	case strings.Contains(msg, "po_number"):
		return true, "nomor PO sudah digunakan"
	default:
		return true, "nomor dokumen sudah digunakan"
	}
}

type InvoiceService struct {
	db          *gorm.DB
	log         *zap.SugaredLogger
	validate    *validator.Validate
	invoiceRepo InvoiceRepository
}

type InvoiceRepository interface {
	FindById(db *gorm.DB, id uint, dapurID uint) (*entity.Invoice, error)
	FindByIdWithItems(db *gorm.DB, id uint, dapurID uint) (*entity.Invoice, error)
	FindAll(db *gorm.DB, dapurID uint, searchQuery, startDate, endDate string, page, size int) ([]repository.InvoiceHistoryRow, int64, error)
	Save(db *gorm.DB, invoice *entity.Invoice) error
	Delete(db *gorm.DB, id uint, dapurID uint) error
	SaveItems(db *gorm.DB, items []entity.InvoiceItem) error
	DeleteItemsByInvoiceID(db *gorm.DB, invoiceID uint) error
	FindByIdForUpdate(db *gorm.DB, id uint, dapurID uint) (*entity.Invoice, error)
	Update(db *gorm.DB, invoice *entity.Invoice) error
	FindStocksForInvoice(db *gorm.DB, stockType string, stockIDs []uint, dateFrom, dateTo string, dapurID uint) ([]entity.StockTracking, error)
	FindStocksByIDs(db *gorm.DB, stockIDs []int, dapurID uint) ([]entity.StockTracking, error)
	FindItemsFlat(db *gorm.DB, dapurID uint, stockType, searchQuery, startDate, endDate string, page, size int) ([]repository.InvoiceItemFlatRow, int64, error)
}

var _ InvoiceRepository = (*repository.InvoiceRepository)(nil)

func NewInvoiceService(db *gorm.DB, logger *zap.SugaredLogger, validate *validator.Validate, invoiceRepo InvoiceRepository) *InvoiceService {
	return &InvoiceService{db: db, log: logger, validate: validate, invoiceRepo: invoiceRepo}
}

func (s *InvoiceService) GetInvoiceItems(ctx context.Context, request *model.GetInvoiceItemsRequest) (*model.InvoiceSummary, error) {
	db := s.db.WithContext(ctx)
	if err := s.validate.Struct(request); err != nil {
		s.log.Errorf("failed to validate request body: %v", err)
		return nil, err
	}
	stockType := request.StockType
	if stockType == "" {
		stockType = "OUT"
	}
	stocks, err := s.invoiceRepo.FindStocksForInvoice(db, stockType, request.StockIDs, request.DateFrom, request.DateTo, request.DapurID)
	if err != nil {
		s.log.Errorf("failed to get invoice items: %v", err)
		return nil, exception.InternalServerError
	}
	var grandTotal float64
	responses := make([]model.StockResponse, len(stocks))
	for i, stock := range stocks {
		mappedResponse := converter.StockToResponse(&stock)
		lineTotal := util.Round2(stock.Amount * stock.UnitPrice)
		mappedResponse.TotalPrice = lineTotal
		grandTotal = util.Round2(grandTotal + lineTotal)
		responses[i] = *mappedResponse
	}
	return &model.InvoiceSummary{Items: responses, GrandTotal: grandTotal}, nil
}

func (s *InvoiceService) SaveInvoice(ctx context.Context, request *model.GenerateInvoiceRequest, summary *model.InvoiceSummary) error {
	tx := s.db.WithContext(ctx).Begin()
	defer tx.Rollback()

	if err := repository.LockDocumentSequence(tx, request.DapurID); err != nil {
		s.log.Errorf("failed to acquire document sequence lock: %v", err)
		return exception.InternalServerError
	}

	invCur, hasInv := repository.ReadSeq(tx, KeySeqInvoice, request.DapurID)
	poCur, hasPo := repository.ReadSeq(tx, KeySeqPO, request.DapurID)

	var invBase, poBase int
	if hasInv {
		invBase = invCur
	}
	if hasPo {
		poBase = poCur
	}
	if !hasInv || !hasPo {
		maxInv, maxPo, err := repository.MaxDocumentSequence(tx, request.DapurID)
		if err != nil {
			return err
		}
		if !hasInv {
			invBase = maxInv
		}
		if !hasPo {
			poBase = maxPo
		}
	}

	invNo := strings.TrimSpace(request.InvoiceNo)
	poNo := strings.TrimSpace(request.PONo)
	invGenerated := false
	poGenerated := false
	if invNo == "" {
		invNo = fmt.Sprintf("INV-%03d", invBase+1)
		invGenerated = true
	}
	if poNo == "" {
		poNo = fmt.Sprintf("PO-%03d", poBase+1)
		poGenerated = true
	}

	invSeq, invOK := repository.ParseDocumentSequence(invNo, "INV")
	poSeq, poOK := repository.ParseDocumentSequence(poNo, "PO")
	invBump := invGenerated || (invOK && invSeq == invBase+1)
	poBump := poGenerated || (poOK && poSeq == poBase+1)

	invoice := entity.Invoice{
		DapurID:         request.DapurID,
		StockType:       request.StockType,
		CompanyName:     request.CompanyName,
		CompanyContact:  request.CompanyContact,
		CompanyAddress:  request.CompanyAddress,
		InvoiceNumber:   invNo,
		PONumber:        poNo,
		Kebutuhan:       request.Kebutuhan,
		ReceiverName:    request.ReceiverName,
		ReceiverAddress: request.ReceiverAddress,
		InvoiceDate:     request.Date,
		Keterangan:      request.Keterangan,
		Penanggungjawab: request.Penanggungjawab,
		Jabatan:         request.Jabatan,
		BankAccount:     request.BankAccount,
	}
	if summary != nil {
		invoice.GrandTotal = summary.GrandTotal
		invoice.Items = make([]entity.InvoiceItem, len(summary.Items))
		for i, it := range summary.Items {
			invoice.Items[i] = entity.InvoiceItem{
				ItemID:        it.Item.ID,
				ItemName:      it.Item.Name,
				Category:      it.Item.Category,
				MeasureUnit:   it.Item.MeasureUnit,
				Amount:        it.Amount,
				UnitPrice:     it.UnitPrice,
				TotalPrice:    it.TotalPrice,
				PreviousStock: it.PreviousStock,
				NewStock:      it.NewStock,
				Supplier:      it.Supplier,
				StockType:     it.Type,
				CreatedAt:     it.CreatedAt,
			}
		}
	}

	if err := s.invoiceRepo.Save(tx, &invoice); err != nil {
		if ok, msg := isDuplicateKeyError(err); ok {
			return exception.DuplicateDocumentError(msg)
		}
		s.log.Errorf("failed to save invoice record: %v", err)
		return exception.InternalServerError
	}
	if invBump {
		if err := repository.UpsertSeq(tx, KeySeqInvoice, request.DapurID, invSeq); err != nil {
			s.log.Errorf("failed to bump invoice seq: %v", err)
			return exception.InternalServerError
		}
	}
	if poBump {
		if err := repository.UpsertSeq(tx, KeySeqPO, request.DapurID, poSeq); err != nil {
			s.log.Errorf("failed to bump po seq: %v", err)
			return exception.InternalServerError
		}
	}
	if err := tx.Commit().Error; err != nil {
		if ok, msg := isDuplicateKeyError(err); ok {
			return exception.DuplicateDocumentError(msg)
		}
		s.log.Errorf("failed to commit invoice save: %v", err)
		return exception.InternalServerError
	}
	return nil
}

func (s *InvoiceService) FindInvoiceByID(ctx context.Context, id uint, dapurID uint) (*model.InvoiceData, error) {
	db := s.db.WithContext(ctx)
	invoice, err := s.invoiceRepo.FindByIdWithItems(db, id, dapurID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, exception.InvoiceNotFoundError
		}
		s.log.Errorf("failed to find invoice by id: %v", err)
		return nil, exception.InternalServerError
	}
	items := make([]model.StockResponse, len(invoice.Items))
	for i, it := range invoice.Items {
		items[i] = model.StockResponse{
			ID: int(it.ID), Type: it.StockType, Amount: it.Amount,
			PreviousStock: it.PreviousStock, NewStock: it.NewStock,
			UnitPrice: it.UnitPrice, TotalPrice: it.TotalPrice,
			Supplier: it.Supplier, CreatedAt: it.CreatedAt,
			Item: model.ItemResponse{
				ID: it.ItemID, Name: it.ItemName, Category: it.Category,
				MeasureUnit: it.MeasureUnit, UnitPrice: it.UnitPrice,
			},
		}
	}
	return &model.InvoiceData{
		StockType: invoice.StockType, CompanyName: invoice.CompanyName,
		CompanyAddress: invoice.CompanyAddress, CompanyContact: invoice.CompanyContact,
		InvoiceNo: invoice.InvoiceNumber, Date: invoice.InvoiceDate,
		PONo: invoice.PONumber, Kebutuhan: invoice.Kebutuhan,
		ReceiverName: invoice.ReceiverName, ReceiverAddress: invoice.ReceiverAddress,
		Items: items, GrandTotal: invoice.GrandTotal, Keterangan: invoice.Keterangan,
		Penanggungjawab: invoice.Penanggungjawab, Jabatan: invoice.Jabatan,
		BankAccount: invoice.BankAccount,
	}, nil
}

func (s *InvoiceService) FindInvoiceDetail(ctx context.Context, id uint, dapurID uint) (*model.InvoiceDetailResponse, error) {
	db := s.db.WithContext(ctx)
	invoice, err := s.invoiceRepo.FindByIdWithItems(db, id, dapurID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, exception.InvoiceNotFoundError
		}
		s.log.Errorf("failed to find invoice detail %d: %v", id, err)
		return nil, exception.InternalServerError
	}
	items := make([]model.InvoiceItemResponse, len(invoice.Items))
	for i, it := range invoice.Items {
		items[i] = model.InvoiceItemResponse{
			ID: it.ID, ItemName: it.ItemName, Category: it.Category,
			MeasureUnit: it.MeasureUnit, Amount: it.Amount, UnitPrice: it.UnitPrice,
			TotalPrice: it.TotalPrice, Supplier: it.Supplier, StockType: it.StockType,
		}
	}
	return &model.InvoiceDetailResponse{
		InvoiceResponse: model.InvoiceResponse{
			ID: invoice.ID, StockType: invoice.StockType, CompanyName: invoice.CompanyName,
			CompanyContact: invoice.CompanyContact, CompanyAddress: invoice.CompanyAddress,
			InvoiceNumber: invoice.InvoiceNumber, PONumber: invoice.PONumber,
			Kebutuhan: invoice.Kebutuhan, ReceiverName: invoice.ReceiverName,
			ReceiverAddress: invoice.ReceiverAddress, InvoiceDate: invoice.InvoiceDate,
			Keterangan: invoice.Keterangan, Penanggungjawab: invoice.Penanggungjawab,
			Jabatan: invoice.Jabatan, BankAccount: invoice.BankAccount,
			HasItems: len(invoice.Items) > 0, CreatedAt: invoice.CreatedAt, UpdatedAt: invoice.UpdatedAt,
		},
		Items: items, GrandTotal: invoice.GrandTotal,
	}, nil
}

func (s *InvoiceService) FindAllInvoices(ctx context.Context, request *model.FindAllInvoicesRequest) ([]model.InvoiceResponse, int64, error) {
	db := s.db.WithContext(ctx)
	page := request.Page
	if page < 1 {
		page = 1
	}
	size := request.Size
	if size < 1 {
		size = 10
	}
	rows, total, err := s.invoiceRepo.FindAll(db, request.DapurID, request.SearchQuery, request.StartDate, request.EndDate, page, size)
	if err != nil {
		s.log.Errorf("failed to get invoice records: %v", err)
		return nil, 0, exception.InternalServerError
	}
	responses := make([]model.InvoiceResponse, len(rows))
	for i, invoice := range rows {
		responses[i] = model.InvoiceResponse{
			ID: invoice.ID, StockType: invoice.StockType, CompanyName: invoice.CompanyName,
			CompanyContact: invoice.CompanyContact, CompanyAddress: invoice.CompanyAddress,
			InvoiceNumber: invoice.InvoiceNumber, PONumber: invoice.PONumber,
			Kebutuhan: invoice.Kebutuhan, ReceiverName: invoice.ReceiverName,
			ReceiverAddress: invoice.ReceiverAddress, InvoiceDate: invoice.InvoiceDate,
			Keterangan: invoice.Keterangan, Penanggungjawab: invoice.Penanggungjawab,
			Jabatan: invoice.Jabatan, BankAccount: invoice.BankAccount,
			HasItems: invoice.HasItems, CreatedAt: invoice.CreatedAt, UpdatedAt: invoice.UpdatedAt,
		}
	}
	return responses, total, nil
}

func (s *InvoiceService) UpdateInvoice(ctx context.Context, request *model.UpdateInvoiceRequest) error {
	tx := s.db.WithContext(ctx).Begin()
	defer tx.Rollback()

	invoice, err := s.invoiceRepo.FindByIdForUpdate(tx, request.ID, request.DapurID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return exception.InvoiceNotFoundError
		}
		s.log.Errorf("failed to find invoice %d: %v", request.ID, err)
		return exception.InternalServerError
	}

	invoice.CompanyName = request.CompanyName
	invoice.CompanyAddress = request.CompanyAddress
	invoice.CompanyContact = request.CompanyContact
	invoice.ReceiverName = request.ReceiverName
	invoice.ReceiverAddress = request.ReceiverAddress
	invoice.InvoiceDate = request.InvoiceDate
	invoice.Keterangan = request.Keterangan
	invoice.Penanggungjawab = request.Penanggungjawab
	invoice.Jabatan = request.Jabatan
	invoice.BankAccount = request.BankAccount

	if len(request.StockIDs) > 0 {
		stocks, err := s.invoiceRepo.FindStocksByIDs(tx, request.StockIDs, request.DapurID)
		if err != nil {
			s.log.Errorf("failed to fetch stocks for invoice update %d: %v", request.ID, err)
			return exception.InternalServerError
		}
		var grandTotal float64
		newItems := make([]entity.InvoiceItem, len(stocks))
		for i, stock := range stocks {
			lineTotal := util.Round2(stock.Amount * stock.UnitPrice)
			grandTotal = util.Round2(grandTotal + lineTotal)
			newItems[i] = entity.InvoiceItem{
				InvoiceID: invoice.ID, ItemID: stock.Item.ID, ItemName: stock.Item.Name,
				Category: stock.Item.Category, MeasureUnit: stock.Item.MeasureUnit,
				Amount: stock.Amount, UnitPrice: stock.UnitPrice, TotalPrice: lineTotal,
				PreviousStock: stock.PreviousStock, NewStock: stock.NewStock,
				Supplier: stock.Supplier, StockType: stock.Type, CreatedAt: stock.CreatedAt,
			}
		}
		invoice.GrandTotal = grandTotal
		if err := s.invoiceRepo.DeleteItemsByInvoiceID(tx, invoice.ID); err != nil {
			s.log.Errorf("failed to delete old invoice items %d: %v", invoice.ID, err)
			return exception.InternalServerError
		}
		if err := s.invoiceRepo.Update(tx, invoice); err != nil {
			s.log.Errorf("failed to update invoice %d: %v", invoice.ID, err)
			return exception.InternalServerError
		}
		if err := s.invoiceRepo.SaveItems(tx, newItems); err != nil {
			s.log.Errorf("failed to create new invoice items %d: %v", invoice.ID, err)
			return exception.InternalServerError
		}
		if err := tx.Commit().Error; err != nil {
			s.log.Errorf("failed to commit invoice update %d: %v", invoice.ID, err)
			return exception.InternalServerError
		}
		return nil
	}

	if err := s.invoiceRepo.Update(tx, invoice); err != nil {
		s.log.Errorf("failed to update invoice %d: %v", request.ID, err)
		return exception.InternalServerError
	}
	if err := tx.Commit().Error; err != nil {
		s.log.Errorf("failed to commit invoice update %d: %v", request.ID, err)
		return exception.InternalServerError
	}
	return nil
}

func (s *InvoiceService) GetInvoiceItemsFlat(ctx context.Context, req *model.GetInvoiceItemsFlatRequest) ([]model.InvoiceItemFlatResponse, int64, error) {
	db := s.db.WithContext(ctx)
	page := req.Page
	if page < 1 {
		page = 1
	}
	size := req.Size
	if size < 1 {
		size = 10
	}
	rows, total, err := s.invoiceRepo.FindItemsFlat(db, req.DapurID, req.StockType, req.SearchQuery, req.StartDate, req.EndDate, page, size)
	if err != nil {
		s.log.Errorf("failed to fetch invoice items flat: %v", err)
		return nil, 0, exception.InternalServerError
	}
	responses := make([]model.InvoiceItemFlatResponse, len(rows))
	for i, r := range rows {
		date := ""
		if !r.Date.IsZero() {
			date = r.Date.Format("2006-01-02")
		}
		responses[i] = model.InvoiceItemFlatResponse{
			Date: date, InvoiceNumber: r.InvoiceNumber, ItemName: r.ItemName,
			MeasureUnit: r.MeasureUnit, Amount: r.Amount, StockType: r.StockType,
			BuyPrice: r.BuyPrice, SellPrice: r.SellPrice,
			TotalBuyPrice: r.TotalBuyPrice, TotalSellPrice: r.TotalSellPrice,
		}
	}
	return responses, total, nil
}

func (s *InvoiceService) DeleteInvoice(ctx context.Context, id uint, dapurID uint) error {
	tx := s.db.WithContext(ctx).Begin()
	defer tx.Rollback()

	if _, err := s.invoiceRepo.FindByIdForUpdate(tx, id, dapurID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return exception.InvoiceNotFoundError
		}
		s.log.Errorf("failed to find invoice %d: %v", id, err)
		return exception.InternalServerError
	}
	if err := s.invoiceRepo.DeleteItemsByInvoiceID(tx, id); err != nil {
		s.log.Errorf("failed to delete invoice items for invoice %d: %v", id, err)
		return exception.InternalServerError
	}
	if err := s.invoiceRepo.Delete(tx, id, dapurID); err != nil {
		s.log.Errorf("failed to delete invoice %d: %v", id, err)
		return exception.InternalServerError
	}
	if err := tx.Commit().Error; err != nil {
		s.log.Errorf("failed to commit invoice delete %d: %v", id, err)
		return exception.InternalServerError
	}
	return nil
}
