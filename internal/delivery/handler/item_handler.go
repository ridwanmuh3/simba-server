package handler

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"io"
	"math"
	"path/filepath"
	"strconv"
	"strings"
	"time"

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
	reader.FieldsPerRecord = -1

	var items []model.AddItemRequest
	lineCount := 0

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			h.log.Warnf("error reading csv line %d: %v", lineCount, err)
			return exception.InvalidCsvFormatError
		}

		if lineCount == 0 {
			lineCount++
			continue
		}

		if len(record) < 5 {
			return exception.ItemCsvTooMuchColumnError
		}

		qty, _ := strconv.Atoi(record[2])
		price, _ := strconv.ParseFloat(record[4], 64)

		item := model.AddItemRequest{
			Name:        record[0],
			Category:    record[1],
			Quantity:    qty,
			MeasureUnit: record[3],
			UnitPrice:   price,
			ModifiedBy:  auth.Fullname,
		}

		items = append(items, item)
		lineCount++
	}

	if len(items) > 0 {
		err = h.itemService.AddBatches(c.Context(), &model.AddItemBatchRequest{Items: items})
		if err != nil {
			h.log.Warnf("failed to bulk insert items: %v", err)
			return exception.InternalServerError
		}
	}

	return c.JSON(fiber.Map{
		"status":  "success",
		"message": "items imported successfully",
		"count":   len(items),
	})
}

func (h *ItemHandler) ExportItems(c *fiber.Ctx) error {
	items, _, err := h.itemService.FindAll(c.Context(), nil)
	if err != nil {
		h.log.Warnf("failed to find all items: %v", err)
		return err
	}

	filename := fmt.Sprintf("./tmp/export-barang-%v.csv", time.Now().Format("02-01-2006-15-04-05"))
	c.Set("Content-Type", "text/csv")
	c.Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))

	c.Context().Response.SetBodyStreamWriter(func(w *bufio.Writer) {
		csvWriter := csv.NewWriter(w)
		csvWriter.Write([]string{"Kode", "Nama", "Kategori", "Jumlah Barang", "Harga Satuan", "Total Harga"})

		for _, item := range items {
			if err := csvWriter.Write([]string{
				item.ID,
				item.Name,
				item.Category,
				strconv.Itoa(item.Quantity) + item.MeasureUnit,
				strconv.FormatFloat(item.UnitPrice, 'f', 2, 64),
				strconv.FormatFloat(item.TotalPrice, 'f', 2, 64),
			}); err != nil {
				h.log.Warnf("failed to write item stream: %v", err)
				return
			}
		}

		csvWriter.Flush()
		if err := csvWriter.Error(); err != nil {
			h.log.Warnf("error flushing csv: %v", err)
		}
	})

	return nil
}

func (h *ItemHandler) Update(c *fiber.Ctx) error {
	auth := middleware.GetAuthUser(c)

	request := new(model.UpdateItemRequest)
	if err := c.ParamsParser(request); err != nil {
		h.log.Warnf("failed to parse request body: %v", err)
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
