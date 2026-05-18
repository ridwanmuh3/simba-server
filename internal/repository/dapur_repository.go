package repository

import (
	"github.com/ridwanmuh3/simba-server/internal/entity"
	"gorm.io/gorm"
)

type DapurRepository struct {
	Repository[entity.Dapur]
}

func NewDapurRepository() *DapurRepository {
	return &DapurRepository{}
}

func (r *DapurRepository) FindAll(db *gorm.DB) ([]entity.Dapur, error) {
	var dapurs []entity.Dapur
	err := db.Order("updated_at DESC, created_at DESC, id DESC").Find(&dapurs).Error
	return dapurs, err
}

func (r *DapurRepository) FindByName(db *gorm.DB, name string) (*entity.Dapur, error) {
	var dapur entity.Dapur
	err := db.Where("name = ?", name).First(&dapur).Error
	if err != nil {
		return nil, err
	}
	return &dapur, nil
}

func (r *DapurRepository) CountByName(db *gorm.DB, name string) (int64, error) {
	var count int64
	err := db.Model(new(entity.Dapur)).Where("name = ?", name).Count(&count).Error
	return count, err
}
