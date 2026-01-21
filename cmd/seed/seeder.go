package main

import (
	"context"

	"github.com/ridwanmuh3/simba-server/internal/config"
	"github.com/ridwanmuh3/simba-server/internal/model"
	"github.com/ridwanmuh3/simba-server/internal/repository"
	"github.com/ridwanmuh3/simba-server/internal/service"
)

func main() {
	log := config.NewLogger()
	viperConfig := config.NewViper()
	db := config.NewDB(viperConfig, log)
	validate := config.NewValidator(viperConfig)
	userRepo := repository.NewUserRepository()
	userService := service.NewUserService(db, log, validate, userRepo)

	users := []*model.CreateUserRequest{
		{
			Username: "superadmin2",
			Password: "superadmin2",
			Fullname: "Budi Santoso",
			Role:     "Super Admin",
		},
		{
			Username: "superadmin3",
			Password: "superadmin3",
			Fullname: "Dewi Lestari",
			Role:     "Super Admin",
		},
		{
			Username: "superadmin4",
			Password: "superadmin4",
			Fullname: "Rizky Pratama",
			Role:     "Super Admin",
		},
		{
			Username: "superadmin5",
			Password: "superadmin5",
			Fullname: "Siti Aisyah",
			Role:     "Super Admin",
		},

		{
			Username: "admin10",
			Password: "admin10",
			Fullname: "Andi Wijaya",
			Role:     "Admin",
		},
		{
			Username: "admin11",
			Password: "admin11",
			Fullname: "Agus Salim",
			Role:     "Admin",
		},
		{
			Username: "admin12",
			Password: "admin12",
			Fullname: "Rina Marlina",
			Role:     "Admin",
		},
		{
			Username: "admin13",
			Password: "admin13",
			Fullname: "Muhammad Fajar",
			Role:     "Admin",
		},
		{
			Username: "admin14",
			Password: "admin14",
			Fullname: "Nadia Putri",
			Role:     "Admin",
		},
		{
			Username: "admin15",
			Password: "admin15",
			Fullname: "Fikri Ramadhan",
			Role:     "Admin",
		},
		{
			Username: "admin16",
			Password: "admin16",
			Fullname: "Intan Permata",
			Role:     "Admin",
		},
		{
			Username: "admin17",
			Password: "admin17",
			Fullname: "Yoga Prakoso",
			Role:     "Admin",
		},
		{
			Username: "admin18",
			Password: "admin18",
			Fullname: "Putra Mahendra",
			Role:     "Admin",
		},
		{
			Username: "admin19",
			Password: "admin19",
			Fullname: "Taufik Hidayat",
			Role:     "Admin",
		},
		{
			Username: "admin20",
			Password: "admin20",
			Fullname: "Aulia Rahman",
			Role:     "Admin",
		},
		{
			Username: "admin21",
			Password: "admin21",
			Fullname: "Farah Nabila",
			Role:     "Admin",
		},
		{
			Username: "admin22",
			Password: "admin22",
			Fullname: "Ilham Akbar",
			Role:     "Admin",
		},
		{
			Username: "admin23",
			Password: "admin23",
			Fullname: "Rafi Kurniawan",
			Role:     "Admin",
		},
		{
			Username: "admin24",
			Password: "admin24",
			Fullname: "Dimas Saputra",
			Role:     "Admin",
		},
		{
			Username: "admin25",
			Password: "admin25",
			Fullname: "Naufal Hakim",
			Role:     "Admin",
		},
		{
			Username: "admin26",
			Password: "admin26",
			Fullname: "Hendra Gunawan",
			Role:     "Admin",
		},
		{
			Username: "admin27",
			Password: "admin27",
			Fullname: "Wahyu Setiawan",
			Role:     "Admin",
		},
		{
			Username: "admin28",
			Password: "admin28",
			Fullname: "Ardiansyah Putra",
			Role:     "Admin",
		},
		{
			Username: "admin29",
			Password: "admin29",
			Fullname: "Bayu Prakoso",
			Role:     "Admin",
		},
	}

	for _, user := range users {
		userService.Create(context.Background(), user)
	}

	log.Info("seeding test user success")
}
