package main

import (
	"context"
	"fmt"
	"time"

	"github.com/ridwanmuh3/simba-server/internal/config"
	"github.com/ridwanmuh3/simba-server/internal/entity"
	"github.com/ridwanmuh3/simba-server/internal/model"
	"github.com/ridwanmuh3/simba-server/internal/repository"
	"github.com/ridwanmuh3/simba-server/internal/service"
	"github.com/ridwanmuh3/simba-server/internal/util"
)

func main() {
	viperConfig := config.NewViper()
	log := config.NewLogger(viperConfig)
	db := config.NewDB(viperConfig, log)
	validate := config.NewValidator(viperConfig)

	userRepo := repository.NewUserRepository()
	userService := service.NewUserService(db, log, validate, userRepo)

	// ──────────────────────────────────────────────
	// 1. Seed Users
	// ──────────────────────────────────────────────
	users := []*model.CreateUserRequest{
		{Username: "superadmin1", Password: "superadmin1", Fullname: "Budi Almaliki", Role: "Super Admin"},
		{Username: "admin1", Password: "admin1", Fullname: "Andi Wijaya", Role: "Admin"},
		{Username: "admin2", Password: "admin2", Fullname: "Aulia Rahman", Role: "Admin"},
		{Username: "admin3", Password: "admin3", Fullname: "Farah Nabila", Role: "Admin"},
		{Username: "admin4", Password: "admin4", Fullname: "Ilham Akbar", Role: "Admin"},
		{Username: "admin5", Password: "admin5", Fullname: "Rafi Kurniawan", Role: "Admin"},
		{Username: "admin6", Password: "admin6", Fullname: "Dimas Saputra", Role: "Admin"},
		{Username: "admin7", Password: "admin7", Fullname: "Naufal Hakim", Role: "Admin"},
		{Username: "admin8", Password: "admin8", Fullname: "Hendra Gunawan", Role: "Admin"},
		{Username: "admin9", Password: "admin9", Fullname: "Wahyu Setiawan", Role: "Admin"},
		{Username: "admin10", Password: "admin10", Fullname: "Bayu Prakoso", Role: "Admin"},
	}

	for _, user := range users {
		_, err := userService.Create(context.Background(), user)
		if err != nil {
			log.Warnf("user %s may already exist: %v", user.Username, err)
		}
	}
	log.Info("seeding users done")

	// ──────────────────────────────────────────────
	// 2. Seed Items (12 bahan)
	// ──────────────────────────────────────────────
	now := time.Now()
	baseDate := now.AddDate(0, -1, 0) // 1 month ago
	categories := []string{
		"Bahan Pokok", "Bahan Tambahan", "Minuman", "Kemasan",
	}

	units := []string{"Kg", "Liter", "Butir", "Pcs"}

	var items []entity.Item

	for i := 1; i <= 120; i++ {
		price := float64(5000 + (i * 1000))

		item := entity.Item{
			ID:           fmt.Sprintf("MBG-BHN-%04d", i),
			Name:         fmt.Sprintf("Bahan-%d", i),
			Category:     categories[i%len(categories)],
			InitialStock: float64(50 + i%100),
			Stock:        float64(50 + i%100),
			MeasureUnit:  units[i%len(units)],
			UnitPrice:    price,
			TotalPrice:   price * float64(50+i%100),
			ModifiedBy:   "System Seeder",
			CreatedAt:    baseDate.AddDate(0, 0, -i),
			DapurID:      1,
		}

		items = append(items, item)
	}

	for i := range items {
		if err := db.Create(&items[i]).Error; err != nil {
			log.Warnf("item %s may already exist: %v", items[i].ID, err)
		}
	}
	log.Info("seeding items done")

	// ──────────────────────────────────────────────
	// 3. Seed Stock Tracks (IN and OUT)
	//    Apply stock changes to items as we go.
	// ──────────────────────────────────────────────
	type stockSeed struct {
		ItemID    string
		Type      string
		Amount    float64
		UnitPrice float64
		Supplier  string
		DaysAgo   int
	}
	var stockSeeds []stockSeed

	for i := 0; i < 300; i++ {
		item := items[i%len(items)]

		isIn := i%2 == 0

		stockSeeds = append(stockSeeds, stockSeed{
			ItemID:    item.ID,
			Type:      map[bool]string{true: "IN", false: "OUT"}[isIn],
			Amount:    float64(5 + i%20),
			UnitPrice: item.UnitPrice + float64(i%5000),
			Supplier:  fmt.Sprintf("Supplier-%d", i%10),
			DaysAgo:   i % 30,
		})
	}
	// Build a map of current item state for running calculations
	itemMap := make(map[string]*entity.Item)
	for i := range items {
		itemCopy := items[i]
		itemMap[itemCopy.ID] = &itemCopy
	}

	for _, ss := range stockSeeds {
		item, ok := itemMap[ss.ItemID]
		if !ok {
			log.Warnf("item %s not found in map, skipping", ss.ItemID)
			continue
		}

		st := entity.StockTracking{
			Type:          ss.Type,
			Amount:        ss.Amount,
			PreviousStock: item.Stock,
			UnitPrice:     ss.UnitPrice,
			TotalPrice:    util.Round2(ss.Amount * ss.UnitPrice),
			Supplier:      ss.Supplier,
			ModifiedBy:    "Budi Almaliki",
			ItemID:        ss.ItemID,
			DapurID:       1,
		}

		switch ss.Type {
		case "IN":
			newStock := util.Round4(item.Stock + ss.Amount)
			addedTotal := util.Round2(ss.Amount * ss.UnitPrice)
			st.NewStock = newStock
			st.TotalPrice = util.Round2(ss.Amount * ss.UnitPrice)
			item.Stock = newStock
			item.TotalPrice = util.Round2(item.TotalPrice + addedTotal)
		case "OUT":
			if item.Stock < ss.Amount {
				log.Warnf("skipping OUT for %s: insufficient stock (%.2f < %.2f)", ss.ItemID, item.Stock, ss.Amount)
				continue
			}
			newStock := util.Round4(item.Stock - ss.Amount)
			avg := item.TotalPrice / item.Stock
			deduction := util.Round2(avg * ss.Amount)
			st.NewStock = newStock
			st.TotalPrice = util.Round2(ss.Amount * ss.UnitPrice)
			item.Stock = newStock
			item.TotalPrice = util.Round2(item.TotalPrice - deduction)
		}

		// Set created_at via raw SQL after creation
		createdAt := now.AddDate(0, 0, -ss.DaysAgo)

		var existingCount int64
		db.Model(&entity.StockTracking{}).
			Where("item_id = ? AND type = ? AND amount = ? AND supplier = ? AND created_at = ?",
				ss.ItemID, ss.Type, ss.Amount, ss.Supplier, createdAt).
			Count(&existingCount)

		if err := db.Create(&st).Error; err != nil {
			log.Warnf("stock track for %s failed: %v", ss.ItemID, err)
			continue
		}
		db.Model(&entity.StockTracking{}).Where("id = ?", st.ID).Update("created_at", createdAt)
	}

	// Update items with final stock values
	for _, item := range itemMap {
		db.Model(&entity.Item{}).Where("id = ?", item.ID).Updates(map[string]any{
			"stock":       item.Stock,
			"total_price": item.TotalPrice,
		})
	}
	log.Info("seeding stock tracks done")

	// ──────────────────────────────────────────────
	// 4. Seed Finances (12 records)
	// ──────────────────────────────────────────────
	// Normalize old seed values ("in"/"out") to match app convention.
	db.Model(&entity.Finance{}).Where("type = ?", "in").Update("type", "PEMASUKAN")
	db.Model(&entity.Finance{}).Where("type = ?", "out").Update("type", "PENGELUARAN")
	categoriesFinance := []string{
		"Penjualan", "Pembelian Bahan", "Operasional", "Gaji",
	}

	var finances []entity.Finance

	for i := 1; i <= 120; i++ {

		isIncome := i%2 == 0

		amount := 100000 + (i * 50000)

		f := entity.Finance{
			Type:        map[bool]string{true: "PEMASUKAN", false: "PENGELUARAN"}[isIncome],
			Category:    categoriesFinance[i%len(categoriesFinance)],
			Description: fmt.Sprintf("Transaksi ke-%d", i),
			Amount:      amount,
			ExtraNote:   fmt.Sprintf("Auto generated #%d", i),
			ProofImage:  "/uploads/finances-proof/seed-proof-1.png",
			ModifiedBy:  "System Seeder",
			DapurID:     1,
		}

		// spread dates (important for charts)
		f.Model.CreatedAt = now.AddDate(0, 0, -(i % 60))

		finances = append(finances, f)
	}

	for i, f := range finances {
		f.Model.CreatedAt = now.AddDate(0, 0, -(28 - i*2))
		var existingCount int64
		db.Model(&entity.Finance{}).
			Where("description = ? AND amount = ? AND type = ?", f.Description, f.Amount, f.Type).
			Count(&existingCount)
		if err := db.Create(&f).Error; err != nil {
			log.Warnf("finance record %d failed: %v", i+1, err)
		}
	}
	log.Info("seeding finances done")

	// ──────────────────────────────────────────────
	// 5. Seed Invoices (6 records)
	// ──────────────────────────────────────────────
	db.Exec("UPDATE invoices SET invoice_number = regexp_replace(invoice_number, '^INV-[0-9]{4}-', 'INV-') WHERE invoice_number ~ '^INV-[0-9]{4}-'")
	db.Exec("UPDATE invoices SET po_number = regexp_replace(po_number, '^PO-[0-9]{4}-', 'PO-') WHERE po_number ~ '^PO-[0-9]{4}-'")
	db.Exec("UPDATE invoices SET kebutuhan = regexp_replace(kebutuhan, '^QUO-[0-9]{4}-', 'QUO-') WHERE kebutuhan ~ '^QUO-[0-9]{4}-'")

	invoices := []entity.Invoice{
		{
			StockType:      "OUT",
			CompanyName:    "CV Sinar Pangan",
			CompanyContact: "0812-3456-7890",
			CompanyAddress: "Jl. Melati No. 12, Bandung",
			InvoiceNumber:  "INV-0001",
			PONumber:       "PO-0101",
			Kebutuhan:      "QUO-0091",
		},
		{
			StockType:      "OUT",
			CompanyName:    "PT Rasa Nusantara",
			CompanyContact: "0813-2222-3333",
			CompanyAddress: "Jl. Sudirman No. 88, Jakarta",
			InvoiceNumber:  "INV-0002",
			PONumber:       "PO-0102",
			Kebutuhan:      "QUO-0092",
		},
		{
			StockType:      "OUT",
			CompanyName:    "UD Kue Manis",
			CompanyContact: "0812-7788-9900",
			CompanyAddress: "Jl. Diponegoro No. 45, Surabaya",
			InvoiceNumber:  "INV-0003",
			PONumber:       "PO-0103",
			Kebutuhan:      "QUO-0093",
		},
		{
			StockType:      "IN",
			CompanyName:    "PT Bumi Boga",
			CompanyContact: "0812-9090-1234",
			CompanyAddress: "Jl. Gatot Subroto No. 7, Semarang",
			InvoiceNumber:  "INV-0004",
			PONumber:       "PO-0104",
			Kebutuhan:      "QUO-0094",
		},
		{
			StockType:      "OUT",
			CompanyName:    "CV Manis Jaya",
			CompanyContact: "0821-4567-8910",
			CompanyAddress: "Jl. Ahmad Yani No. 23, Yogyakarta",
			InvoiceNumber:  "INV-0005",
			PONumber:       "PO-0105",
			Kebutuhan:      "QUO-0095",
		},
		{
			StockType:      "OUT",
			CompanyName:    "PT Sentra Bakery",
			CompanyContact: "0813-5555-6677",
			CompanyAddress: "Jl. Asia Afrika No. 99, Bandung",
			InvoiceNumber:  "INV-0006",
			PONumber:       "PO-0106",
			Kebutuhan:      "QUO-0096",
		},
	}

	for i := range invoices {
		invoices[i].CreatedAt = now.AddDate(0, 0, -(12 - i*2))
		var existing entity.Invoice
		if err := db.Where("invoice_number = ?", invoices[i].InvoiceNumber).First(&existing).Error; err == nil {
			log.Warnf("invoice %s may already exist", invoices[i].InvoiceNumber)
			continue
		}
		if err := db.Create(&invoices[i]).Error; err != nil {
			log.Warnf("invoice record %d failed: %v", i+1, err)
		}
	}
	log.Info("seeding invoices done")

	fmt.Println("✅ All seed data inserted successfully!")
}
