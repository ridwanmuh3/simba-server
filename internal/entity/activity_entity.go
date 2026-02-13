package entity

import (
	"gorm.io/gorm"
)

type ActivityLog struct {
	gorm.Model
	Type        string `gorm:"size:30;not null"`
	Title       string `gorm:"size:100;not null"`
	Description string `gorm:"size:100;not null"`
	ActionBy    string `gorm:"size:100;not null"`
}

func (l *ActivityLog) TableName() string {
	return "activity_logs"
}

func (l *ActivityLog) AfterCreate(tx *gorm.DB) (err error) {
	var ids []uint

	if err = tx.Model(l).
		Order("created_at ASC").
		Offset(4).
		Pluck("id", &ids).Error; err != nil {
		return
	}

	if len(ids) > 0 {
		if err = tx.Where("id IN ?", ids).Delete(l).Error; err != nil {
			return
		}
	}

	return
}
