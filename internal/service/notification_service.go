package service

import (
	"context"

	"github.com/go-playground/validator/v10"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/ridwanmuh3/simba-server/internal/entity"
	"github.com/ridwanmuh3/simba-server/internal/exception"
	"github.com/ridwanmuh3/simba-server/internal/model"
)

type NotificationService struct {
	db           *gorm.DB
	log          *zap.SugaredLogger
	validate     *validator.Validate
	notificationRepo NotificationRepository
}

type NotificationRepository interface {
	FindAll(db *gorm.DB, req *model.FindAllNotificationsRequest) ([]entity.Notification, int64, error)
	CountUnread(db *gorm.DB, dapurID, userID uint) (int64, error)
	MarkRead(db *gorm.DB, dapurID, notificationID uint, userID uint) error
	MarkAllRead(db *gorm.DB, dapurID, userID uint) error
	Delete(db *gorm.DB, dapurID, notificationID uint, userID uint) error
}

func NewNotificationService(db *gorm.DB, log *zap.SugaredLogger, validate *validator.Validate, notificationRepo NotificationRepository) *NotificationService {
	return &NotificationService{
		db:              db,
		log:             log,
		validate:        validate,
		notificationRepo: notificationRepo,
	}
}

func (s *NotificationService) FindAll(ctx context.Context, req *model.FindAllNotificationsRequest) ([]model.NotificationResponse, int64, error) {
	db := s.db.WithContext(ctx)

	if err := s.validate.Struct(req); err != nil {
		return nil, 0, err
	}

	page := req.Page
	if page < 1 {
		page = 1
	}
	size := req.Size
	if size < 1 {
		size = 10
	}
	req.Page = page
	req.Size = size

	notifications, total, err := s.notificationRepo.FindAll(db, req)
	if err != nil {
		s.log.Errorf("failed to find notifications: %v", err)
		return nil, 0, exception.InternalServerError
	}

	responses := make([]model.NotificationResponse, len(notifications))
	for i, n := range notifications {
		responses[i] = toNotificationResponse(&n)
	}

	return responses, total, nil
}

func (s *NotificationService) CountUnread(ctx context.Context, dapurID, userID uint) (*model.UnreadCountResponse, error) {
	db := s.db.WithContext(ctx)

	count, err := s.notificationRepo.CountUnread(db, dapurID, userID)
	if err != nil {
		s.log.Errorf("failed to count unread notifications: %v", err)
		return nil, exception.InternalServerError
	}

	return &model.UnreadCountResponse{Count: count}, nil
}

func (s *NotificationService) MarkRead(ctx context.Context, dapurID, notificationID, userID uint) error {
	db := s.db.WithContext(ctx)

	if err := s.notificationRepo.MarkRead(db, dapurID, notificationID, userID); err != nil {
		s.log.Errorf("failed to mark notification read: %v", err)
		return exception.InternalServerError
	}

	return nil
}

func (s *NotificationService) MarkAllRead(ctx context.Context, dapurID, userID uint) error {
	db := s.db.WithContext(ctx)

	if err := s.notificationRepo.MarkAllRead(db, dapurID, userID); err != nil {
		s.log.Errorf("failed to mark all notifications read: %v", err)
		return exception.InternalServerError
	}

	return nil
}

func (s *NotificationService) Delete(ctx context.Context, dapurID, notificationID, userID uint) error {
	db := s.db.WithContext(ctx)

	if err := s.notificationRepo.Delete(db, dapurID, notificationID, userID); err != nil {
		s.log.Errorf("failed to delete notification: %v", err)
		return exception.InternalServerError
	}

	return nil
}

func toNotificationResponse(n *entity.Notification) model.NotificationResponse {
	var actorID *uint
	if n.ActorID != nil {
		actorID = n.ActorID
	}
	return model.NotificationResponse{
		ID:        uint(n.ID),
		Type:      n.Type,
		Title:     n.Title,
		Message:   n.Message,
		Level:     n.Level,
		IsRead:    n.IsRead,
		ActorID:   actorID,
		MetaJSON:  n.MetaJSON,
		CreatedAt: n.CreatedAt,
	}
}
