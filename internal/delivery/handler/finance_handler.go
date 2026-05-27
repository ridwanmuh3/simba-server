package handler

import (
	"context"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"

	"github.com/ridwanmuh3/simba-server/internal/delivery/middleware"
	"github.com/ridwanmuh3/simba-server/internal/exception"
	"github.com/ridwanmuh3/simba-server/internal/model"
	"github.com/ridwanmuh3/simba-server/internal/service"
)

type FinanceHandler struct {
	log            *zap.SugaredLogger
	financeService FinanceService
}

type FinanceService interface {
	Add(ctx context.Context, request *model.AddFinanceRequest) (*model.FinanceResponse, error)
	Update(ctx context.Context, request *model.UpdateFinanceRequest) (*model.FinanceResponse, error)
	Delete(ctx context.Context, request *model.DeleteFinanceRequest) (bool, error)
	FindById(ctx context.Context, request *model.FindByIdFinanceRequest) (*model.FinanceResponse, error)
	FindAll(ctx context.Context, request *model.FindAllFinanceRequest) ([]model.FinanceResponse, int64, error)
	Export(ctx context.Context, dapurID uint) ([]model.FinanceResponse, int64, error)
}

var _ FinanceService = (*service.FinanceService)(nil)

func NewFinanceHandler(logger *zap.SugaredLogger, financeService FinanceService) *FinanceHandler {
	return &FinanceHandler{
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
		return fiber.NewError(fiber.StatusBadRequest, "amount must be a valid integer")
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

	proofImage, filePath, err := saveFinanceProofImage(h.log, c, proofImg)
	if err != nil {
		h.log.Warnf("failed to save proof image: %v", err)
		return err
	}

	request.ModifiedBy = auth.Fullname
	request.DapurID = *auth.CurrentDapurID
	request.ProofImage = proofImage

	response, err := h.financeService.Add(c.Context(), request)
	if err != nil {
		h.log.Warnf("failed to add finance data: %v", err)
		removeSavedFile(h.log, filePath)
		return err
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
	request.DapurID = *auth.CurrentDapurID

	oldData, err := h.financeService.FindById(c.Context(), &model.FindByIdFinanceRequest{ID: financeId, DapurID: *auth.CurrentDapurID})
	if err != nil {
		h.log.Warnf("failed to get finance data: %v", err)
		return err
	}

	request.ProofImage = oldData.ProofImage

	newFilePath := ""
	if proofImg, err := c.FormFile("proof_image"); err == nil {
		proofImage, filePath, err := saveFinanceProofImage(h.log, c, proofImg)
		if err != nil {
			h.log.Warnf("failed to save proof image: %v", err)
			return err
		}
		request.ProofImage = proofImage
		newFilePath = filePath
	}

	request.ModifiedBy = auth.Fullname

	response, err := h.financeService.Update(c.Context(), request)
	if err != nil {
		h.log.Warnf("failed to update finance data: %v", err)
		removeSavedFile(h.log, newFilePath)
		return err
	}
	if newFilePath != "" && oldData.ProofImage != request.ProofImage {
		removeFinanceProofImage(h.log, oldData.ProofImage)
	}

	return c.JSON(model.Response[*model.FinanceResponse]{
		Status:  fiber.StatusOK,
		Message: "update finance data success",
		Data:    response,
	})
}

func (h *FinanceHandler) Export(c *fiber.Ctx) error {
	auth := middleware.GetAuthUser(c)
	finances, _, err := h.financeService.Export(c.Context(), *auth.CurrentDapurID)
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
	auth := middleware.GetAuthUser(c)
	request := new(model.DeleteFinanceRequest)
	if err := c.ParamsParser(request); err != nil {
		h.log.Warnf("failed to parse request body: %v", err)
		return err
	}
	request.DapurID = *auth.CurrentDapurID

	oldData, err := h.financeService.FindById(c.Context(), &model.FindByIdFinanceRequest{ID: request.ID, DapurID: request.DapurID})
	if err != nil {
		h.log.Warnf("failed to get finance data: %v", err)
		return err
	}

	response, err := h.financeService.Delete(c.Context(), request)
	if err != nil {
		h.log.Warnf("failed to delete finance data: %v", err)
		return err
	}
	removeFinanceProofImage(h.log, oldData.ProofImage)

	return c.JSON(model.Response[bool]{
		Status:  fiber.StatusOK,
		Message: "delete finance data success",
		Data:    response,
	})
}

func (h *FinanceHandler) FindById(c *fiber.Ctx) error {
	auth := middleware.GetAuthUser(c)
	request := new(model.FindByIdFinanceRequest)
	if err := c.ParamsParser(request); err != nil {
		h.log.Warnf("failed to parse request body: %v", err)
		return err
	}
	request.DapurID = *auth.CurrentDapurID

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
	auth := middleware.GetAuthUser(c)
	request := &model.FindAllFinanceRequest{
		SearchQuery: c.Query("search_query", ""),
		StartDate:   c.Query("start_date", ""),
		EndDate:     c.Query("end_date", ""),
		Page:        c.QueryInt("page", 1),
		Size:        c.QueryInt("size", 10),
		DapurID:     *auth.CurrentDapurID,
	}

	response, total, err := h.financeService.FindAll(c.Context(), request)
	if err != nil {
		h.log.Warnf("failed to find all finances: %v", err)
		return err
	}

	return c.JSON(model.Response[[]model.FinanceResponse]{
		Status:  fiber.StatusOK,
		Message: "find all finances data success",
		Data:    response,
		Paging:  newPageMetadata(request.Page, request.Size, total),
	})
}
