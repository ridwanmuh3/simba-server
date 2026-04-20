package handler

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"math"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/spf13/viper"
	"go.uber.org/zap"

	"github.com/ridwanmuh3/simba-server/internal/delivery/middleware"
	"github.com/ridwanmuh3/simba-server/internal/exception"
	"github.com/ridwanmuh3/simba-server/internal/model"
	"github.com/ridwanmuh3/simba-server/internal/service"
	"github.com/ridwanmuh3/simba-server/internal/util"
)

type ItemHandler struct {
	config         *viper.Viper
	log            *zap.SugaredLogger
	validate       *validator.Validate
	itemService    ItemService
	stockService   StockService
	invoiceService InvoiceService
	settingService SettingService
}

type ItemService interface {
	Add(ctx context.Context, request *model.AddItemRequest) (*model.ItemResponse, error)
	AddBatches(ctx context.Context, request *model.AddItemBatchRequest) error
	ExportItems(ctx context.Context) ([]model.ItemResponse, int, error)
	Update(ctx context.Context, request *model.UpdateItemRequest) (*model.ItemResponse, error)
	Delete(ctx context.Context, request *model.DeleteItemRequest) (bool, error)
	FindById(ctx context.Context, request *model.FindByIdItemRequest) (*model.ItemResponse, error)
	FindAll(ctx context.Context, request *model.FindAllItemsRequest) ([]model.ItemResponse, int64, error)
}

type StockService interface {
	UpdateStock(ctx context.Context, request *model.UpdateItemStockRequest) (*model.StockResponse, error)
	EditStock(ctx context.Context, request *model.EditStockRequest) (*model.StockResponse, error)
	DeleteStock(ctx context.Context, request *model.DeleteStockRequest) (bool, error)
	FindAllStocks(ctx context.Context, request *model.FindAllStocksRequest) ([]model.StockResponse, int64, error)
	GetStocksFinanceSummary(ctx context.Context) (*model.StocksFinanceSummaryResponse, error)
	GetItemStocksSummary(ctx context.Context, request *model.GetItemStockSummaryRequest) ([]model.ItemStocksSummaryResponse, int64, error)
}

type InvoiceService interface {
	GetInvoiceItems(ctx context.Context, request *model.GetInvoiceItemsRequest) (*model.InvoiceSummary, error)
	SaveInvoice(ctx context.Context, request *model.GenerateInvoiceRequest, summary *model.InvoiceSummary) error
	FindAllInvoices(ctx context.Context, request *model.FindAllInvoicesRequest) ([]model.InvoiceResponse, int64, error)
	FindInvoiceByID(ctx context.Context, id uint) (*model.InvoiceData, error)
	DeleteInvoice(ctx context.Context, id uint) error
}

type SettingService interface {
	GetCompanyProfile(ctx context.Context) (*model.CompanyProfileResponse, error)
	GetNextDocumentNumbers(ctx context.Context) (*model.DocumentSequenceResponse, error)
	ConsumeDocumentNumbers(ctx context.Context) error
}

var (
	_ ItemService    = (*service.ItemService)(nil)
	_ StockService   = (*service.StockService)(nil)
	_ InvoiceService = (*service.InvoiceService)(nil)
	_ SettingService = (*service.SettingService)(nil)
)

func NewItemHandler(
	config *viper.Viper,
	logger *zap.SugaredLogger,
	validate *validator.Validate,
	itemService ItemService,
	stockService StockService,
	invoiceService InvoiceService,
	settingService SettingService,
) *ItemHandler {
	return &ItemHandler{
		config:         config,
		log:            logger,
		validate:       validate,
		itemService:    itemService,
		stockService:   stockService,
		invoiceService: invoiceService,
		settingService: settingService,
	}
}

