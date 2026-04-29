package handler

import (
	"context"
	"fmt"
	"math"
	"net/url"
	"strconv"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"

	"github.com/ridwanmuh3/simba-server/internal/delivery/middleware"
	"github.com/ridwanmuh3/simba-server/internal/model"
	"github.com/ridwanmuh3/simba-server/internal/service"
	"github.com/ridwanmuh3/simba-server/internal/util"
)

type InvoiceHandler struct {
	log            *zap.SugaredLogger
	validate       *validator.Validate
	invoiceService InvoiceService
	settingService SettingService
}

type InvoiceService interface {
	GetInvoiceItems(ctx context.Context, request *model.GetInvoiceItemsRequest) (*model.InvoiceSummary, error)
	SaveInvoice(ctx context.Context, request *model.GenerateInvoiceRequest, summary *model.InvoiceSummary) error
	FindAllInvoices(ctx context.Context, request *model.FindAllInvoicesRequest) ([]model.InvoiceResponse, int64, error)
	FindInvoiceByID(ctx context.Context, id uint, dapurID uint) (*model.InvoiceData, error)
	FindInvoiceDetail(ctx context.Context, id uint, dapurID uint) (*model.InvoiceDetailResponse, error)
	UpdateInvoice(ctx context.Context, request *model.UpdateInvoiceRequest) error
	DeleteInvoice(ctx context.Context, id uint, dapurID uint) error
}

type SettingService interface {
	GetCompanyProfile(ctx context.Context, dapurID uint) (*model.CompanyProfileResponse, error)
	GetNextDocumentNumbers(ctx context.Context, dapurID uint) (*model.DocumentSequenceResponse, error)
	ConsumeDocumentNumbers(ctx context.Context, dapurID uint) error
}

var (
	_ InvoiceService = (*service.InvoiceService)(nil)
	_ SettingService = (*service.SettingService)(nil)
)

func NewInvoiceHandler(
	logger *zap.SugaredLogger,
	validate *validator.Validate,
	invoiceService InvoiceService,
	settingService SettingService,
) *InvoiceHandler {
	return &InvoiceHandler{
		log:            logger,
		validate:       validate,
		invoiceService: invoiceService,
		settingService: settingService,
	}
}

func (h *InvoiceHandler) GetInvoiceItems(c *fiber.Ctx) error {
	auth := middleware.GetAuthUser(c)
	dapurID := *auth.CurrentDapurID

	request := new(model.GenerateInvoiceRequest)
	if err := c.BodyParser(request); err != nil {
		h.log.Warnf("failed to parse invoice request body: %v", err)
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	request.DapurID = dapurID

	if request.CompanyName == "" || request.CompanyAddress == "" || request.CompanyContact == "" {
		profile, err := h.settingService.GetCompanyProfile(c.Context(), dapurID)
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
		numbers, err := h.settingService.GetNextDocumentNumbers(c.Context(), dapurID)
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
		StockIDs:  request.StockIDs,
		DapurID:   dapurID,
	}

	summary, err := h.invoiceService.GetInvoiceItems(c.Context(), invoiceItemsReq)
	if err != nil {
		h.log.Warnf("failed to get invoice items: %v", err)
		return err
	}

	label := "KELUAR"
	if stockType == "IN" {
		label = "MASUK"
	}
	if len(summary.Items) == 0 {
		return fiber.NewError(fiber.StatusNotFound, fmt.Sprintf("tidak ada data bahan %s pada rentang tanggal yang dipilih", label))
	}

	if err := h.invoiceService.SaveInvoice(c.Context(), request, summary); err != nil {
		h.log.Warnf("failed to save invoice record: %v", err)
		return err
	}

	if err := h.settingService.ConsumeDocumentNumbers(c.Context(), dapurID); err != nil {
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

	safeInvoiceNo := strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			return r
		}
		return '_'
	}, request.InvoiceNo)

	filename := url.PathEscape(fmt.Sprintf("INVOICE-BARANG %s-%s.pdf", label, safeInvoiceNo))

	c.Set("Content-Type", "application/pdf")
	c.Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	c.Set("Access-Control-Expose-Headers", "Content-Disposition")
	return c.Send(pdfBuffer.Bytes())
}

