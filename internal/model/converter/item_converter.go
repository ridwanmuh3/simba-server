package converter

import (
	"github.com/ridwanmuh3/simba-server/internal/entity"
	"github.com/ridwanmuh3/simba-server/internal/model"
)

func ItemToResponse(item *entity.Item) *model.ItemResponse {
	return &model.ItemResponse{
		ID:          item.ID,
		Name:        item.Name,
		Category:    item.Category,
		Quantity:    item.Quantity,
		MeasureUnit: item.MeasureUnit,
		UnitPrice:   item.UnitPrice,
		TotalPrice:  item.TotalPrice,
	}
}
