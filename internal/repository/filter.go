package repository

import (
	"time"

	"gorm.io/gorm"
)

// parseDateParam tries RFC3339 then RFC3339Nano so both frontend ISO strings
// (with or without fractional seconds) are accepted consistently.
func parseDateParam(s string) (time.Time, bool) {
	for _, layout := range []string{time.RFC3339, time.RFC3339Nano} {
		if t, err := time.Parse(layout, s); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

// DateRangeScope returns a GORM scope that filters a column by an optional
// date range (RFC3339 / RFC3339Nano). Both bounds are inclusive. Empty strings
// are ignored.
func DateRangeScope(column, from, to string) func(*gorm.DB) *gorm.DB {
	return func(tx *gorm.DB) *gorm.DB {
		loc, _ := time.LoadLocation("Asia/Jakarta")
		if from != "" {
			if parsed, ok := parseDateParam(from); ok {
				parsed = parsed.In(loc)
				startOfDay := time.Date(parsed.Year(), parsed.Month(), parsed.Day(), 0, 0, 0, 0, loc)
				tx = tx.Where(column+" >= ?", startOfDay.UTC())
			}
		}
		if to != "" {
			if parsed, ok := parseDateParam(to); ok {
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
