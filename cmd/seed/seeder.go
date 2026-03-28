package main

import (
	"context"

	"github.com/ridwanmuh3/simba-server/internal/config"
	"github.com/ridwanmuh3/simba-server/internal/entity"
	"github.com/ridwanmuh3/simba-server/internal/model"
	"github.com/ridwanmuh3/simba-server/internal/repository"
	"github.com/ridwanmuh3/simba-server/internal/service"
)

func main() {
	viperConfig := config.NewViper()
	log := config.NewLogger(viperConfig)
	db := config.NewDB(viperConfig, log)
	validate := config.NewValidator(viperConfig)

	if err := db.AutoMigrate(
		&entity.User{},
		&entity.Item{},
		&entity.StockTracking{},
		&entity.Finance{},
		&entity.ActivityLog{},
	); err != nil {
		log.Fatalf("failed to auto migrate: %v", err)
	}
	log.Info("auto migrate success")

	userRepo := repository.NewUserRepository()
	userService := service.NewUserService(db, log, validate, userRepo)

	users := []*model.CreateUserRequest{
		{
			Username: "superadmin1",
			Password: "superadmin1",
			Fullname: "Budi Almaliki",
			Role:     "Super Admin",
		},
		{
			Username: "admin1",
			Password: "admin10",
			Fullname: "Andi Wijaya",
			Role:     "Admin",
		},
		{
			Username: "admin2",
			Password: "admin2",
			Fullname: "Aulia Rahman",
			Role:     "Admin",
		},
		{
			Username: "admin3",
			Password: "admin3",
			Fullname: "Farah Nabila",
			Role:     "Admin",
		},
		{
			Username: "admin4",
			Password: "admin4",
			Fullname: "Ilham Akbar",
			Role:     "Admin",
		},
		{
			Username: "admin5",
			Password: "admin5",
			Fullname: "Rafi Kurniawan",
			Role:     "Admin",
		},
		{
			Username: "admin6",
			Password: "admin6",
			Fullname: "Dimas Saputra",
			Role:     "Admin",
		},
		{
			Username: "admin7",
			Password: "admin7",
			Fullname: "Naufal Hakim",
			Role:     "Admin",
		},
		{
			Username: "admin8",
			Password: "admin26",
			Fullname: "Hendra Gunawan",
			Role:     "Admin",
		},
		{
			Username: "admin9",
			Password: "admin9",
			Fullname: "Wahyu Setiawan",
			Role:     "Admin",
		},
		{
			Username: "admin10",
			Password: "admin10",
			Fullname: "Bayu Prakoso",
			Role:     "Admin",
		},
	}

	var err any
	for _, user := range users {
		_, err = userService.Create(context.Background(), user)
	}

	if err == nil {
		log.Info("seeding test user success")
	}
}
