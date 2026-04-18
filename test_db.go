package main

import (
	"fmt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	dsn := "host=localhost user=postgres password=postgres dbname=simba port=5432 sslmode=disable"
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		fmt.Println("Error connecting:", err)
		return
	}

	type MonthlyBudgetStat struct {
		Month string
		In    float64
		Out   float64
	}
	var monthly []MonthlyBudgetStat

	err = db.Table("finances").
		Select(`
			TO_CHAR(created_at, 'Mon') AS month,
			SUM(CASE WHEN type='PEMASUKAN' THEN amount ELSE 0 END) AS "in",
			SUM(CASE WHEN type='PENGELUARAN' THEN amount ELSE 0 END) AS "out"
		`).
		Group("month").
		Order("MIN(created_at)").
		Scan(&monthly).Error
	fmt.Println("Error:", err)
	fmt.Println("Result:", monthly)
}
