package handler

import (
	"github.com/gofiber/fiber/v2"
	"github.com/spf13/viper"
	"go.uber.org/zap"

	"github.com/ridwanmuh3/simba-server/internal/model"
	"github.com/ridwanmuh3/simba-server/internal/service"
)

type DashboardHandler struct {
	config           *viper.Viper
	log              *zap.SugaredLogger
	dashboardService *service.DashboardService
}

func NewDashboardHandler(config *viper.Viper, logger *zap.SugaredLogger, dashboardService *service.DashboardService) *DashboardHandler {
	return &DashboardHandler{
		config:           config,
		log:              logger,
		dashboardService: dashboardService,
	}
}

func (h *DashboardHandler) GetDashboardStats(c *fiber.Ctx) error {
	response, err := h.dashboardService.GetDashboardStats(c.Context())
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
