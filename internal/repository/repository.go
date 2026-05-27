package repository

import (
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Repository[T any] struct{}

func (r *Repository[T]) Save(db *gorm.DB, entity *T) error {
	return db.Save(entity).Error
}

func (r *Repository[T]) Delete(db *gorm.DB, entity *T) error {
	return db.Unscoped().Delete(entity).Error
}

func (r *Repository[T]) FindById(db *gorm.DB, entity *T, id any) error {
	return db.Where("id = ?", id).Take(entity).Error
}

func OrderBy() clause.OrderBy {
	return clause.OrderBy{Columns: []clause.OrderByColumn{
		{Column: clause.Column{Name: "created_at"}, Desc: true},
		{Column: clause.Column{Name: "id"}, Desc: true},
	}}
}
