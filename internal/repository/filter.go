package repository

import (
	"time"

	"gorm.io/gorm"
)

// DateRangeScope returns a GORM scope that filters a column by an optional
// RFC3339Nano date range. Both bounds are inclusive. Empty strings are ignored.
func DateRangeScope(column, from, to string) func(*gorm.DB) *gorm.DB {
	return func(tx *gorm.DB) *gorm.DB {
		loc, _ := time.LoadLocation("Asia/Jakarta")
		if from != "" {
			parsed, err := time.Parse(time.RFC3339Nano, from)
			if err == nil {
				parsed = parsed.In(loc)
				startOfDay := time.Date(parsed.Year(), parsed.Month(), parsed.Day(), 0, 0, 0, 0, loc)
				tx = tx.Where(column+" >= ?", startOfDay.UTC())
			}
		}
		if to != "" {
			parsed, err := time.Parse(time.RFC3339Nano, to)
			if err == nil {
				parsed = parsed.In(loc)
				endOfDay := time.Date(parsed.Year(), parsed.Month(), parsed.Day(), 23, 59, 59, 999999999, loc)
				tx = tx.Where(column+" <= ?", endOfDay.UTC())
			}
		}
		return tx
	}
}

// SearchScope returns a GORM scope that applies an ILIKE search across
// multiple columns with OR logic.
func SearchScope(columns []string, query string) func(*gorm.DB) *gorm.DB {
	return func(tx *gorm.DB) *gorm.DB {
		if query == "" {
			return tx
		}
		pattern := "%" + query + "%"
		sub := tx.Session(&gorm.Session{NewDB: true})
		for i, col := range columns {
			if i == 0 {
				sub = sub.Where(col+" ILIKE ?", pattern)
			} else {
				sub = sub.Or(col+" ILIKE ?", pattern)
			}
		}
		return tx.Where(sub)
	}
}
