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
			Username: "superadmin1",
			Password: "superadmin1",
			Fullname: "Alex Komarudin",
			Role:     "Super Admin",
		},
		{
			Username: "admin1",
			Password: "admin1",
			Fullname: "Alex Samsudin",
			Role:     "Admin",
		},
	}

	userService.Create(context.Background(), users[0])
	userService.Create(context.Background(), users[1])

	log.Info("seeding test user success")
}
