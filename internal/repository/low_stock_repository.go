package repository

import (
	"gorm.io/gorm"

	"github.com/ridwanmuh3/simba-server/internal/entity"
)

// LowStockRepository backs the low-stock alert checker.
type LowStockRepository struct{}

func NewLowStockRepository() *LowStockRepository { return &LowStockRepository{} }

// SaveNotification persists a low-stock notification.
func (r *LowStockRepository) SaveNotification(db *gorm.DB, notif *entity.Notification) error {
	return db.Create(notif).Error
}

// FindSuperAdminUsernames returns usernames of active super admins.
func (r *LowStockRepository) FindSuperAdminUsernames(db *gorm.DB) ([]string, error) {
	var emails []string
	err := db.Model(&entity.User{}).
		Where("role = ? AND deleted_at IS NULL", "Super Admin").
		Pluck("username", &emails).Error
	return emails, err
}
