package util

import (
	"fmt"

	"gorm.io/gorm"

	"github.com/ridwanmuh3/simba-server/internal/entity"
)

// RecalculateMAC rebuilds the running MAC (Merger-Average-Cost) chain for an item
// after a stock tracking mutation (create/edit/delete). It iterates all stock
// tracks in chronological order and recomputes previous_stock, new_stock, and
// total_price for every row, persisting the results within tx.
//
// Returns the final stock and total_price values for the item, plus the
// updated tracks slice so callers can inspect individual rows.
func RecalculateMAC(tx *gorm.DB, item *entity.Item, dapurID uint) (tracks []entity.StockTracking, finalStock, finalTotalPrice float64, err error) {
	if err = tx.
		Where("item_id = ? AND dapur_id = ? AND deleted_at IS NULL", item.ID, dapurID).
		Order("created_at ASC, id ASC").
		Find(&tracks).Error; err != nil {
		return nil, 0, 0, err
	}

	runningStock := item.InitialStock
	totalPrice := Round2(item.InitialStock * item.UnitPrice)

	for i := range tracks {
		tracks[i].PreviousStock = Round4(runningStock)

		switch tracks[i].Type {
		case "IN":
			runningStock = Round4(runningStock + tracks[i].Amount)
			totalPrice = Round2(totalPrice + Round2(tracks[i].Amount*tracks[i].UnitPrice))
			tracks[i].TotalPrice = Round2(tracks[i].Amount * tracks[i].UnitPrice)
			tracks[i].NewStock = runningStock

		case "OUT":
			amount := tracks[i].Amount
			if amount > runningStock {
				amount = runningStock
			}

			var avg float64
			if runningStock > 0 {
				avg = Round4(totalPrice / runningStock)
			}

			deduction := Round2(avg * amount)
			runningStock = Round4(runningStock - amount)
			if runningStock < 0 {
				runningStock = 0
			}

			totalPrice = Round2(totalPrice - deduction)
			if totalPrice < 0 {
				totalPrice = 0
			}

			tracks[i].TotalPrice = Round2(amount * tracks[i].UnitPrice)
			tracks[i].NewStock = runningStock
		}
	}

	finalStock = Round4(runningStock)
	finalTotalPrice = Round2(totalPrice)

	for i := range tracks {
		if err = tx.Model(&entity.StockTracking{}).
			Where("id = ?", tracks[i].ID).
			UpdateColumns(map[string]any{
				"previous_stock": tracks[i].PreviousStock,
				"new_stock":      tracks[i].NewStock,
				"total_price":    tracks[i].TotalPrice,
			}).Error; err != nil {
			return nil, 0, 0, fmt.Errorf("failed to update stock tracking id=%d: %w", tracks[i].ID, err)
		}
	}

	return tracks, finalStock, finalTotalPrice, nil
}
