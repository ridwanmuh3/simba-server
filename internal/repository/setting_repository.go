package repository

import (
	"fmt"

	"github.com/ridwanmuh3/simba-server/internal/entity"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type SettingRepository struct{}

func NewSettingRepository() *SettingRepository {
	return &SettingRepository{}
}

func (r *SettingRepository) GetByKey(db *gorm.DB, key string, dapurID uint) (*entity.AppSetting, error) {
	var setting entity.AppSetting
	err := db.Where("key = ? AND dapur_id = ?", key, dapurID).First(&setting).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &setting, nil
}

func (r *SettingRepository) Save(db *gorm.DB, setting *entity.AppSetting) error {
	return db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "key"}, {Name: "dapur_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"value", "updated_at"}),
	}).Create(setting).Error
}

// IncrementSequence returns incremented value safely, scoped to a dapur.
func (r *SettingRepository) IncrementSequence(db *gorm.DB, key string, dapurID uint) (int, error) {
	var value int
	err := db.Transaction(func(tx *gorm.DB) error {
		var setting entity.AppSetting
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("key = ? AND dapur_id = ?", key, dapurID).
			First(&setting).Error

		if err != nil && err != gorm.ErrRecordNotFound {
			return err
		}

		if err == gorm.ErrRecordNotFound {
			setting = entity.AppSetting{Key: key, Value: "1", DapurID: dapurID}
			value = 1
		} else {
			var currentSeq int
			_, _ = fmt.Sscanf(setting.Value, "%d", &currentSeq)
			value = currentSeq + 1
			setting.Value = fmt.Sprintf("%d", value)
		}

		return tx.Save(&setting).Error
	})
	return value, err
}
