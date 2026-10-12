package repository

import (
	"gorm.io/gorm"

	"github.com/ridwanmuh3/simba-server/internal/entity"
	"github.com/ridwanmuh3/simba-server/internal/model"
)

type NotificationRepository struct {
	Repository[entity.Notification]
}

func NewNotificationRepository() *NotificationRepository {
	return &NotificationRepository{}
}

func (r *NotificationRepository) FindAll(db *gorm.DB, req *model.FindAllNotificationsRequest) ([]entity.Notification, int64, error) {
	var notifications []entity.Notification

	query := db.Where("dapur_id = ?", req.DapurID)

	if req.UserID > 0 {
		query = query.Where("user_id = ? OR user_id IS NULL", req.UserID)
	}

	if req.Type != "" {
		query = query.Where("type = ?", req.Type)
	}

	if req.IsRead != nil {
		query = query.Where("is_read = ?", *req.IsRead)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := query.
		Order("created_at DESC, id DESC").
		Limit(req.Size).
		Offset((req.Page - 1) * req.Size).
		Find(&notifications).Error; err != nil {
		return nil, 0, err
	}

	return notifications, total, nil
}

func (r *NotificationRepository) CountUnread(db *gorm.DB, dapurID, userID uint) (int64, error) {
	var count int64
	query := db.Model(&entity.Notification{}).
		Where("dapur_id = ? AND is_read = ?", dapurID, false)

	if userID > 0 {
		query = query.Where("user_id = ? OR user_id IS NULL", userID)
	}

	err := query.Count(&count).Error
	return count, err
}

func (r *NotificationRepository) MarkRead(db *gorm.DB, dapurID, notificationID uint, userID uint) error {
	query := db.Model(&entity.Notification{}).
		Where("id = ? AND dapur_id = ? AND is_read = false", notificationID, dapurID)

	if userID > 0 {
		query = query.Where("user_id = ? OR user_id IS NULL", userID)
	}

	return query.Update("is_read", true).Error
}

func (r *NotificationRepository) MarkAllRead(db *gorm.DB, dapurID, userID uint) error {
	query := db.Model(&entity.Notification{}).
		Where("dapur_id = ? AND is_read = false", dapurID)

	if userID > 0 {
		query = query.Where("user_id = ? OR user_id IS NULL", userID)
	}

	return query.Update("is_read", true).Error
}

func (r *NotificationRepository) Delete(db *gorm.DB, dapurID, notificationID uint, userID uint) error {
	query := db.Where("id = ? AND dapur_id = ?", notificationID, dapurID)

	if userID > 0 {
		query = query.Where("user_id = ? OR user_id IS NULL", userID)
	}

	return query.Delete(&entity.Notification{}).Error
}
