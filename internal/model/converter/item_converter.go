package converter

import (
	"github.com/ridwanmuh3/simba-server/internal/entity"
	"github.com/ridwanmuh3/simba-server/internal/model"
)

func ItemToResponse(item *entity.Item) *model.ItemResponse {
	return &model.ItemResponse{
		ID:           item.ID,
		Name:         item.Name,
		Category:     item.Category,
		Stock:        item.Stock,
		InitialStock: item.InitialStock,
		MeasureUnit:  item.MeasureUnit,
		UnitPrice:    item.UnitPrice,
		TotalPrice:   item.TotalPrice,
	}
}

func StockToResponse(stockTracking *entity.StockTracking) *model.StockResponse {
	return &model.StockResponse{
		ID:            int(stockTracking.ID),
		Type:          stockTracking.Type,
		Amount:        stockTracking.Amount,
		PreviousStock: stockTracking.PreviousStock,
		NewStock:      stockTracking.NewStock,
		Supplier:      stockTracking.Supplier,
		UnitPrice:     stockTracking.UnitPrice,
		TotalPrice:    stockTracking.TotalPrice,
		ModifiedBy:    stockTracking.ModifiedBy,
		CreatedAt:     stockTracking.CreatedAt,
		Item:          *ItemToResponse(&stockTracking.Item),
	}
}

func StockToItemStockSummaryResponse(stockTracking *entity.StockTracking) *model.ItemStocksSummaryResponse {
	return &model.ItemStocksSummaryResponse{
		ItemID:       stockTracking.Item.ID,
		Name:         stockTracking.Item.Name,
		Category:     stockTracking.Item.Category,
		InitialStock: stockTracking.Item.InitialStock,
		CurrentStock: stockTracking.Item.Stock,
		StockValue:   float64(stockTracking.Item.Stock) * stockTracking.Item.UnitPrice,
	}
}
