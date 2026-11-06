package repository

import (
	"fmt"
	"strconv"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/ridwanmuh3/simba-server/internal/entity"
)

// Document sequence (invoice / PO numbers) persistence. All reads/writes
// are tenant-scoped and atomic via ON CONFLICT upsert — two concurrent
// first-time requests can no longer both INSERT and hit the
// uidx_setting_key_dapur unique violation.
const docSeqLockKeyBase = int64(7291000)

// LockDocumentSequence serializes sequence allocation per dapur for the
// caller's transaction (pg_advisory_xact_lock).
func LockDocumentSequence(db *gorm.DB, dapurID uint) error {
	return db.Exec("SELECT pg_advisory_xact_lock(?)", docSeqLockKeyBase+int64(dapurID)).Error
}

// ReadSeq returns the stored sequence value for key, if present.
func ReadSeq(db *gorm.DB, key string, dapurID uint) (int, bool) {
	var setting entity.AppSetting
	if err := db.Scopes(Tenant(dapurID)).Where("key = ?", key).First(&setting).Error; err != nil {
		return 0, false
	}
	v, err := strconv.Atoi(strings.TrimSpace(setting.Value))
	if err != nil {
		return 0, false
	}
	return v, true
}

// UpsertSeq atomically sets the sequence value for key.
func UpsertSeq(db *gorm.DB, key string, dapurID uint, value int) error {
	return db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "key"}, {Name: "dapur_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"value", "updated_at"}),
	}).Create(&entity.AppSetting{Key: key, Value: fmt.Sprintf("%d", value), DapurID: dapurID}).Error
}

// MaxDocumentSequence derives the max invoice/PO sequence from existing
// documents (seed fallback when no sequence row exists yet).
func MaxDocumentSequence(db *gorm.DB, dapurID uint) (maxInv, maxPo int, err error) {
	var invoices []entity.Invoice
	if err = db.Model(new(entity.Invoice)).
		Select("invoice_number, po_number").
		Scopes(Tenant(dapurID)).
		Find(&invoices).Error; err != nil {
		return 0, 0, err
	}
	for _, invoice := range invoices {
		if seq, ok := ParseDocumentSequence(invoice.InvoiceNumber, "INV"); ok && seq > maxInv {
			maxInv = seq
		}
		if seq, ok := ParseDocumentSequence(invoice.PONumber, "PO"); ok && seq > maxPo {
			maxPo = seq
		}
	}
	return maxInv, maxPo, nil
}

// ParseDocumentSequence extracts the numeric suffix of "INV-042"/"PO-007".
func ParseDocumentSequence(value, prefix string) (int, bool) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return 0, false
	}
	parts := strings.Split(trimmed, "-")
	if len(parts) < 2 {
		return 0, false
	}
	if !strings.EqualFold(parts[0], prefix) {
		return 0, false
	}
	seq, err := strconv.Atoi(parts[len(parts)-1])
	if err != nil {
		return 0, false
	}
	return seq, true
}
