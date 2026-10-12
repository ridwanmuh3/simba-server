package handler

import (
	"context"
	"strconv"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"

	"github.com/ridwanmuh3/simba-server/internal/delivery/middleware"
	"github.com/ridwanmuh3/simba-server/internal/model"
	"github.com/ridwanmuh3/simba-server/internal/service"
)

type NotificationHandler struct {
	log                *zap.SugaredLogger
	validate           *validator.Validate
	notificationService NotificationService
}

type NotificationService interface {
	FindAll(ctx context.Context, req *model.FindAllNotificationsRequest) ([]model.NotificationResponse, int64, error)
	CountUnread(ctx context.Context, dapurID, userID uint) (*model.UnreadCountResponse, error)
	MarkRead(ctx context.Context, dapurID, notificationID, userID uint) error
	MarkAllRead(ctx context.Context, dapurID, userID uint) error
	Delete(ctx context.Context, dapurID, notificationID, userID uint) error
}

var _ NotificationService = (*service.NotificationService)(nil)

func NewNotificationHandler(logger *zap.SugaredLogger, validate *validator.Validate, notificationService NotificationService) *NotificationHandler {
	return &NotificationHandler{
		log:                 logger,
		validate:             validate,
		notificationService:  notificationService,
	}
}

func (h *NotificationHandler) FindAll(c *fiber.Ctx) error {
	dapurID, err := middleware.GetRequiredDapurID(c)
	if err != nil {
		return err
	}

	auth := middleware.GetAuthUser(c)
	req := &model.FindAllNotificationsRequest{
		Page:    c.QueryInt("page", 1),
		Size:    c.QueryInt("size", 10),
		IsRead:  parseBoolPtr(c.Query("is_read", "")),
		Type:    c.Query("type", ""),
		DapurID: dapurID,
		UserID:  uint(auth.ID),
	}

	response, total, err := h.notificationService.FindAll(c.Context(), req)
	if err != nil {
		h.log.Warnf("failed to get notifications: %v", err)
		return err
	}

	return c.JSON(model.Response[[]model.NotificationResponse]{
		Status:  fiber.StatusOK,
		Message: "get notifications success",
		Data:    response,
		Paging:  newPageMetadata(req.Page, req.Size, total),
	})
}

func (h *NotificationHandler) UnreadCount(c *fiber.Ctx) error {
	dapurID, err := middleware.GetRequiredDapurID(c)
	if err != nil {
		return err
	}

	auth := middleware.GetAuthUser(c)
	response, err := h.notificationService.CountUnread(c.Context(), dapurID, uint(auth.ID))
	if err != nil {
		h.log.Warnf("failed to count unread notifications: %v", err)
		return err
	}

	return c.JSON(model.Response[*model.UnreadCountResponse]{
		Status:  fiber.StatusOK,
		Message: "get unread count success",
		Data:    response,
	})
}

func (h *NotificationHandler) MarkRead(c *fiber.Ctx) error {
	dapurID, err := middleware.GetRequiredDapurID(c)
	if err != nil {
		return err
	}
	auth := middleware.GetAuthUser(c)

	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil || id == 0 {
		return fiber.NewError(fiber.StatusBadRequest, "invalid notification id")
	}

	if err := h.notificationService.MarkRead(c.Context(), dapurID, uint(id), uint(auth.ID)); err != nil {
		h.log.Warnf("failed to mark notification read: %v", err)
		return err
	}

	return c.JSON(model.Response[bool]{
		Status:  fiber.StatusOK,
		Message: "notification marked as read",
		Data:    true,
	})
}

func (h *NotificationHandler) MarkAllRead(c *fiber.Ctx) error {
	dapurID, err := middleware.GetRequiredDapurID(c)
	if err != nil {
		return err
	}
	auth := middleware.GetAuthUser(c)

	if err := h.notificationService.MarkAllRead(c.Context(), dapurID, uint(auth.ID)); err != nil {
		h.log.Warnf("failed to mark all notifications read: %v", err)
		return err
	}

	return c.JSON(model.Response[bool]{
		Status:  fiber.StatusOK,
		Message: "all notifications marked as read",
		Data:    true,
	})
}

func (h *NotificationHandler) Delete(c *fiber.Ctx) error {
	dapurID, err := middleware.GetRequiredDapurID(c)
	if err != nil {
		return err
	}
	auth := middleware.GetAuthUser(c)

	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil || id == 0 {
		return fiber.NewError(fiber.StatusBadRequest, "invalid notification id")
	}

	if err := h.notificationService.Delete(c.Context(), dapurID, uint(id), uint(auth.ID)); err != nil {
		h.log.Warnf("failed to delete notification: %v", err)
		return err
	}

	return c.JSON(model.Response[bool]{
		Status:  fiber.StatusOK,
		Message: "notification deleted",
		Data:    true,
	})
}

func parseBoolPtr(s string) *bool {
	if s == "" {
		return nil
	}
	b := s == "true" || s == "1"
	return &b
}
