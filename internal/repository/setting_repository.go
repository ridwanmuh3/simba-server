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

func (r *SettingRepository) GetByKey(db *gorm.DB, key string) (*entity.AppSetting, error) {
	var setting entity.AppSetting
	err := db.Where("key = ?", key).First(&setting).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil // Not found is not an error
		}
		return nil, err
	}
	return &setting, nil
}

func (r *SettingRepository) Save(db *gorm.DB, setting *entity.AppSetting) error {
	return db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "key"}},
		DoUpdates: clause.AssignmentColumns([]string{"value", "updated_at"}),
	}).Create(setting).Error
}

// IncrementAndGet Sequence returns the incremented value safely
func (r *SettingRepository) IncrementSequence(db *gorm.DB, key string) (int, error) {
	var value int
	err := db.Transaction(func(tx *gorm.DB) error {
		var setting entity.AppSetting
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("key = ?", key).First(&setting).Error

		if err != nil && err != gorm.ErrRecordNotFound {
			return err
		}

		if err == gorm.ErrRecordNotFound {
			setting = entity.AppSetting{Key: key, Value: "1"}
			value = 1
		} else {
			// parse and increment
			var currentSeq int
			_, _ = fmt.Sscanf(setting.Value, "%d", &currentSeq)
			value = currentSeq + 1
			setting.Value = fmt.Sprintf("%d", value)
		}

		return tx.Save(&setting).Error
	})
	return value, err
}
