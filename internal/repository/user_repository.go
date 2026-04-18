package repository

import (
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/ridwanmuh3/simba-server/internal/entity"
	"github.com/ridwanmuh3/simba-server/internal/model"
)

type UserRepository struct {
	Repository[entity.User]
}

func NewUserRepository() *UserRepository {
	return &UserRepository{}
}

func (r *UserRepository) CountByUsername(db *gorm.DB, username string) (int64, error) {
	var count int64
	err := db.Model(new(entity.User)).Where("username = ?", username).Count(&count).Error
	return count, err
}

func (r *UserRepository) FindByToken(db *gorm.DB, user *entity.User, token string) error {
	return db.Where("token = ?", token).Take(user).Error
}

func (r *UserRepository) FindByRefreshToken(db *gorm.DB, user *entity.User, token string) error {
	return db.Where("refresh_token = ?", token).Take(user).Error
}

func (r *UserRepository) FindByUsername(db *gorm.DB, user *entity.User, username string) error {
	return db.Where("username = ?", username).Take(user).Error
}

func (r *UserRepository) FindAll(db *gorm.DB, query *model.FindAllUserRequest) ([]entity.User, int64, error) {
	var users []entity.User

	orderBy := clause.OrderBy{Columns: []clause.OrderByColumn{
		{Column: clause.Column{Name: "created_at"}, Desc: true},
		{Column: clause.Column{Name: "is_active"}, Desc: true},
	}}

	if err := db.Scopes(r.FilterUser(query)).Offset((query.Page - 1) * query.Size).Limit(query.Size).Order(orderBy).Find(&users).Error; err != nil {
		return nil, 0, err
	}

	var total int64
	if err := db.Model(new(entity.User)).Scopes(r.FilterUser(query)).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	return users, total, nil
}

func (r *UserRepository) GetUsersStats(db *gorm.DB) (*entity.UserStats, error) {
	var stats entity.UserStats

	err := db.Model(new(entity.User)).
		Select(`
            COUNT(*) as total,
            COUNT(CASE WHEN role = 'Super Admin' THEN 1 END) as super_admin,
            COUNT(CASE WHEN role = 'Admin' THEN 1 END) as admin,
            COUNT(CASE WHEN is_active = true THEN 1 END) as active
        `).
		Scan(&stats).Error
	if err != nil {
		return nil, err
	}

	return &stats, nil
}

func (r *UserRepository) FilterUser(query *model.FindAllUserRequest) func(tx *gorm.DB) *gorm.DB {
	return func(tx *gorm.DB) *gorm.DB {
		if fullname := query.Fullname; fullname != "" {
			fullname = "%" + fullname + "%"
			tx = tx.Where("fullname LIKE ?", fullname)
		}

		return tx
	}
}
