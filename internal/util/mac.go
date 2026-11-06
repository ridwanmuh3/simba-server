package util

import (
	"math"

	"github.com/ridwanmuh3/simba-server/internal/entity"
	"github.com/ridwanmuh3/simba-server/internal/exception"
)

// CalculateMAC is the pure domain core of the Merger-Average-Cost (MAC)
// recalculation: it rebuilds the running previous_stock / new_stock /
// total_price chain for tracks (already in chronological order) starting
// from the item's initial stock and unit price.
//
// It takes no *gorm.DB and performs no I/O, so it is unit-testable without
// a database. Persistence (loading tracks, saving the rewritten chain)
// belongs to the repository layer; see service.recalculateMAC.
//
// Arithmetic is exact integer arithmetic in scaled units — quantities in
// 1e-4 units and money in cents — instead of accumulated float64 math,
// so chains like 0.1 + 0.2 never drift. Float inputs are converted once
// via Round4/Round2-compatible rounding; outputs are converted back.
//
// If an OUT movement would drive the running stock negative it returns
// exception.InsufficientStockError and no partial result.
func CalculateMAC(initialStock, initialUnitPrice float64, tracks []entity.StockTracking) (updated []entity.StockTracking, finalStock, finalTotalPrice float64, err error) {
	updated = make([]entity.StockTracking, len(tracks))
	copy(updated, tracks)

	running := stockScaled(initialStock)
	total := moneyCents(initialStock * initialUnitPrice)

	for i := range updated {
		updated[i].PreviousStock = float64(running) / stockScale

		switch updated[i].Type {
		case "IN":
			amount := stockScaled(updated[i].Amount)
			running += amount
			added := lineTotalCents(updated[i].Amount, updated[i].UnitPrice)
			total += added
			updated[i].TotalPrice = float64(added) / moneyScale
			updated[i].NewStock = float64(running) / stockScale

		case "OUT":
			amount := stockScaled(updated[i].Amount)
			if amount > running {
				return nil, 0, 0, exception.InsufficientStockError
			}

			var deduction int64
			if running > 0 {
				// avg = Round4(totalPrice / runningStock), deduction =
				// Round2(avg * amount), computed in scaled integers:
				//   avgScaled4 = round(totalCents * 1e6 / running)
				//   deductionC = round(avgScaled4 * amount / 1e6)
				avgScaled4 := divRound(total*1_000_000, running)
				deduction = divRound(avgScaled4*amount, 1_000_000)
			}

			running -= amount
			total -= deduction
			if total < 0 {
				total = 0
			}

			updated[i].TotalPrice = float64(lineTotalCents(updated[i].Amount, updated[i].UnitPrice)) / moneyScale
			updated[i].NewStock = float64(running) / stockScale
		}
	}

	return updated, float64(running) / stockScale, float64(total) / moneyScale, nil
}

const (
	stockScale = 10000.0 // quantities: 4 decimal places (NUMERIC(18,4))
	moneyScale = 100.0   // money: cents (NUMERIC(18,2))
)

// stockScaled converts a quantity to exact 1e-4 units.
func stockScaled(v float64) int64 {
	return int64(math.Round(v * stockScale))
}

// moneyCents converts a money amount to exact cents.
func moneyCents(v float64) int64 {
	return int64(math.Round(v * moneyScale))
}

// lineTotalCents is Round2(amount * unitPrice) in exact integer math:
// amount (1e-4 units) * unitPrice (cents) / 1e4 = cents.
func lineTotalCents(amount, unitPrice float64) int64 {
	return divRound(stockScaled(amount)*moneyCents(unitPrice), 10000)
}

// divRound is integer division rounded half away from zero (all callers
// pass non-negative values).
func divRound(n, d int64) int64 {
	if n >= 0 {
		return (n + d/2) / d
	}
	return -((-n + d/2) / d)
}
