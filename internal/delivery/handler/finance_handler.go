package handler

import (
	"math"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/spf13/viper"
	"go.uber.org/zap"

	"github.com/ridwanmuh3/simba-server/internal/delivery/middleware"
	"github.com/ridwanmuh3/simba-server/internal/exception"
	"github.com/ridwanmuh3/simba-server/internal/model"
	"github.com/ridwanmuh3/simba-server/internal/service"
)

type FinanceHandler struct {
	config         *viper.Viper
	log            *zap.SugaredLogger
	financeService *service.FinanceService
}

func NewFinanceHandler(config *viper.Viper, logger *zap.SugaredLogger, financeService *service.FinanceService) *FinanceHandler {
	return &FinanceHandler{
		config:         config,
		log:            logger,
		financeService: financeService,
	}
}

func (h *FinanceHandler) Add(c *fiber.Ctx) error {
	auth := middleware.GetAuthUser(c)

	typeFinance := c.FormValue("type")
	category := c.FormValue("category")
	description := c.FormValue("description")
	strAmount := c.FormValue("amount")
	amount, err := strconv.Atoi(strAmount)
	if err != nil {
		h.log.Warnf("failed to convert amount: %v", err)
		return exception.InternalServerError
	}
	extraNote := c.FormValue("extra_note")

	request := new(model.AddFinanceRequest)
	request.Type = typeFinance
	request.Category = category
	request.Description = description
	request.Amount = amount
	request.ExtraNote = extraNote

	proofImg, err := c.FormFile("proof_image")
	if err != nil {
		h.log.Warnf("failed to get uploaded file: %v", err)
		return exception.InvalidUploadedFileError
	}

	ext := strings.ToLower(filepath.Ext(proofImg.Filename))
	allowedExt := []string{".png", ".jpg", ".jpeg"}

	if !slices.Contains(allowedExt, ext) {
		h.log.Warnf("invalid uploaded file format: %s", ext)
		return exception.InvalidFileFormatError
	}

	const maxFileSize = int64(15 * 1024 * 1024)
	if proofImg.Size > maxFileSize {
		h.log.Warnf("file size exceeded: %d bytes", proofImg.Size)
		return exception.ExceedMaximumFileSizeError
	}

	fileName := uuid.New().String() + ext

	uploadDir := filepath.Join("uploads", "finances-proof")

	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		h.log.Warnf("failed to create upload directory: %v", err)
		return exception.InternalServerError
	}

	filePath := filepath.Join(uploadDir, fileName)

	if err := c.SaveFile(proofImg, filePath); err != nil {
		h.log.Warnf("failed to save uploaded file: %v", err)
		return exception.InternalServerError
	}

	request.ModifiedBy = auth.Fullname
	request.ProofImage = "/uploads/finances-proof/" + fileName

	response, err := h.financeService.Add(c.Context(), request)
	if err != nil {
		h.log.Warnf("failed to add finance data: %v", err)
		return exception.InternalServerError
	}

	return c.Status(fiber.StatusCreated).JSON(model.Response[*model.FinanceResponse]{
		Status:  fiber.StatusCreated,
		Message: "add finance data success",
		Data:    response,
	})
}

