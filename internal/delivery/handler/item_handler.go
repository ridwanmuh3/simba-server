package handler

import (
	"encoding/csv"
	"fmt"
	"io"
	"math"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/spf13/viper"
	"go.uber.org/zap"

	"github.com/ridwanmuh3/simba-server/internal/delivery/middleware"
	"github.com/ridwanmuh3/simba-server/internal/exception"
	"github.com/ridwanmuh3/simba-server/internal/model"
	"github.com/ridwanmuh3/simba-server/internal/service"
)

type ItemHandler struct {
	config      *viper.Viper
	log         *zap.SugaredLogger
	itemService *service.ItemService
}

func NewItemHandler(config *viper.Viper, logger *zap.SugaredLogger, itemService *service.ItemService) *ItemHandler {
	return &ItemHandler{
		config:      config,
		log:         logger,
		itemService: itemService,
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

		stock, err := strconv.Atoi(record[2])
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

	if len(items) > 0 {
		err = h.itemService.AddBatches(c.Context(), &model.AddItemBatchRequest{Items: items})
		if err != nil {
			h.log.Warnf("failed to bulk insert items: %v", err)
			return exception.InternalServerError
		}
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
	h.log.Info(request)

	response, err := h.itemService.UpdateStock(c.Context(), request)
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
	id := c.Params("id")
	stockIdStr := c.Params("stock_id")

	stockIdInt, err := strconv.Atoi(stockIdStr)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "stock_id must be a number")
	}

	request := &model.DeleteStockRequest{
		ID:      id,
		StockID: stockIdInt,
	}

	response, err := h.itemService.DeleteStock(c.Context(), request)
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

	response, total, err := h.itemService.FindAllStocks(c.Context(), request)
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
	response, err := h.itemService.GetStocksFinanceSummary(c.Context())
	if err != nil {
		h.log.Warnf("failed to get stocks finance summary: %v", err)
		return err
	}

	return c.JSON(model.Response[*model.StocksSummaryResponse]{
		Status:  fiber.StatusOK,
		Message: "get stocks finance summary success",
		Data:    response,
	})
}
