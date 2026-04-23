package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/ridwanmuh3/simba-server/internal/entity"
	"github.com/ridwanmuh3/simba-server/internal/exception"
	"github.com/ridwanmuh3/simba-server/internal/model"
	"github.com/ridwanmuh3/simba-server/internal/model/converter"
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
	case strings.Contains(msg, "quo_number"):
		return true, "nomor quotation sudah digunakan"
	default:
		return true, "nomor dokumen sudah digunakan"
	}
}

type InvoiceService struct {
	db       *gorm.DB
	log      *zap.SugaredLogger
	validate *validator.Validate
}

func NewInvoiceService(db *gorm.DB, logger *zap.SugaredLogger, validate *validator.Validate) *InvoiceService {
	return &InvoiceService{
		db:       db,
		log:      logger,
		validate: validate,
	}
}

func (s *InvoiceService) GetInvoiceItems(ctx context.Context, request *model.GetInvoiceItemsRequest) (*model.InvoiceSummary, error) {
	db := s.db.WithContext(ctx)

	if err := s.validate.Struct(request); err != nil {
		s.log.Errorf("failed to validate request body: %v", err)
		return nil, err
	}

	var stocks []entity.StockTracking
	stockType := request.StockType
	if stockType == "" {
		stockType = "OUT"
	}
	query := db.Model(new(entity.StockTracking)).Preload("Item").Where("type = ?", stockType)

	if len(request.StockIDs) > 0 {
		query = query.Where("id IN ?", request.StockIDs)
	} else {
		if request.DateFrom != "" {
			parsedFrom, err := time.Parse(time.RFC3339Nano, request.DateFrom)
			if err != nil {
				s.log.Errorf("failed to parse date from: %v", err)
				return nil, exception.InternalServerError
			}
			query = query.Where("created_at >= ?", parsedFrom.UTC())
		}

		if request.DateTo != "" {
			parsedTo, err := time.Parse(time.RFC3339Nano, request.DateTo)
			if err != nil {
				s.log.Errorf("failed to parse date to: %v", err)
				return nil, exception.InternalServerError
			}
			query = query.Where("created_at <= ?", parsedTo.UTC())
		}
	}

	if err := query.Order("item_id ASC").Limit(500).Find(&stocks).Error; err != nil {
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

	return &model.InvoiceSummary{
		Items:      responses,
		GrandTotal: grandTotal,
	}, nil
}

func (s *InvoiceService) SaveInvoice(ctx context.Context, request *model.GenerateInvoiceRequest, summary *model.InvoiceSummary) error {
	db := s.db.WithContext(ctx)

	invoice := entity.Invoice{
		StockType:       request.StockType,
		CompanyName:     request.CompanyName,
		CompanyContact:  request.CompanyContact,
		CompanyAddress:  request.CompanyAddress,
		InvoiceNumber:   request.InvoiceNo,
		PONumber:        request.PONo,
		QuoNumber:       request.QuoNo,
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

	if err := db.Create(&invoice).Error; err != nil {
		if ok, msg := isDuplicateKeyError(err); ok {
			return fiber.NewError(fiber.StatusConflict, msg)
		}
		s.log.Errorf("failed to save invoice record: %v", err)
		return exception.InternalServerError
	}

	return nil
}

func (s *InvoiceService) FindInvoiceByID(ctx context.Context, id uint) (*model.InvoiceData, error) {
	db := s.db.WithContext(ctx)

	var invoice entity.Invoice
	if err := db.Preload("Items").First(&invoice, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fiber.NewError(fiber.StatusNotFound, "invoice tidak ditemukan")
		}
		s.log.Errorf("failed to find invoice by id: %v", err)
		return nil, exception.InternalServerError
	}

	items := make([]model.StockResponse, len(invoice.Items))
	for i, it := range invoice.Items {
		items[i] = model.StockResponse{
			ID:            int(it.ID),
			Type:          it.StockType,
			Amount:        it.Amount,
			PreviousStock: it.PreviousStock,
			NewStock:      it.NewStock,
			UnitPrice:     it.UnitPrice,
			TotalPrice:    it.TotalPrice,
			Supplier:      it.Supplier,
			CreatedAt:     it.CreatedAt,
			Item: model.ItemResponse{
				ID:          it.ItemID,
				Name:        it.ItemName,
				Category:    it.Category,
				MeasureUnit: it.MeasureUnit,
				UnitPrice:   it.UnitPrice,
			},
		}
	}

	return &model.InvoiceData{
		StockType:       invoice.StockType,
		CompanyName:     invoice.CompanyName,
		CompanyAddress:  invoice.CompanyAddress,
		CompanyContact:  invoice.CompanyContact,
		InvoiceNo:       invoice.InvoiceNumber,
		Date:            invoice.InvoiceDate,
		PONo:            invoice.PONumber,
		QuoNo:           invoice.QuoNumber,
		ReceiverName:    invoice.ReceiverName,
		ReceiverAddress: invoice.ReceiverAddress,
		Items:           items,
		GrandTotal:      invoice.GrandTotal,
		Keterangan:      invoice.Keterangan,
		Penanggungjawab: invoice.Penanggungjawab,
		Jabatan:         invoice.Jabatan,
		BankAccount:     invoice.BankAccount,
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

	query := db.Table("invoices")

	if request.SearchQuery != "" {
		search := "%" + request.SearchQuery + "%"
		query = query.Where(
			"invoice_number LIKE ? OR company_name LIKE ? OR company_contact LIKE ? OR po_number LIKE ? OR quo_number LIKE ?",
			search, search, search, search, search,
		)
	}

	if request.StartDate != "" {
		parsedFrom, err := time.Parse(time.RFC3339Nano, request.StartDate)
		if err != nil {
			s.log.Errorf("failed to parse invoice start date: %v", err)
			return nil, 0, exception.InternalServerError
		}
		query = query.Where("created_at >= ?", parsedFrom.UTC())
	}

	if request.EndDate != "" {
		parsedTo, err := time.Parse(time.RFC3339Nano, request.EndDate)
		if err != nil {
			s.log.Errorf("failed to parse invoice end date: %v", err)
			return nil, 0, exception.InternalServerError
		}
		query = query.Where("created_at <= ?", parsedTo.UTC())
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		s.log.Errorf("failed to count invoice records: %v", err)
		return nil, 0, exception.InternalServerError
	}

	type invoiceHistoryRow struct {
		ID             uint
		StockType      string
		CompanyName    string
		CompanyContact string
		CompanyAddress string
		InvoiceNumber  string
		PONumber       string
		QuoNumber      string
		HasItems       bool
		CreatedAt      time.Time
		UpdatedAt      time.Time
	}

	selectColumns := []string{
		"invoices.id",
		"invoices.company_name",
		"invoices.company_contact",
		"invoices.company_address",
		"invoices.invoice_number",
		"invoices.po_number",
		"invoices.quo_number",
		"invoices.created_at",
		"invoices.updated_at",
		"EXISTS (SELECT 1 FROM invoice_items WHERE invoice_items.invoice_id = invoices.id) AS has_items",
	}
	if db.Migrator().HasColumn(&entity.Invoice{}, "stock_type") {
		selectColumns = append([]string{"invoices.stock_type"}, selectColumns...)
	}

	var invoices []invoiceHistoryRow
	if err := query.
		Select(strings.Join(selectColumns, ", ")).
		Order("invoices.created_at DESC").
		Offset((page - 1) * size).
		Limit(size).
		Scan(&invoices).Error; err != nil {
		s.log.Errorf("failed to get invoice records: %v", err)
		return nil, 0, exception.InternalServerError
	}

	responses := make([]model.InvoiceResponse, len(invoices))
	for i, invoice := range invoices {
		responses[i] = model.InvoiceResponse{
			ID:             invoice.ID,
			StockType:      invoice.StockType,
			CompanyName:    invoice.CompanyName,
			CompanyContact: invoice.CompanyContact,
			CompanyAddress: invoice.CompanyAddress,
			InvoiceNumber:  invoice.InvoiceNumber,
			PONumber:       invoice.PONumber,
			QuoNumber:      invoice.QuoNumber,
			HasItems:       invoice.HasItems,
			CreatedAt:      invoice.CreatedAt,
			UpdatedAt:      invoice.UpdatedAt,
		}
	}

	return responses, total, nil
}

func (s *InvoiceService) DeleteInvoice(ctx context.Context, id uint) error {
	db := s.db.WithContext(ctx)

	var count int64
	if err := db.Model(&entity.Invoice{}).Where("id = ?", id).Count(&count).Error; err != nil {
		s.log.Errorf("failed to check invoice %d: %v", id, err)
		return exception.InternalServerError
	}
	if count == 0 {
		return fiber.NewError(fiber.StatusNotFound, "invoice tidak ditemukan")
	}

	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("invoice_id = ?", id).Delete(&entity.InvoiceItem{}).Error; err != nil {
			s.log.Errorf("failed to delete invoice items for invoice %d: %v", id, err)
			return exception.InternalServerError
		}
		if err := tx.Delete(&entity.Invoice{}, id).Error; err != nil {
			s.log.Errorf("failed to delete invoice %d: %v", id, err)
			return exception.InternalServerError
		}
		return nil
	})
}
