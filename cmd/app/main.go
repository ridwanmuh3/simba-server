package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ridwanmuh3/simba-server/internal/config"
)

func main() {
	viperConfig := config.NewViper()
	log := config.NewLogger(viperConfig)
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

	go func() {
		if err := app.Listen(fmt.Sprintf(":%d", appPort)); err != nil {
			log.Fatalf("failed to start app server: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Warn("shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := app.ShutdownWithContext(ctx); err != nil {
		log.Fatalf("server forced to shutdown: %v", err)
	}

	log.Info("server exiting")

	if err := log.Sync(); err != nil {
		return
	}
}
