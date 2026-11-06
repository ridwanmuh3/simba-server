package service

import (
	"gorm.io/gorm"

	"github.com/ridwanmuh3/simba-server/internal/entity"
	"github.com/ridwanmuh3/simba-server/internal/util"
)

// macRepository is the persistence port for MAC chain rebuilds. Load and
// save go through the repository; the chain math is the pure
// util.CalculateMAC, so domain logic no longer depends on *gorm.DB.
type macRepository interface {
	ListTracksChronological(db *gorm.DB, itemID string, dapurID uint) ([]entity.StockTracking, error)
	UpdateTracksMAC(db *gorm.DB, tracks []entity.StockTracking) error
}

// recalculateMAC loads an item's tracks via the repository, rebuilds the
// chain with the pure CalculateMAC, and persists the result — all inside
// the caller's transaction (the item row lock serializes writers).
func recalculateMAC(repo macRepository, tx *gorm.DB, item *entity.Item, dapurID uint) (tracks []entity.StockTracking, finalStock, finalTotalPrice float64, err error) {
	loaded, err := repo.ListTracksChronological(tx, item.ID, dapurID)
	if err != nil {
		return nil, 0, 0, err
	}
	updated, finalStock, finalTotalPrice, err := util.CalculateMAC(item.InitialStock, item.UnitPrice, loaded)
	if err != nil {
		return nil, 0, 0, err
	}
	if err := repo.UpdateTracksMAC(tx, updated); err != nil {
		return nil, 0, 0, err
	}
	return updated, finalStock, finalTotalPrice, nil
}
