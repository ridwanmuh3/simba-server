package service

import (
	"context"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/ridwanmuh3/simba-server/internal/entity"
	"github.com/ridwanmuh3/simba-server/internal/exception"
	"github.com/ridwanmuh3/simba-server/internal/model"
	"github.com/ridwanmuh3/simba-server/internal/model/converter"
	"github.com/ridwanmuh3/simba-server/internal/util"
)

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

func (s *InvoiceService) SaveInvoice(ctx context.Context, request *model.GenerateInvoiceRequest) error {
	db := s.db.WithContext(ctx)

	values := map[string]any{
		"company_name":    request.CompanyName,
		"company_contact": request.CompanyContact,
		"company_address": request.CompanyAddress,
		"invoice_number":  request.InvoiceNo,
		"po_number":       request.PONo,
		"quo_number":      request.QuoNo,
	}

	if db.Migrator().HasColumn(&entity.Invoice{}, "stock_type") {
		values["stock_type"] = request.StockType
	}

	if err := db.Table("invoices").Create(values).Error; err != nil {
		s.log.Errorf("failed to save invoice record: %v", err)
		return exception.InternalServerError
	}

	return nil
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
		CreatedAt      time.Time
		UpdatedAt      time.Time
	}

	selectColumns := []string{
		"id",
		"company_name",
		"company_contact",
		"company_address",
		"invoice_number",
		"po_number",
		"quo_number",
		"created_at",
		"updated_at",
	}
	if db.Migrator().HasColumn(&entity.Invoice{}, "stock_type") {
		selectColumns = append([]string{"stock_type"}, selectColumns...)
	}

	var invoices []invoiceHistoryRow
	if err := query.
		Select(strings.Join(selectColumns, ", ")).
		Order("created_at DESC").
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
			CreatedAt:      invoice.CreatedAt,
			UpdatedAt:      invoice.UpdatedAt,
		}
	}

	return responses, total, nil
}
