package handler

import (
	"context"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"

	"github.com/ridwanmuh3/simba-server/internal/delivery/middleware"
	"github.com/ridwanmuh3/simba-server/internal/model"
	"github.com/ridwanmuh3/simba-server/internal/service"
)

type SettingHandler struct {
	log            *zap.SugaredLogger
	settingService CompanySettingService
}

type CompanySettingService interface {
	GetCompanyProfile(ctx context.Context, dapurID uint) (*model.CompanyProfileResponse, error)
	UpdateCompanyProfile(ctx context.Context, request *model.CompanyProfileRequest, dapurID uint) error
	GetNextDocumentNumbers(ctx context.Context, dapurID uint) (*model.DocumentSequenceResponse, error)
}

var _ CompanySettingService = (*service.SettingService)(nil)

func NewSettingHandler(logger *zap.SugaredLogger, settingService CompanySettingService) *SettingHandler {
	return &SettingHandler{
		log:            logger,
		settingService: settingService,
	}
}

func (h *SettingHandler) GetCompanyProfile(c *fiber.Ctx) error {
	auth := middleware.GetAuthUser(c)

	response, err := h.settingService.GetCompanyProfile(c.Context(), *auth.CurrentDapurID)
	if err != nil {
		h.log.Warnf("failed to get company profile: %v", err)
		return err
	}

	return c.JSON(model.Response[*model.CompanyProfileResponse]{
		Status:  fiber.StatusOK,
		Message: "get company profile success",
		Data:    response,
	})
}

func (h *SettingHandler) UpdateCompanyProfile(c *fiber.Ctx) error {
	auth := middleware.GetAuthUser(c)

	request := new(model.CompanyProfileRequest)
	if err := c.BodyParser(request); err != nil {
		h.log.Warnf("failed to parse request body: %v", err)
		return err
	}

	if err := h.settingService.UpdateCompanyProfile(c.Context(), request, *auth.CurrentDapurID); err != nil {
		h.log.Warnf("failed to update company profile: %v", err)
		return err
	}

	return c.JSON(model.Response[any]{
		Status:  fiber.StatusOK,
		Message: "update company profile success",
		Data:    nil,
	})
}

func (h *SettingHandler) GetNextDocumentNumbers(c *fiber.Ctx) error {
	auth := middleware.GetAuthUser(c)

	response, err := h.settingService.GetNextDocumentNumbers(c.Context(), *auth.CurrentDapurID)
	if err != nil {
		h.log.Warnf("failed to get next document numbers: %v", err)
		return err
	}

	return c.JSON(model.Response[*model.DocumentSequenceResponse]{
		Status:  fiber.StatusOK,
		Message: "get next document numbers success",
		Data:    response,
	})
}