func (h *FinanceHandler) Update(c *fiber.Ctx) error {
	auth := middleware.GetAuthUser(c)

	financeId, err := c.ParamsInt("id")
	if err != nil {
		h.log.Warnf("invalid type of passing params")
		return exception.InternalServerError
	}

	request := new(model.UpdateFinanceRequest)
	if err := c.BodyParser(request); err != nil {
		h.log.Warnf("failed to parse request body: %v", err)
		return err
	}
	request.ID = financeId

	oldData, err := h.financeService.FindById(c.Context(), &model.FindByIdFinanceRequest{ID: financeId})
	if err != nil {
		h.log.Warnf("failed to get finance data: %v", err)
		return err
	}

	request.ProofImage = oldData.ProofImage

	proofImg, err := c.FormFile("proof_image")
	if err == nil {
		ext := strings.ToLower(filepath.Ext(proofImg.Filename))
		allowedExt := []string{".png", ".jpg", ".jpeg"}

		if !slices.Contains(allowedExt, ext) {
			h.log.Warnf("invalid uploaded file format: %s", ext)
			return exception.InvalidFileFormatError
		}

		const maxFileSize = int64(15 * 1024 * 1024)
		if proofImg.Size > maxFileSize {
			h.log.Warnf("file size exceeded: %d bytes", proofImg.Size)
			return exception.ExceedMaximumFileSizeError
		}

		fileName := uuid.New().String() + ext
		uploadDir := filepath.Join("uploads", "finances-proof")

		if err := os.MkdirAll(uploadDir, 0755); err != nil {
			h.log.Warnf("failed to create upload directory: %v", err)
			return exception.InternalServerError
		}

		newFilePath := filepath.Join(uploadDir, fileName)

		if err := c.SaveFile(proofImg, newFilePath); err != nil {
			h.log.Warnf("failed to save uploaded file: %v", err)
			return exception.InternalServerError
		}

		if oldData.ProofImage != "" {
			fileName := filepath.Base(oldData.ProofImage)
			oldFilePath := filepath.Join("uploads", "finances-proof", fileName)

			if err := os.Remove(oldFilePath); err != nil {
				if !os.IsNotExist(err) {
					h.log.Warnf("failed to delete old file: %v", err)
				} else {
					h.log.Warnf("file not found, skip delete: %s", oldFilePath)
				}
			}
		}

		request.ProofImage = "/uploads/finances-proof/" + fileName
	}

	request.ModifiedBy = auth.Fullname

	response, err := h.financeService.Update(c.Context(), request)
	if err != nil {
		h.log.Warnf("failed to update finance data: %v", err)
		return err
	}

	return c.JSON(model.Response[*model.FinanceResponse]{
		Status:  fiber.StatusOK,
		Message: "update finance data success",
		Data:    response,
	})
}

func (h *FinanceHandler) Export(c *fiber.Ctx) error {
	finances, _, err := h.financeService.Export(c.Context())
	if err != nil {
		h.log.Warnf("failed to find all finances data: %v", err)
		return err
	}

	return c.JSON(model.Response[[]model.FinanceResponse]{
		Status:  fiber.StatusOK,
		Message: "export finances data success",
		Data:    finances,
	})
}

func (h *FinanceHandler) Delete(c *fiber.Ctx) error {
	request := new(model.DeleteFinanceRequest)
	if err := c.ParamsParser(request); err != nil {
		h.log.Warnf("failed to parse request body: %v", err)
		return err
	}

	response, err := h.financeService.Delete(c.Context(), request)
	if err != nil {
		h.log.Warnf("failed to delete finance data: %v", err)
		return err
	}

	return c.JSON(model.Response[bool]{
		Status:  fiber.StatusOK,
		Message: "delete finance data success",
		Data:    response,
	})
}

func (h *FinanceHandler) FindById(c *fiber.Ctx) error {
	request := new(model.FindByIdFinanceRequest)
	if err := c.ParamsParser(request); err != nil {
		h.log.Warnf("failed to parse request body: %v", err)
		return err
	}

	response, err := h.financeService.FindById(c.Context(), request)
	if err != nil {
		h.log.Warnf("failed to find finance data by id: %v", err)
		return err
	}

	return c.JSON(model.Response[*model.FinanceResponse]{
		Status:  fiber.StatusOK,
		Message: "find finance data by id success",
		Data:    response,
	})
}

func (h *FinanceHandler) FindAll(c *fiber.Ctx) error {
	request := &model.FindAllFinanceRequest{
		SearchQuery: c.Query("search_query", ""),
		StartDate:   c.Query("start_date", ""),
		EndDate:     c.Query("end_date", ""),
		Page:        c.QueryInt("page", 1),
		Size:        c.QueryInt("size", 10),
	}

	response, total, err := h.financeService.FindAll(c.Context(), request)
	if err != nil {
		h.log.Warnf("failed to find all finances: %v", err)
		return err
	}

	pagingMetadata := &model.PageMetadata{
		Page:      request.Page,
		Size:      request.Size,
		TotalItem: total,
		TotalPage: int64(math.Ceil(float64(total) / float64(request.Size))),
	}

	return c.JSON(model.Response[[]model.FinanceResponse]{
		Status:  fiber.StatusOK,
		Message: "find all finances data success",
		Data:    response,
		Paging:  pagingMetadata,
	})
}
