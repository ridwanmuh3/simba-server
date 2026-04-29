package handler

import (
	"context"
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
	itemService ItemService
}

type ItemService interface {
	Add(ctx context.Context, request *model.AddItemRequest) (*model.ItemResponse, error)
	AddBatches(ctx context.Context, request *model.AddItemBatchRequest) error
	ExportItems(ctx context.Context, dapurID uint) ([]model.ItemResponse, int, error)
	Update(ctx context.Context, request *model.UpdateItemRequest) (*model.ItemResponse, error)
	Delete(ctx context.Context, request *model.DeleteItemRequest) (bool, error)
	FindById(ctx context.Context, request *model.FindByIdItemRequest) (*model.ItemResponse, error)
	FindAll(ctx context.Context, request *model.FindAllItemsRequest) ([]model.ItemResponse, int64, error)
	GetItemCategories(ctx context.Context, dapurID uint) ([]string, error)
}

var _ ItemService = (*service.ItemService)(nil)

func NewItemHandler(config *viper.Viper, logger *zap.SugaredLogger, itemService ItemService) *ItemHandler {
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
	request.DapurID = *auth.CurrentDapurID

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
			DapurID:     *auth.CurrentDapurID,
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

	err = h.itemService.AddBatches(c.Context(), &model.AddItemBatchRequest{Items: items, DapurID: *auth.CurrentDapurID})
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
	auth := middleware.GetAuthUser(c)

	items, _, err := h.itemService.ExportItems(c.Context(), *auth.CurrentDapurID)
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
	request.DapurID = *auth.CurrentDapurID

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

func (h *ItemHandler) Delete(c *fiber.Ctx) error {
	auth := middleware.GetAuthUser(c)

	request := new(model.DeleteItemRequest)
	if err := c.ParamsParser(request); err != nil {
		h.log.Warnf("failed to parse request body: %v", err)
		return err
	}

	request.DapurID = *auth.CurrentDapurID

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

func (h *ItemHandler) FindById(c *fiber.Ctx) error {
	auth := middleware.GetAuthUser(c)

	request := new(model.FindByIdItemRequest)
	if err := c.ParamsParser(request); err != nil {
		h.log.Warnf("failed to parse request body: %v", err)
		return err
	}

	request.DapurID = *auth.CurrentDapurID

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

func (h *ItemHandler) GetItemCategories(c *fiber.Ctx) error {
	auth := middleware.GetAuthUser(c)

	categories, err := h.itemService.GetItemCategories(c.Context(), *auth.CurrentDapurID)
	if err != nil {
		h.log.Warnf("failed to get item categories: %v", err)
		return err
	}

	return c.JSON(model.Response[[]string]{
		Status:  fiber.StatusOK,
		Message: "get item categories success",
		Data:    categories,
	})
}

func (h *ItemHandler) FindAll(c *fiber.Ctx) error {
	auth := middleware.GetAuthUser(c)

	request := &model.FindAllItemsRequest{
		SearchQuery: c.Query("search_query", ""),
		StartDate:   c.Query("start_date", ""),
		EndDate:     c.Query("end_date", ""),
		Page:        c.QueryInt("page", 1),
		Size:        c.QueryInt("size", 10),
		DapurID:     *auth.CurrentDapurID,
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
