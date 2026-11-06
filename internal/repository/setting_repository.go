package repository

import (
	"github.com/ridwanmuh3/simba-server/internal/entity"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type SettingRepository struct {
	Repository[entity.AppSetting]
}

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
