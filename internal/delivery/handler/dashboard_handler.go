package handler

import (
	"context"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"

	"github.com/ridwanmuh3/simba-server/internal/delivery/middleware"
	"github.com/ridwanmuh3/simba-server/internal/model"
	"github.com/ridwanmuh3/simba-server/internal/service"
)

type DashboardHandler struct {
	log               *zap.SugaredLogger
	dashboardService  DashboardServiceInterface
}

type DashboardServiceInterface interface {
	GetDashboardStats(ctx context.Context, dapurID uint) (*model.DashboardStatsResponse, error)
}

var _ DashboardServiceInterface = (*service.DashboardService)(nil)

func NewDashboardHandler(logger *zap.SugaredLogger, dashboardService DashboardServiceInterface) *DashboardHandler {
	return &DashboardHandler{
		log:               logger,
		dashboardService:  dashboardService,
	}
}

func (h *DashboardHandler) GetDashboardStats(c *fiber.Ctx) error {
	auth := middleware.GetAuthUser(c)

	response, err := h.dashboardService.GetDashboardStats(c.Context(), *auth.CurrentDapurID)
	if err != nil {
		h.log.Warnf("failed to get dashboard stats: %v", err)
		return err
	}

	return c.JSON(model.Response[*model.DashboardStatsResponse]{
		Status:  fiber.StatusOK,
		Message: "get dashboard stats success",
		Data:    response,
	})
}