func (h *ItemHandler) Add(c *fiber.Ctx) error {
	auth := middleware.GetAuthUser(c)

	request := new(model.AddItemRequest)
	if err := c.BodyParser(request); err != nil {
		h.log.Warnf("failed to parse request body: %v", err)
		return err
	}

	request.ModifiedBy = auth.Fullname

	response, err := h.itemService.Add(c.Context(), request)
	if err != nil {
		h.log.Warnf("failed to add item: %v", err)
		return err
	}

	return c.Status(fiber.StatusCreated).JSON(model.Response[*model.ItemResponse]{
		Status:  fiber.StatusCreated,
		Message: "add new item success",
		Data:    response,
	})
}

func (h *ItemHandler) ImportItems(c *fiber.Ctx) error {
	auth := middleware.GetAuthUser(c)

	fileHeader, err := c.FormFile("import_file")
	if err != nil {
		h.log.Warnf("failed to get uploaded file: %v", err)
		return exception.InvalidUploadedFileError
	}

	ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
	if ext != ".csv" {
		h.log.Warnf("invalid uploaded file format")
		return exception.InvalidCsvFormatError
	}

	fileSize := fileHeader.Size
	maxFileSize := int64(10 * 1024 * 1024)
	if fileSize > maxFileSize {
		h.log.Warnf("failed to process file size: %vMB", fileSize)
		return exception.ExceedMaximumFileSizeError
	}

	src, err := fileHeader.Open()
	if err != nil {
		h.log.Warnf("failed to open file: %v", err)
		return exception.InternalServerError
	}
	defer src.Close()

	reader := csv.NewReader(src)
	reader.FieldsPerRecord = 5

	var items []model.AddItemRequest
	lineCount := 0

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}

		lineCount++

		if err != nil {
			if err == csv.ErrFieldCount {
				h.log.Warnf("CSV line %d: incorrect column count", lineCount)
				return exception.InvalidCsvFormatError
			}
			h.log.Warnf("error reading csv line %d: %v", lineCount, err)
			return exception.InvalidCsvFormatError
		}

		// Skip Header
		if lineCount == 1 {
			continue
		}

		if strings.TrimSpace(record[0]) == "" {
			h.log.Warnf("CSV line %d: name is empty", lineCount)
			return fiber.NewError(fiber.StatusBadRequest, fmt.Sprintf("Baris %d: Nama barang tidak boleh kosong", lineCount))
		}

		stock, err := strconv.ParseFloat(record[2], 64)
		if err != nil || stock < 0 {
			h.log.Warnf("CSV line %d: invalid stock format '%s'", lineCount, record[2])
			return fiber.NewError(fiber.StatusBadRequest, fmt.Sprintf("Baris %d: Format stok salah (harus angka)", lineCount))
		}

		price, err := strconv.ParseFloat(record[4], 64)
		if err != nil || price < 0 {
			h.log.Warnf("CSV line %d: invalid price format '%s'", lineCount, record[4])
			return fiber.NewError(fiber.StatusBadRequest, fmt.Sprintf("Baris %d: Format harga salah (harus angka)", lineCount))
		}

		item := model.AddItemRequest{
			Name:        strings.TrimSpace(record[0]),
			Category:    strings.TrimSpace(record[1]),
			Stock:       stock,
			MeasureUnit: strings.TrimSpace(record[3]),
			UnitPrice:   price,
			ModifiedBy:  auth.Fullname,
		}

		items = append(items, item)
	}

	if len(items) == 0 {
		return fiber.NewError(fiber.StatusBadRequest, "File CSV tidak berisi data barang")
	}

	const maxCSVRows = 500
	if len(items) > maxCSVRows {
		return fiber.NewError(fiber.StatusBadRequest, fmt.Sprintf("Jumlah baris melebihi batas maksimum %d baris", maxCSVRows))
	}

	err = h.itemService.AddBatches(c.Context(), &model.AddItemBatchRequest{Items: items})
	if err != nil {
		h.log.Warnf("failed to bulk insert items: %v", err)
		return exception.InternalServerError
	}

	return c.JSON(model.Response[int]{
		Status:  fiber.StatusCreated,
		Message: "items imported successfully",
		Data:    len(items),
	})
}

