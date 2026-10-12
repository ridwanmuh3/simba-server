package service

import (
	"fmt"
	"strings"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/ridwanmuh3/simba-server/internal/entity"
	"github.com/ridwanmuh3/simba-server/internal/model"
	"github.com/ridwanmuh3/simba-server/internal/transport/websocket"
)

type LowStockChecker struct {
	db    *gorm.DB
	hub   *websocket.Hub
	log   *zap.SugaredLogger
	email *EmailService
}

type LowStockCheckerRepository interface {
	FindThreshold(db *gorm.DB, dapurID uint) (float64, error)
}

func NewLowStockChecker(db *gorm.DB, hub *websocket.Hub, log *zap.SugaredLogger, email *EmailService) *LowStockChecker {
	return &LowStockChecker{
		db:    db,
		hub:   hub,
		log:   log,
		email: email,
	}
}

func (c *LowStockChecker) Check(item *entity.Item, dapurID uint) {
	threshold := item.LowStockThreshold

	if item.Stock <= threshold && threshold > 0 {
		metaJSON := fmt.Sprintf(`{"item_id":"%s","current_stock":%.4f,"threshold":%.4f}`, item.ID, item.Stock, threshold)

		notif := &entity.Notification{
			Type:    "LOW_STOCK",
			Title:   fmt.Sprintf("Stok %s rendah", item.Name),
			Message: fmt.Sprintf("Stok %s tinggal %.2f %s (threshold: %.2f)", item.Name, item.Stock, item.MeasureUnit, threshold),
			Level:   "warning",
			DapurID: dapurID,
			MetaJSON: metaJSON,
		}

		if err := c.db.Create(notif).Error; err != nil {
			c.log.Errorw("failed to save low stock notification", "error", err, "item_id", item.ID)
			return
		}

		if c.hub != nil {
			payload := model.EventPayload[any]{
				Type:    "LOW_STOCK",
				Message: notif.Title,
				Data: map[string]any{
					"item_id":       item.ID,
					"item_name":     item.Name,
					"current_stock": item.Stock,
					"threshold":     threshold,
					"measure_unit":  item.MeasureUnit,
				},
			}
			c.hub.BroadcastToDapur(dapurID, payload)
		}

		c.sendEmailAlert(item, dapurID, threshold)
	}
}

// sendEmailAlert notifies Super Admin users via email when stock drops below threshold.
func (c *LowStockChecker) sendEmailAlert(item *entity.Item, dapurID uint, threshold float64) {
	if c.email == nil || !c.email.IsConfigured() {
		return
		}

	var adminEmails []string
	err := c.db.Model(&entity.User{}).
		Where("role = ? AND deleted_at IS NULL", "Super Admin").
		Pluck("username", &adminEmails).Error
	if err != nil {
		c.log.Errorw("failed to fetch admin emails for low stock alert", "error", err)
		return
	}

	// Filter: only use usernames that look like email addresses.
	var recipients []string
	for _, e := range adminEmails {
		e = strings.TrimSpace(e)
		if strings.Contains(e, "@") {
			recipients = append(recipients, e)
		}
	}

	if len(recipients) == 0 {
		c.log.Warnw("no admin email recipients for low stock alert", "item_id", item.ID)
		return
	}

	subject := fmt.Sprintf("[SIMBA] Peringatan Stok Rendah: %s", item.Name)
	body := fmt.Sprintf(
		"Stok item %s di dapur %d telah turun di bawah ambang batas.\n\n"+
			"Item: %s (%s)\n"+
			"Stok saat ini: %.2f %s\n"+
			"Ambang batas: %.2f\n\n"+
			"Segera lakukan pemesanan ulang.",
		item.Name, dapurID,
		item.ID, item.Name,
		item.Stock, item.MeasureUnit,
		threshold,
	)

	if err := c.email.Send(subject, body, recipients); err != nil {
		c.log.Errorw("failed to send low stock email", "error", err, "item_id", item.ID)
	}
}
