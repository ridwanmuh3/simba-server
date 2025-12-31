package main

import (
	"fmt"

	"github.com/ridwanmuh3/simba-server/internal/config"
)

func main() {
	viperConfig := config.NewViper()
	log := config.NewLogger()
	db := config.NewDB(viperConfig, log)
	validate := config.NewValidator(viperConfig)
	app := config.NewFiber(viperConfig)

	config.Bootstrap(&config.BootstrapConfig{
		DB:       db,
		App:      app,
		Log:      log,
		Validate: validate,
		Config:   viperConfig,
	})

	appPort := viperConfig.GetInt("APP_PORT")
	if err := app.Listen(fmt.Sprintf(":%d", appPort)); err != nil {
		log.Fatalf("failed to start app server: %v", err)
	}

	defer log.Sync()
}