func (h *ItemHandler) ExportItems(c *fiber.Ctx) error {
	items, _, err := h.itemService.ExportItems(c.Context())
	if err != nil {
		h.log.Warnf("failed to find all items: %v", err)
		return err
	}

	return c.JSON(model.Response[[]model.ItemResponse]{
		Status:  fiber.StatusOK,
		Message: "export items success",
		Data:    items,
	})
}

func (h *ItemHandler) Update(c *fiber.Ctx) error {
	auth := middleware.GetAuthUser(c)

	request := new(model.UpdateItemRequest)
	if err := c.ParamsParser(request); err != nil {
		h.log.Warnf("failed to parse request params: %v", err)
		return err
	}

	if err := c.BodyParser(request); err != nil {
		h.log.Warnf("failed to parse request body: %v", err)
		return err
	}

	request.ModifiedBy = auth.Fullname

	response, err := h.itemService.Update(c.Context(), request)
	if err != nil {
		h.log.Warnf("failed to update item: %v", err)
		return err
	}

	return c.JSON(model.Response[*model.ItemResponse]{
		Status:  fiber.StatusOK,
		Message: "update item success",
		Data:    response,
	})
}

func (h *ItemHandler) UpdateStock(c *fiber.Ctx) error {
	auth := middleware.GetAuthUser(c)

	request := new(model.UpdateItemStockRequest)
	if err := c.ParamsParser(request); err != nil {
		h.log.Warnf("failed to parse request params: %v", err)
		return err
	}

	if err := c.BodyParser(request); err != nil {
		h.log.Warnf("failed to parse request body: %v", err)
		return err
	}

	request.ModifiedBy = auth.Fullname

	response, err := h.stockService.UpdateStock(c.Context(), request)
	if err != nil {
		h.log.Warnf("failed to update item stock: %v", err)
		return err
	}

	return c.JSON(model.Response[*model.StockResponse]{
		Status:  fiber.StatusOK,
		Message: "update stock item success",
		Data:    response,
	})
}

func (h *ItemHandler) EditStock(c *fiber.Ctx) error {
	auth := middleware.GetAuthUser(c)

	stockIdStr := c.Params("stock_id")
	stockIdInt, err := strconv.Atoi(stockIdStr)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "stock_id must be a number")
	}

	request := new(model.EditStockRequest)
	if err := c.ParamsParser(request); err != nil {
		h.log.Warnf("failed to parse request params: %v", err)
		return err
	}
	if err := c.BodyParser(request); err != nil {
		h.log.Warnf("failed to parse request body: %v", err)
		return err
	}

	request.StockID = stockIdInt
	request.ModifiedBy = auth.Fullname

	response, err := h.stockService.EditStock(c.Context(), request)
	if err != nil {
		h.log.Warnf("failed to edit stock item: %v", err)
		return err
	}

	return c.JSON(model.Response[*model.StockResponse]{
		Status:  fiber.StatusOK,
		Message: "edit stock item success",
		Data:    response,
	})
}

func (h *ItemHandler) Delete(c *fiber.Ctx) error {
	request := new(model.DeleteItemRequest)
	if err := c.ParamsParser(request); err != nil {
		h.log.Warnf("failed to parse request body: %v", err)
		return err
	}

	response, err := h.itemService.Delete(c.Context(), request)
	if err != nil {
		h.log.Warnf("failed to delete item: %v", err)
		return err
	}

	return c.JSON(model.Response[bool]{
		Status:  fiber.StatusOK,
		Message: "delete item success",
		Data:    response,
	})
}

