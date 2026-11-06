package repository

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/ridwanmuh3/simba-server/internal/entity"
	"github.com/ridwanmuh3/simba-server/internal/model"
)

// itemIDLockKey serializes item-ID generation across instances.
const itemIDLockKey = 7283910

// NextItemNumber acquires the item-ID advisory lock, then returns the
// next numeric suffix. Caller must be inside a transaction.
func (r *ItemRepository) NextItemNumber(db *gorm.DB) (int, error) {
	if err := db.Exec("SELECT pg_advisory_xact_lock(?)", itemIDLockKey).Error; err != nil {
		return 0, err
	}
	var lastItem entity.Item
	err := db.Unscoped().
		Where("id ~ ?", `^MBG-BHN-[0-9]+$`).
		Order("CAST(split_part(id, '-', 3) AS INTEGER) DESC").
		First(&lastItem).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 1, nil
		}
		return 0, err
	}
	parts := strings.Split(lastItem.ID, "-")
	if len(parts) == 3 {
		if n, parseErr := strconv.Atoi(parts[2]); parseErr == nil {
			return n + 1, nil
		}
	}
	return 0, fmt.Errorf("malformed last item ID: %s", lastItem.ID)
}

// FindItemForUpdate locks the item row (SELECT ... FOR UPDATE), scoped to dapur.
func (r *ItemRepository) FindItemForUpdate(db *gorm.DB, itemID string, dapurID uint) (*entity.Item, error) {
	item := new(entity.Item)
	err := db.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id = ? AND dapur_id = ?", itemID, dapurID).
		First(item).Error
	return item, err
}

// FindItemByIDTenant finds one item scoped to dapur.
func (r *ItemRepository) FindItemByIDTenant(db *gorm.DB, itemID string, dapurID uint) (*entity.Item, error) {
	item := new(entity.Item)
	err := db.Where("id = ? AND dapur_id = ?", itemID, dapurID).Take(item).Error
	return item, err
}

// UpdateItemFields updates the editable item columns (ItemService.Update).
func (r *ItemRepository) UpdateItemFields(db *gorm.DB, item *entity.Item) error {
	return db.Model(item).
		Select("Name", "Category", "MeasureUnit", "Stock", "InitialStock", "UnitPrice", "TotalPrice", "ModifiedBy", "CreatedAt").
		Updates(item).Error
}

// FindCategories returns distinct categories for a dapur.
func (r *ItemRepository) FindCategories(db *gorm.DB, dapurID uint) ([]string, error) {
	var categories []string
	err := db.Model(new(entity.Item)).
		Where("dapur_id = ? AND category != ''", dapurID).
		Distinct("category").
		Order("category ASC").
		Pluck("category", &categories).Error
	return categories, err
}

// FindAllForExport returns all items of a dapur, newest first.
func (r *ItemRepository) FindAllForExport(db *gorm.DB, dapurID uint) ([]entity.Item, error) {
	var items []entity.Item
	err := db.Model(new(entity.Item)).
		Scopes(Tenant(dapurID)).
		Order("created_at DESC, id DESC").Find(&items).Error
	return items, err
}

// --- stock tracks ---

// FindTrackForUpdate locks one stock track, scoped to item + dapur.
func (r *ItemRepository) FindTrackForUpdate(db *gorm.DB, trackID uint, itemID string, dapurID uint) (*entity.StockTracking, error) {
	stock := new(entity.StockTracking)
	err := db.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id = ? AND item_id = ? AND dapur_id = ?", trackID, itemID, dapurID).
		First(stock).Error
	return stock, err
}

func (r *ItemRepository) CreateTrack(db *gorm.DB, track *entity.StockTracking) error {
	return db.Create(track).Error
}

// UpdateTrackFields updates the editable track columns (StockService.EditStock).
func (r *ItemRepository) UpdateTrackFields(db *gorm.DB, track *entity.StockTracking) error {
	return db.Model(track).Updates(map[string]any{
		"amount":      track.Amount,
		"unit_price":  track.UnitPrice,
		"supplier":    track.Supplier,
		"modified_by": track.ModifiedBy,
	}).Error
}

// DeleteTrackHard permanently deletes one track (stock chain is rebuilt after).
func (r *ItemRepository) DeleteTrackHard(db *gorm.DB, trackID uint) error {
	return db.Unscoped().Delete(&entity.StockTracking{}, "id = ?", trackID).Error
}

// DeleteTracksByItem permanently deletes all tracks of an item in a dapur.
func (r *ItemRepository) DeleteTracksByItem(db *gorm.DB, itemID string, dapurID uint) error {
	return db.Unscoped().
		Where("item_id = ? AND dapur_id = ?", itemID, dapurID).
		Delete(&entity.StockTracking{}).Error
}

// ListTracksChronological loads an item's tracks oldest-first for MAC rebuild.
func (r *ItemRepository) ListTracksChronological(db *gorm.DB, itemID string, dapurID uint) ([]entity.StockTracking, error) {
	var tracks []entity.StockTracking
	err := db.Where("item_id = ? AND dapur_id = ? AND deleted_at IS NULL", itemID, dapurID).
		Order("created_at ASC, id ASC").
		Find(&tracks).Error
	return tracks, err
}

