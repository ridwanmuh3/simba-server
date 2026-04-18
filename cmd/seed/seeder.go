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

	items := []entity.Item{
		{ID: "MBG-BHN-0001", Name: "Tepung Terigu", Category: "Bahan Pokok", InitialStock: 100, Stock: 100, MeasureUnit: "Kg", UnitPrice: 12000, TotalPrice: 1200000, ModifiedBy: "Budi Almaliki", CreatedAt: baseDate},
		{ID: "MBG-BHN-0002", Name: "Gula Pasir", Category: "Bahan Pokok", InitialStock: 80, Stock: 80, MeasureUnit: "Kg", UnitPrice: 15000, TotalPrice: 1200000, ModifiedBy: "Budi Almaliki", CreatedAt: baseDate},
		{ID: "MBG-BHN-0003", Name: "Mentega", Category: "Bahan Pokok", InitialStock: 50, Stock: 50, MeasureUnit: "Kg", UnitPrice: 25000, TotalPrice: 1250000, ModifiedBy: "Budi Almaliki", CreatedAt: baseDate},
		{ID: "MBG-BHN-0004", Name: "Telur Ayam", Category: "Bahan Pokok", InitialStock: 200, Stock: 200, MeasureUnit: "Butir", UnitPrice: 2500, TotalPrice: 500000, ModifiedBy: "Budi Almaliki", CreatedAt: baseDate},
		{ID: "MBG-BHN-0005", Name: "Susu Cair", Category: "Minuman", InitialStock: 60, Stock: 60, MeasureUnit: "Liter", UnitPrice: 18000, TotalPrice: 1080000, ModifiedBy: "Budi Almaliki", CreatedAt: baseDate},
		{ID: "MBG-BHN-0006", Name: "Cokelat Bubuk", Category: "Bahan Tambahan", InitialStock: 30, Stock: 30, MeasureUnit: "Kg", UnitPrice: 45000, TotalPrice: 1350000, ModifiedBy: "Budi Almaliki", CreatedAt: baseDate},
		{ID: "MBG-BHN-0007", Name: "Minyak Goreng", Category: "Bahan Pokok", InitialStock: 40, Stock: 40, MeasureUnit: "Liter", UnitPrice: 17000, TotalPrice: 680000, ModifiedBy: "Budi Almaliki", CreatedAt: baseDate},
		{ID: "MBG-BHN-0008", Name: "Keju Cheddar", Category: "Bahan Tambahan", InitialStock: 25, Stock: 25, MeasureUnit: "Kg", UnitPrice: 85000, TotalPrice: 2125000, ModifiedBy: "Budi Almaliki", CreatedAt: baseDate},
		{ID: "MBG-BHN-0009", Name: "Vanili Bubuk", Category: "Bahan Tambahan", InitialStock: 10, Stock: 10, MeasureUnit: "Kg", UnitPrice: 120000, TotalPrice: 1200000, ModifiedBy: "Budi Almaliki", CreatedAt: baseDate},
		{ID: "MBG-BHN-0010", Name: "Garam", Category: "Bahan Pokok", InitialStock: 50, Stock: 50, MeasureUnit: "Kg", UnitPrice: 5000, TotalPrice: 250000, ModifiedBy: "Budi Almaliki", CreatedAt: baseDate},
		{ID: "MBG-BHN-0011", Name: "Baking Powder", Category: "Bahan Tambahan", InitialStock: 20, Stock: 20, MeasureUnit: "Kg", UnitPrice: 35000, TotalPrice: 700000, ModifiedBy: "Budi Almaliki", CreatedAt: baseDate},
		{ID: "MBG-BHN-0012", Name: "Krim Kocok", Category: "Bahan Tambahan", InitialStock: 15, Stock: 15, MeasureUnit: "Liter", UnitPrice: 55000, TotalPrice: 825000, ModifiedBy: "Budi Almaliki", CreatedAt: baseDate},
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

	stockSeeds := []stockSeed{
		// IN records
		{ItemID: "MBG-BHN-0001", Type: "IN", Amount: 50, UnitPrice: 12000, Supplier: "PT Bogasari", DaysAgo: 25},
		{ItemID: "MBG-BHN-0002", Type: "IN", Amount: 30, UnitPrice: 15000, Supplier: "CV Makmur", DaysAgo: 22},
		{ItemID: "MBG-BHN-0003", Type: "IN", Amount: 20, UnitPrice: 25000, Supplier: "PT Blue Band", DaysAgo: 20},
		{ItemID: "MBG-BHN-0004", Type: "IN", Amount: 100, UnitPrice: 2500, Supplier: "Peternakan Jaya", DaysAgo: 18},
		{ItemID: "MBG-BHN-0005", Type: "IN", Amount: 25, UnitPrice: 18000, Supplier: "PT Ultrajaya", DaysAgo: 15},
		{ItemID: "MBG-BHN-0006", Type: "IN", Amount: 10, UnitPrice: 45000, Supplier: "PT Van Houten", DaysAgo: 14},
		{ItemID: "MBG-BHN-0007", Type: "IN", Amount: 20, UnitPrice: 17000, Supplier: "PT Bimoli", DaysAgo: 12},
		{ItemID: "MBG-BHN-0008", Type: "IN", Amount: 10, UnitPrice: 85000, Supplier: "PT Kraft", DaysAgo: 10},
		{ItemID: "MBG-BHN-0009", Type: "IN", Amount: 5, UnitPrice: 120000, Supplier: "CV Rempah Nusantara", DaysAgo: 8},
		{ItemID: "MBG-BHN-0010", Type: "IN", Amount: 30, UnitPrice: 5000, Supplier: "PT Garam Indonesia", DaysAgo: 7},
		{ItemID: "MBG-BHN-0011", Type: "IN", Amount: 10, UnitPrice: 35000, Supplier: "CV Kimia Pangan", DaysAgo: 5},
		{ItemID: "MBG-BHN-0012", Type: "IN", Amount: 8, UnitPrice: 55000, Supplier: "PT Anchor", DaysAgo: 3},
		// OUT records
		{ItemID: "MBG-BHN-0001", Type: "OUT", Amount: 30, UnitPrice: 14000, Supplier: "-", DaysAgo: 20},
		{ItemID: "MBG-BHN-0002", Type: "OUT", Amount: 15, UnitPrice: 17000, Supplier: "-", DaysAgo: 17},
		{ItemID: "MBG-BHN-0003", Type: "OUT", Amount: 10, UnitPrice: 30000, Supplier: "-", DaysAgo: 15},
		{ItemID: "MBG-BHN-0004", Type: "OUT", Amount: 60, UnitPrice: 3000, Supplier: "-", DaysAgo: 13},
		{ItemID: "MBG-BHN-0005", Type: "OUT", Amount: 20, UnitPrice: 22000, Supplier: "-", DaysAgo: 10},
		{ItemID: "MBG-BHN-0006", Type: "OUT", Amount: 8, UnitPrice: 55000, Supplier: "-", DaysAgo: 9},
		{ItemID: "MBG-BHN-0007", Type: "OUT", Amount: 15, UnitPrice: 20000, Supplier: "-", DaysAgo: 7},
		{ItemID: "MBG-BHN-0008", Type: "OUT", Amount: 5, UnitPrice: 100000, Supplier: "-", DaysAgo: 5},
		{ItemID: "MBG-BHN-0001", Type: "IN", Amount: 25, UnitPrice: 12500, Supplier: "PT Bogasari", DaysAgo: 4},
		{ItemID: "MBG-BHN-0001", Type: "OUT", Amount: 20, UnitPrice: 14500, Supplier: "-", DaysAgo: 2},
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
		}

		switch ss.Type {
		case "IN":
			newStock := util.Round4(item.Stock + ss.Amount)
			addedTotal := util.Round2(ss.Amount * ss.UnitPrice)
			st.NewStock = newStock
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

		if err := db.Create(&st).Error; err != nil {
			log.Warnf("stock track for %s failed: %v", ss.ItemID, err)
			continue
		}
		// Update the timestamp
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
	finances := []entity.Finance{
		{Type: "in", Category: "Penjualan", Description: "Penjualan kue tart pesanan Ibu Sari", Amount: 850000, ExtraNote: "Dibayar tunai", ProofImage: "/uploads/finances-proof/seed-proof-1.png", ModifiedBy: "Budi Almaliki"},
		{Type: "in", Category: "Penjualan", Description: "Penjualan roti tawar harian", Amount: 320000, ExtraNote: "Transfer BCA", ProofImage: "/uploads/finances-proof/seed-proof-2.png", ModifiedBy: "Andi Wijaya"},
		{Type: "out", Category: "Pembelian Bahan", Description: "Pembelian tepung terigu 50kg", Amount: 600000, ExtraNote: "Bayar ke PT Bogasari", ProofImage: "/uploads/finances-proof/seed-proof-3.png", ModifiedBy: "Budi Almaliki"},
		{Type: "out", Category: "Operasional", Description: "Bayar listrik bulan Maret", Amount: 450000, ExtraNote: "Token listrik", ProofImage: "/uploads/finances-proof/seed-proof-4.png", ModifiedBy: "Farah Nabila"},
		{Type: "in", Category: "Penjualan", Description: "Penjualan brownies box 20pcs", Amount: 1200000, ExtraNote: "Pesanan katering", ProofImage: "/uploads/finances-proof/seed-proof-5.png", ModifiedBy: "Budi Almaliki"},
		{Type: "out", Category: "Pembelian Bahan", Description: "Pembelian gula pasir 30kg", Amount: 450000, ExtraNote: "Bayar tunai ke CV Makmur", ProofImage: "/uploads/finances-proof/seed-proof-6.png", ModifiedBy: "Andi Wijaya"},
		{Type: "out", Category: "Gaji", Description: "Gaji karyawan bulan Maret", Amount: 3500000, ExtraNote: "Transfer payroll", ProofImage: "/uploads/finances-proof/seed-proof-7.png", ModifiedBy: "Budi Almaliki"},
		{Type: "in", Category: "Penjualan", Description: "Penjualan donat isi 50pcs", Amount: 500000, ExtraNote: "Cash on delivery", ProofImage: "/uploads/finances-proof/seed-proof-8.png", ModifiedBy: "Aulia Rahman"},
		{Type: "out", Category: "Operasional", Description: "Bayar sewa tempat bulan Maret", Amount: 2000000, ExtraNote: "Transfer ke pemilik ruko", ProofImage: "/uploads/finances-proof/seed-proof-9.png", ModifiedBy: "Budi Almaliki"},
		{Type: "in", Category: "Penjualan", Description: "Penjualan cake ulang tahun", Amount: 750000, ExtraNote: "DP 50%, sisanya COD", ProofImage: "/uploads/finances-proof/seed-proof-10.png", ModifiedBy: "Farah Nabila"},
		{Type: "out", Category: "Pembelian Bahan", Description: "Pembelian keju cheddar 10kg", Amount: 850000, ExtraNote: "PT Kraft, invoice NET30", ProofImage: "/uploads/finances-proof/seed-proof-11.png", ModifiedBy: "Andi Wijaya"},
		{Type: "in", Category: "Penjualan", Description: "Penjualan roti manis assorted", Amount: 680000, ExtraNote: "Dibayar via QRIS", ProofImage: "/uploads/finances-proof/seed-proof-12.png", ModifiedBy: "Budi Almaliki"},
	}

	for i, f := range finances {
		f.Model.CreatedAt = now.AddDate(0, 0, -(28-i*2))
		if err := db.Create(&f).Error; err != nil {
			log.Warnf("finance record %d failed: %v", i+1, err)
		}
	}
	log.Info("seeding finances done")

	fmt.Println("✅ All seed data inserted successfully!")
}