func (h *ItemHandler) DeleteStock(c *fiber.Ctx) error {
	auth := middleware.GetAuthUser(c)
	id := c.Params("id")
	stockIdStr := c.Params("stock_id")

	stockIdInt, err := strconv.Atoi(stockIdStr)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "stock_id must be a number")
	}

	request := &model.DeleteStockRequest{
		ID:         id,
		StockID:    stockIdInt,
		ModifiedBy: auth.Fullname,
	}

	response, err := h.stockService.DeleteStock(c.Context(), request)
	if err != nil {
		h.log.Warnf("failed to delete stock item: %v", err)
		return err
	}

	return c.JSON(model.Response[bool]{
		Status:  fiber.StatusOK,
		Message: "delete stock item success",
		Data:    response,
	})
}

func (h *ItemHandler) FindById(c *fiber.Ctx) error {
	request := new(model.FindByIdItemRequest)
	if err := c.ParamsParser(request); err != nil {
		h.log.Warnf("failed to parse request body: %v", err)
		return err
	}

	response, err := h.itemService.FindById(c.Context(), request)
	if err != nil {
		h.log.Warnf("failed to find item by code: %v", err)
		return err
	}

	return c.JSON(model.Response[*model.ItemResponse]{
		Status:  fiber.StatusOK,
		Message: "find item by code success",
		Data:    response,
	})
}

func (h *ItemHandler) FindAll(c *fiber.Ctx) error {
	request := &model.FindAllItemsRequest{
		SearchQuery: c.Query("search_query", ""),
		StartDate:   c.Query("start_date", ""),
		EndDate:     c.Query("end_date", ""),
		Page:        c.QueryInt("page", 1),
		Size:        c.QueryInt("size", 10),
	}

	response, total, err := h.itemService.FindAll(c.Context(), request)
	if err != nil {
		h.log.Warnf("failed to find all items: %v", err)
		return err
	}

	pagingMetadata := &model.PageMetadata{
		Page:      request.Page,
		Size:      request.Size,
		TotalItem: total,
		TotalPage: int64(math.Ceil(float64(total) / float64(request.Size))),
	}

	return c.JSON(model.Response[[]model.ItemResponse]{
		Status:  fiber.StatusOK,
		Message: "find all items success",
		Data:    response,
		Paging:  pagingMetadata,
	})
}

func (h *ItemHandler) FindAllStocks(c *fiber.Ctx) error {
	request := &model.FindAllStocksRequest{
		SearchQuery: c.Query("search_query", ""),
		StartDate:   c.Query("start_date", ""),
		EndDate:     c.Query("end_date", ""),
		Page:        c.QueryInt("page", 1),
		Size:        c.QueryInt("size", 10),
		Type:        c.Query("type", "ALL"),
	}

	response, total, err := h.stockService.FindAllStocks(c.Context(), request)
	if err != nil {
		h.log.Warnf("failed to find all items stocks: %v", err)
		return err
	}

	pagingMetadata := &model.PageMetadata{
		Page:      request.Page,
		Size:      request.Size,
		TotalItem: total,
		TotalPage: int64(math.Ceil(float64(total) / float64(request.Size))),
	}

	return c.JSON(model.Response[[]model.StockResponse]{
		Status:  fiber.StatusOK,
		Message: "find all items stocks success",
		Data:    response,
		Paging:  pagingMetadata,
	})
}

func (h *ItemHandler) GetStocksFinanceSummary(c *fiber.Ctx) error {
	response, err := h.stockService.GetStocksFinanceSummary(c.Context())
	if err != nil {
		h.log.Warnf("failed to get stocks finance summary: %v", err)
		return err
	}

	return c.JSON(model.Response[*model.StocksFinanceSummaryResponse]{
		Status:  fiber.StatusOK,
		Message: "get stocks finance summary success",
		Data:    response,
	})
}

