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

	// Drop old single-column unique indexes on invoices — replaced by
	// composite (stock_type, *) indexes to allow same number across stock types.
	oldIndexes := []string{
		"idx_invoices_invoice_number",
		"idx_invoice_po_number",
		"idx_invoice_quo_number",
	}
	for _, idx := range oldIndexes {
		if db.Migrator().HasIndex(&entity.Invoice{}, idx) {
			if err := db.Migrator().DropIndex(&entity.Invoice{}, idx); err != nil {
				log.Warnf("could not drop index %s: %v", idx, err)
			}
		}
	}

	if err := db.AutoMigrate(
		&entity.Dapur{},
		&entity.User{},
		&entity.Item{},
		&entity.StockTracking{},
		&entity.Finance{},
		&entity.ActivityLog{},
		&entity.AppSetting{},
		&entity.Invoice{},
		&entity.InvoiceItem{},
	); err != nil {
		log.Fatalf("migration failed: %v", err)
	}

	log.Info("migration complete")
}