func (h *InvoiceHandler) GetInvoiceHistory(c *fiber.Ctx) error {
	auth := middleware.GetAuthUser(c)

	request := &model.FindAllInvoicesRequest{
		SearchQuery: c.Query("search_query", ""),
		StartDate:   c.Query("start_date", ""),
		EndDate:     c.Query("end_date", ""),
		Page:        c.QueryInt("page", 1),
		Size:        c.QueryInt("size", 10),
		DapurID:     *auth.CurrentDapurID,
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

func (h *InvoiceHandler) DownloadInvoicePDF(c *fiber.Ctx) error {
	auth := middleware.GetAuthUser(c)

	idStr := c.Params("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil || id == 0 {
		return fiber.NewError(fiber.StatusBadRequest, "invalid invoice id")
	}

	pdfData, err := h.invoiceService.FindInvoiceByID(c.Context(), uint(id), *auth.CurrentDapurID)
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

	label := "KELUAR"
	if pdfData.StockType == "IN" {
		label = "MASUK"
	}
	filename := url.PathEscape(fmt.Sprintf("INVOICE-BARANG %s-%s.pdf", label, safeInvoiceNo))

	disposition := "attachment"
	if strings.EqualFold(c.Query("mode"), "view") {
		disposition = "inline"
	}

	c.Set("Content-Type", "application/pdf")
	c.Set("Content-Disposition", fmt.Sprintf("%s; filename=\"%s\"", disposition, filename))
	c.Set("Access-Control-Expose-Headers", "Content-Disposition")
	return c.Send(pdfBuffer.Bytes())
}

func (h *InvoiceHandler) FindInvoiceDetail(c *fiber.Ctx) error {
	auth := middleware.GetAuthUser(c)

	idStr := c.Params("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil || id == 0 {
		return fiber.NewError(fiber.StatusBadRequest, "invalid invoice id")
	}

	detail, err := h.invoiceService.FindInvoiceDetail(c.Context(), uint(id), *auth.CurrentDapurID)
	if err != nil {
		h.log.Warnf("failed to get invoice detail %d: %v", id, err)
		return err
	}

	return c.JSON(model.Response[*model.InvoiceDetailResponse]{
		Status:  fiber.StatusOK,
		Message: "get invoice detail success",
		Data:    detail,
	})
}

func (h *InvoiceHandler) UpdateInvoice(c *fiber.Ctx) error {
	auth := middleware.GetAuthUser(c)

	idStr := c.Params("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil || id == 0 {
		return fiber.NewError(fiber.StatusBadRequest, "invalid invoice id")
	}

	request := new(model.UpdateInvoiceRequest)
	if err := c.BodyParser(request); err != nil {
		h.log.Warnf("failed to parse update invoice request: %v", err)
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	request.ID = uint(id)
	request.DapurID = *auth.CurrentDapurID

	if err := h.validate.Struct(request); err != nil {
		h.log.Warnf("failed to validate update invoice request: %v", err)
		return fiber.NewError(fiber.StatusBadRequest, "validation failed")
	}

	if err := h.invoiceService.UpdateInvoice(c.Context(), request); err != nil {
		h.log.Warnf("failed to update invoice %d: %v", id, err)
		return err
	}

	return c.JSON(model.Response[bool]{
		Status:  fiber.StatusOK,
		Message: "update invoice success",
		Data:    true,
	})
}

func (h *InvoiceHandler) DeleteInvoice(c *fiber.Ctx) error {
	auth := middleware.GetAuthUser(c)

	idStr := c.Params("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil || id == 0 {
		return fiber.NewError(fiber.StatusBadRequest, "invalid invoice id")
	}

	if err := h.invoiceService.DeleteInvoice(c.Context(), uint(id), *auth.CurrentDapurID); err != nil {
		h.log.Warnf("failed to delete invoice %d: %v", id, err)
		return err
	}

	return c.JSON(model.Response[bool]{
		Status:  fiber.StatusOK,
		Message: "delete invoice success",
		Data:    true,
	})
}