func (h *ItemHandler) GetItemStocksSummary(c *fiber.Ctx) error {
	request := &model.GetItemStockSummaryRequest{
		StartDate:   c.Query("start_date", ""),
		EndDate:     c.Query("end_date", ""),
		Page:        c.QueryInt("page", 1),
		Size:        c.QueryInt("size", 10),
		SearchQuery: c.Query("search_query"),
	}

	response, total, err := h.stockService.GetItemStocksSummary(c.Context(), request)
	if err != nil {
		h.log.Warnf("failed to get item stock summary: %v", err)
		return err
	}

	pagingMetadata := &model.PageMetadata{
		Page:      request.Page,
		Size:      request.Size,
		TotalItem: total,
		TotalPage: int64(math.Ceil(float64(total) / float64(request.Size))),
	}

	return c.JSON(model.Response[[]model.ItemStocksSummaryResponse]{
		Status:  fiber.StatusOK,
		Message: "get item stock summary success",
		Data:    response,
		Paging:  pagingMetadata,
	})
}
func (h *ItemHandler) GetInvoiceItems(c *fiber.Ctx) error {
	request := new(model.GenerateInvoiceRequest)
	if err := c.BodyParser(request); err != nil {
		h.log.Warnf("failed to parse invoice request body: %v", err)
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	if request.CompanyName == "" || request.CompanyAddress == "" || request.CompanyContact == "" {
		profile, err := h.settingService.GetCompanyProfile(c.Context())
		if err != nil {
			h.log.Warnf("failed to get company profile: %v", err)
			return err
		}
		if request.CompanyName == "" {
			request.CompanyName = profile.CompanyName
		}
		if request.CompanyAddress == "" {
			request.CompanyAddress = profile.CompanyAddress
		}
		if request.CompanyContact == "" {
			request.CompanyContact = profile.CompanyContact
		}
	}

	if request.InvoiceNo == "" || request.QuoNo == "" {
		numbers, err := h.settingService.GetNextDocumentNumbers(c.Context())
		if err != nil {
			h.log.Warnf("failed to get next document numbers: %v", err)
			return err
		}
		if request.InvoiceNo == "" {
			request.InvoiceNo = numbers.NextInvoiceNo
		}
		if request.QuoNo == "" {
			request.QuoNo = numbers.NextQuotationNo
		}
	}

	if err := h.validate.Struct(request); err != nil {
		h.log.Warnf("failed to validate invoice request: %v", err)
		return fiber.NewError(fiber.StatusBadRequest, "validation failed")
	}

	stockType := strings.ToUpper(strings.TrimSpace(request.StockType))
	if stockType == "" {
		stockType = "OUT"
	}
	request.StockType = stockType

	invoiceItemsReq := &model.GetInvoiceItemsRequest{
		DateFrom:  request.DateFrom,
		DateTo:    request.DateTo,
		StockType: stockType,
	}

	summary, err := h.invoiceService.GetInvoiceItems(c.Context(), invoiceItemsReq)
	if err != nil {
		h.log.Warnf("failed to get invoice items: %v", err)
		return err
	}

	if len(summary.Items) == 0 {
		label := "keluar"
		if stockType == "IN" {
			label = "masuk"
		}
		return fiber.NewError(fiber.StatusNotFound, fmt.Sprintf("tidak ada data bahan %s pada rentang tanggal yang dipilih", label))
	}

	if err := h.invoiceService.SaveInvoice(c.Context(), request, summary); err != nil {
		h.log.Warnf("failed to save invoice record: %v", err)
		return err
	}

	if err := h.settingService.ConsumeDocumentNumbers(c.Context()); err != nil {
		h.log.Warnf("failed to consume document numbers: %v", err)
		return err
	}

	invoiceDate := util.FormatDateStringID(request.Date)

	if strings.TrimSpace(request.BankAccount) == "" {
		request.BankAccount = service.DefaultBankAccount
	}

	pdfData := &model.InvoiceData{
		StockType:       stockType,
		CompanyName:     request.CompanyName,
		CompanyAddress:  request.CompanyAddress,
		CompanyContact:  request.CompanyContact,
		InvoiceNo:       request.InvoiceNo,
		Date:            invoiceDate,
		PONo:            request.PONo,
		QuoNo:           request.QuoNo,
		ReceiverName:    request.ReceiverName,
		ReceiverAddress: request.ReceiverAddress,
		Items:           summary.Items,
		GrandTotal:      summary.GrandTotal,
		Keterangan:      request.Keterangan,
		Penanggungjawab: request.Penanggungjawab,
		Jabatan:         request.Jabatan,
		BankAccount:     request.BankAccount,
	}

	pdfBuffer, err := util.GenerateTemplateInvoicePDF(pdfData)
	if err != nil {
		h.log.Warnf("failed to generate invoice PDF: %v", err)
		return err
	}

	// Sanitize filename to prevent header injection
	safeInvoiceNo := strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			return r
		}
		return '_'
	}, request.InvoiceNo)
	filename := url.PathEscape(fmt.Sprintf("invoice-%s.pdf", safeInvoiceNo))

	c.Set("Content-Type", "application/pdf")
	c.Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	return c.Send(pdfBuffer.Bytes())
}

