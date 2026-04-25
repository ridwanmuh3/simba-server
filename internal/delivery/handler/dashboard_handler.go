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

type DashboardHandler struct {
	config           *viper.Viper
	log              *zap.SugaredLogger
	dashboardService DashboardServiceIface
}

type DashboardServiceIface interface {
	GetDashboardStats(ctx context.Context, dapurID uint) (*model.DashboardStatsResponse, error)
}

var _ DashboardServiceIface = (*service.DashboardService)(nil)

func NewDashboardHandler(config *viper.Viper, logger *zap.SugaredLogger, dashboardService DashboardServiceIface) *DashboardHandler {
	return &DashboardHandler{
		config:           config,
		log:              logger,
		dashboardService: dashboardService,
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
