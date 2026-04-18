package main

import (
	"github.com/ridwanmuh3/simba-server/internal/config"
	"github.com/ridwanmuh3/simba-server/internal/entity"
)

func main() {
	viperConfig := config.NewViper()
	log := config.NewLogger(viperConfig)
	db := config.NewDB(viperConfig, log)

	log.Info("running schema migration...")

	if err := db.AutoMigrate(
		&entity.User{},
		&entity.Item{},
		&entity.StockTracking{},
		&entity.Finance{},
		&entity.ActivityLog{},
		&entity.AppSetting{},
		&entity.Invoice{},
	); err != nil {
		log.Fatalf("migration failed: %v", err)
	}

	log.Info("migration complete")
}
