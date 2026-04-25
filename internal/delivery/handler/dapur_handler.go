package handler

import (
	"context"

	"github.com/gofiber/fiber/v2"
	"github.com/spf13/viper"
	"go.uber.org/zap"

	"github.com/ridwanmuh3/simba-server/internal/delivery/middleware"
	"github.com/ridwanmuh3/simba-server/internal/model"
	"github.com/ridwanmuh3/simba-server/internal/service"
)

type DapurHandler struct {
	config       *viper.Viper
	log          *zap.SugaredLogger
	dapurService DapurService
}

type DapurService interface {
	FindAll(ctx context.Context) ([]model.DapurResponse, error)
	Create(ctx context.Context, request *model.CreateDapurRequest) (*model.DapurResponse, error)
	Update(ctx context.Context, request *model.UpdateDapurRequest) (*model.DapurResponse, error)
	Delete(ctx context.Context, request *model.DeleteDapurRequest) error
	SelectDapur(ctx context.Context, request *model.SelectDapurRequest) error
}

var _ DapurService = (*service.DapurService)(nil)

func NewDapurHandler(config *viper.Viper, log *zap.SugaredLogger, dapurService DapurService) *DapurHandler {
	return &DapurHandler{
		config:       config,
		log:          log,
		dapurService: dapurService,
	}
}

func (h *DapurHandler) FindAll(c *fiber.Ctx) error {
	dapurs, err := h.dapurService.FindAll(c.Context())
	if err != nil {
		h.log.Warnf("failed to list dapurs: %v", err)
		return err
	}

	return c.JSON(model.Response[[]model.DapurResponse]{
		Status:  fiber.StatusOK,
		Message: "list dapurs success",
		Data:    dapurs,
	})
}

func (h *DapurHandler) Create(c *fiber.Ctx) error {
	request := new(model.CreateDapurRequest)
	if err := c.BodyParser(request); err != nil {
		h.log.Warnf("failed to parse request body: %v", err)
		return err
	}

	dapur, err := h.dapurService.Create(c.Context(), request)
	if err != nil {
		h.log.Warnf("failed to create dapur: %v", err)
		return err
	}

	return c.Status(fiber.StatusCreated).JSON(model.Response[*model.DapurResponse]{
		Status:  fiber.StatusCreated,
		Message: "create dapur success",
		Data:    dapur,
	})
}

func (h *DapurHandler) Update(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return fiber.ErrBadRequest
	}

	request := new(model.UpdateDapurRequest)
	if err := c.BodyParser(request); err != nil {
		h.log.Warnf("failed to parse request body: %v", err)
		return err
	}
	request.ID = uint(id)

	dapur, err := h.dapurService.Update(c.Context(), request)
	if err != nil {
		h.log.Warnf("failed to update dapur: %v", err)
		return err
	}

	return c.JSON(model.Response[*model.DapurResponse]{
		Status:  fiber.StatusOK,
		Message: "update dapur success",
		Data:    dapur,
	})
}

func (h *DapurHandler) Delete(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return fiber.ErrBadRequest
	}

	if err := h.dapurService.Delete(c.Context(), &model.DeleteDapurRequest{ID: uint(id)}); err != nil {
		h.log.Warnf("failed to delete dapur: %v", err)
		return err
	}

	return c.JSON(model.Response[any]{
		Status:  fiber.StatusOK,
		Message: "delete dapur success",
	})
}

// SelectDapur handles POST /api/auth/select-dapur.
// Validates dapur exists and is active, then stores selection server-side.
func (h *DapurHandler) SelectDapur(c *fiber.Ctx) error {
	auth := middleware.GetAuthUser(c)

	request := new(model.SelectDapurRequest)
	if err := c.BodyParser(request); err != nil {
		h.log.Warnf("failed to parse select-dapur request: %v", err)
		return err
	}
	request.UserID = auth.ID

	if err := h.dapurService.SelectDapur(c.Context(), request); err != nil {
		h.log.Warnf("failed to select dapur: %v", err)
		return err
	}

	return c.JSON(model.Response[any]{
		Status:  fiber.StatusOK,
		Message: "select dapur success",
	})
}