func (h *ItemHandler) GetInvoiceHistory(c *fiber.Ctx) error {
	request := &model.FindAllInvoicesRequest{
		SearchQuery: c.Query("search_query", ""),
		StartDate:   c.Query("start_date", ""),
		EndDate:     c.Query("end_date", ""),
		Page:        c.QueryInt("page", 1),
		Size:        c.QueryInt("size", 10),
	}

	response, total, err := h.invoiceService.FindAllInvoices(c.Context(), request)
	if err != nil {
		h.log.Warnf("failed to get invoice history: %v", err)
		return err
	}

	pageSize := request.Size
	if pageSize < 1 {
		pageSize = 10
	}

	page := request.Page
	if page < 1 {
		page = 1
	}

	totalPage := int64(0)
	if total > 0 {
		totalPage = int64(math.Ceil(float64(total) / float64(pageSize)))
	}

	return c.JSON(model.Response[[]model.InvoiceResponse]{
		Status:  fiber.StatusOK,
		Message: "get invoice history success",
		Data:    response,
		Paging: &model.PageMetadata{
			Page:      page,
			Size:      pageSize,
			TotalItem: total,
			TotalPage: totalPage,
		},
	})
}

func (h *ItemHandler) DownloadInvoicePDF(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil || id == 0 {
		return fiber.NewError(fiber.StatusBadRequest, "invalid invoice id")
	}

	pdfData, err := h.invoiceService.FindInvoiceByID(c.Context(), uint(id))
	if err != nil {
		h.log.Warnf("failed to find invoice %d: %v", id, err)
		return err
	}

	if len(pdfData.Items) == 0 {
		return fiber.NewError(fiber.StatusUnprocessableEntity, "invoice tidak memiliki rincian bahan; tidak dapat diunduh ulang")
	}

	pdfBuffer, err := util.GenerateTemplateInvoicePDF(pdfData)
	if err != nil {
		h.log.Warnf("failed to regenerate invoice PDF: %v", err)
		return err
	}

	safeInvoiceNo := strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			return r
		}
		return '_'
	}, pdfData.InvoiceNo)
	filename := url.PathEscape(fmt.Sprintf("invoice-%s.pdf", safeInvoiceNo))

	disposition := "attachment"
	if strings.EqualFold(c.Query("mode"), "view") {
		disposition = "inline"
	}

	c.Set("Content-Type", "application/pdf")
	c.Set("Content-Disposition", fmt.Sprintf("%s; filename=\"%s\"", disposition, filename))
	return c.Send(pdfBuffer.Bytes())
}

func (h *ItemHandler) DeleteInvoice(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil || id == 0 {
		return fiber.NewError(fiber.StatusBadRequest, "invalid invoice id")
	}

	if err := h.invoiceService.DeleteInvoice(c.Context(), uint(id)); err != nil {
		h.log.Warnf("failed to delete invoice %d: %v", id, err)
		return err
	}

	return c.JSON(model.Response[bool]{
		Status:  fiber.StatusOK,
		Message: "delete invoice success",
		Data:    true,
	})
}
