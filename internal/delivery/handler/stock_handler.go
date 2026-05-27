package handler

import (
	"context"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"

	"github.com/ridwanmuh3/simba-server/internal/delivery/middleware"
	"github.com/ridwanmuh3/simba-server/internal/model"
	"github.com/ridwanmuh3/simba-server/internal/service"
)

type StockHandler struct {
	log          *zap.SugaredLogger
	stockService StockService
}

type StockService interface {
	UpdateStock(ctx context.Context, request *model.UpdateItemStockRequest) (*model.StockResponse, error)
	EditStock(ctx context.Context, request *model.EditStockRequest) (*model.StockResponse, error)
	DeleteStock(ctx context.Context, request *model.DeleteStockRequest) (bool, error)
	FindAllStocks(ctx context.Context, request *model.FindAllStocksRequest) ([]model.StockResponse, int64, error)
	GetStocksFinanceSummary(ctx context.Context, dapurID uint) (*model.StocksFinanceSummaryResponse, error)
	GetItemStocksSummary(ctx context.Context, request *model.GetItemStockSummaryRequest) ([]model.ItemStocksSummaryResponse, int64, error)
	GetLastStockPrice(ctx context.Context, itemID string, stockType string, dapurID uint) (float64, error)
}

var _ StockService = (*service.StockService)(nil)

func NewStockHandler(logger *zap.SugaredLogger, stockService StockService) *StockHandler {
	return &StockHandler{
		log:          logger,
		stockService: stockService,
	}
}

func (h *StockHandler) UpdateStock(c *fiber.Ctx) error {
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
	request.DapurID = *auth.CurrentDapurID

	response, err := h.stockService.UpdateStock(c.Context(), request)
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

func (h *StockHandler) EditStock(c *fiber.Ctx) error {
	auth := middleware.GetAuthUser(c)

	stockIdStr := c.Params("stock_id")
	stockIdInt, err := strconv.Atoi(stockIdStr)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "stock_id must be a number")
	}

	request := new(model.EditStockRequest)
	if err := c.ParamsParser(request); err != nil {
		h.log.Warnf("failed to parse request params: %v", err)
		return err
	}
	if err := c.BodyParser(request); err != nil {
		h.log.Warnf("failed to parse request body: %v", err)
		return err
	}

	request.StockID = stockIdInt
	request.ModifiedBy = auth.Fullname
	request.DapurID = *auth.CurrentDapurID

	response, err := h.stockService.EditStock(c.Context(), request)
	if err != nil {
		h.log.Warnf("failed to edit stock item: %v", err)
		return err
	}

	return c.JSON(model.Response[*model.StockResponse]{
		Status:  fiber.StatusOK,
		Message: "edit stock item success",
		Data:    response,
	})
}

func (h *StockHandler) DeleteStock(c *fiber.Ctx) error {
	auth := middleware.GetAuthUser(c)
	id := c.Params("id")
	stockIdStr := c.Params("stock_id")

	stockIdInt, err := strconv.Atoi(stockIdStr)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "stock_id must be a number")
	}

	request := &model.DeleteStockRequest{
		ID:         id,
		StockID:    stockIdInt,
		ModifiedBy: auth.Fullname,
		DapurID:    *auth.CurrentDapurID,
	}

	response, err := h.stockService.DeleteStock(c.Context(), request)
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

func (h *StockHandler) FindAllStocks(c *fiber.Ctx) error {
	auth := middleware.GetAuthUser(c)

	request := &model.FindAllStocksRequest{
		SearchQuery: c.Query("search_query", ""),
		StartDate:   c.Query("start_date", ""),
		EndDate:     c.Query("end_date", ""),
		Page:        c.QueryInt("page", 1),
		Size:        c.QueryInt("size", 10),
		Type:        c.Query("type", "ALL"),
		Category:    c.Query("category", ""),
		DapurID:     *auth.CurrentDapurID,
	}

	response, total, err := h.stockService.FindAllStocks(c.Context(), request)
	if err != nil {
		h.log.Warnf("failed to find all items stocks: %v", err)
		return err
	}

	return c.JSON(model.Response[[]model.StockResponse]{
		Status:  fiber.StatusOK,
		Message: "find all items stocks success",
		Data:    response,
		Paging:  newPageMetadata(request.Page, request.Size, total),
	})
}

func (h *StockHandler) GetStocksFinanceSummary(c *fiber.Ctx) error {
	auth := middleware.GetAuthUser(c)

	response, err := h.stockService.GetStocksFinanceSummary(c.Context(), *auth.CurrentDapurID)
	if err != nil {
		h.log.Warnf("failed to get stocks finance summary: %v", err)
		return err
	}

	return c.JSON(model.Response[*model.StocksFinanceSummaryResponse]{
		Status:  fiber.StatusOK,
		Message: "get stocks finance summary success",
		Data:    response,
	})
}

func (h *StockHandler) GetLastStockPrice(c *fiber.Ctx) error {
	auth := middleware.GetAuthUser(c)
	itemID := c.Params("id")
	stockType := c.Query("type", "IN")

	price, err := h.stockService.GetLastStockPrice(c.Context(), itemID, stockType, *auth.CurrentDapurID)
	if err != nil {
		h.log.Warnf("failed to get last stock price: %v", err)
		return err
	}

	return c.JSON(model.Response[float64]{
		Status:  fiber.StatusOK,
		Message: "get last stock price success",
		Data:    price,
	})
}

func (h *StockHandler) GetItemStocksSummary(c *fiber.Ctx) error {
	auth := middleware.GetAuthUser(c)

	request := &model.GetItemStockSummaryRequest{
		StartDate:   c.Query("start_date", ""),
		EndDate:     c.Query("end_date", ""),
		Page:        c.QueryInt("page", 1),
		Size:        c.QueryInt("size", 10),
		SearchQuery: c.Query("search_query"),
		DapurID:     *auth.CurrentDapurID,
	}

	response, total, err := h.stockService.GetItemStocksSummary(c.Context(), request)
	if err != nil {
		h.log.Warnf("failed to get item stock summary: %v", err)
		return err
	}

	return c.JSON(model.Response[[]model.ItemStocksSummaryResponse]{
		Status:  fiber.StatusOK,
		Message: "get item stock summary success",
		Data:    response,
		Paging:  newPageMetadata(request.Page, request.Size, total),
	})
}