// UpdateTracksMAC persists the rewritten MAC chain (previous/new/total per row).
func (r *ItemRepository) UpdateTracksMAC(db *gorm.DB, tracks []entity.StockTracking) error {
	for i := range tracks {
		if err := db.Model(&entity.StockTracking{}).
			Where("id = ?", tracks[i].ID).
			UpdateColumns(map[string]any{
				"previous_stock": tracks[i].PreviousStock,
				"new_stock":      tracks[i].NewStock,
				"total_price":    tracks[i].TotalPrice,
			}).Error; err != nil {
			return fmt.Errorf("failed to update stock tracking id=%d: %w", tracks[i].ID, err)
		}
	}
	return nil
}

// SaveActivityLog persists an activity log inside the caller's transaction.
func (r *ItemRepository) SaveActivityLog(db *gorm.DB, log *entity.ActivityLog) error {
	return db.Create(log).Error
}

// --- aggregates previously computed in StockService ---

// StockFinanceSummary returns (master budget, cost IN, revenue OUT) for a dapur.
func (r *ItemRepository) StockFinanceSummary(db *gorm.DB, dapurID uint) (master, cost, revenue float64, err error) {
	if err = db.Model(new(entity.Item)).
		Select("COALESCE(SUM(total_price),0)").
		Where("dapur_id = ? AND deleted_at IS NULL", dapurID).
		Scan(&master).Error; err != nil {
		return 0, 0, 0, err
	}
	if err = db.Model(new(entity.StockTracking)).
		Select("COALESCE(SUM(amount * unit_price),0)").
		Where("type = ? AND dapur_id = ? AND deleted_at IS NULL", "IN", dapurID).
		Scan(&cost).Error; err != nil {
		return 0, 0, 0, err
	}
	if err = db.Model(new(entity.StockTracking)).
		Select("COALESCE(SUM(amount * unit_price),0)").
		Where("type = ? AND dapur_id = ? AND deleted_at IS NULL", "OUT", dapurID).
		Scan(&revenue).Error; err != nil {
		return 0, 0, 0, err
	}
	return master, cost, revenue, nil
}

// ItemStocksSummary paginates per-item stock summaries for a dapur.
func (r *ItemRepository) ItemStocksSummary(db *gorm.DB, request *model.GetItemStockSummaryRequest) ([]model.ItemStocksSummaryResponse, int64, error) {
	offset := (request.Page - 1) * request.Size

	baseQuery := db.Table("items").Where("items.deleted_at IS NULL AND items.dapur_id = ?", request.DapurID)
	if request.SearchQuery != "" {
		baseQuery = baseQuery.Where("items.name LIKE ?", "%"+request.SearchQuery+"%")
	}

	var total int64
	if err := baseQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if total == 0 {
		return []model.ItemStocksSummaryResponse{}, 0, nil
	}

	loc, _ := time.LoadLocation("Asia/Jakarta")
	joinClause := "LEFT JOIN stock_tracks st ON st.item_id = items.id AND st.deleted_at IS NULL AND st.dapur_id = ?"
	joinArgs := []interface{}{request.DapurID}

	if request.StartDate != "" {
		if parsed, err := time.Parse(time.RFC3339Nano, request.StartDate); err == nil {
			p := parsed.In(loc)
			start := time.Date(p.Year(), p.Month(), p.Day(), 0, 0, 0, 0, loc)
			joinClause += " AND st.created_at >= ?"
			joinArgs = append(joinArgs, start.UTC())
		}
	}
	if request.EndDate != "" {
		if parsed, err := time.Parse(time.RFC3339Nano, request.EndDate); err == nil {
			p := parsed.In(loc)
			end := time.Date(p.Year(), p.Month(), p.Day(), 23, 59, 59, 999999999, loc)
			joinClause += " AND st.created_at <= ?"
			joinArgs = append(joinArgs, end.UTC())
		}
	}

	var responses []model.ItemStocksSummaryResponse
	err := baseQuery.
		Select(`
            items.id AS item_id,
            items.name AS name,
            items.category AS category,
            items.measure_unit AS measure_unit,
            items.stock AS current_stock,
            items.total_price AS stock_value,
            items.initial_stock AS initial_stock,
            COALESCE(
                SUM(CASE WHEN st.type = 'IN' THEN st.total_price ELSE 0 END) /
                NULLIF(SUM(CASE WHEN st.type = 'IN' THEN st.amount ELSE 0 END), 0),
            items.unit_price) AS buy_price,
            COALESCE(
                SUM(CASE WHEN st.type = 'OUT' THEN st.total_price ELSE 0 END) /
                NULLIF(SUM(CASE WHEN st.type = 'OUT' THEN st.amount ELSE 0 END), 0),
            0) AS sell_price,
            COALESCE(SUM(CASE WHEN st.type = 'IN' THEN st.amount ELSE 0 END), 0) AS total_in,
            COALESCE(SUM(CASE WHEN st.type = 'OUT' THEN st.amount ELSE 0 END), 0) AS total_out
        `).
		Joins(joinClause, joinArgs...).
		Group(`
            items.id,
            items.name,
            items.category,
            items.measure_unit,
            items.stock,
            items.total_price,
            items.initial_stock,
            items.unit_price
        `).
		Order("items.created_at DESC, items.id DESC").
		Limit(request.Size).
		Offset(offset).
		Find(&responses).Error
	if err != nil {
		return nil, 0, err
	}
	return responses, total, nil
}

// DeleteItemHard permanently deletes an item row.
func (r *ItemRepository) DeleteItemHard(db *gorm.DB, item *entity.Item) error {
	return db.Unscoped().Delete(item).Error
}
