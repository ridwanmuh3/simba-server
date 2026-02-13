package converter

import (
	"github.com/ridwanmuh3/simba-server/internal/entity"
	"github.com/ridwanmuh3/simba-server/internal/model"
)

func FinanceToResponse(finance *entity.Finance) *model.FinanceResponse {
	return &model.FinanceResponse{
		ID:          int(finance.ID),
		Type:        finance.Type,
		Category:    finance.Category,
		Description: finance.Description,
		Amount:      finance.Amount,
		ExtraNote:   finance.ExtraNote,
		ProofImage:  finance.ProofImage,
		CreatedAt:   finance.CreatedAt,
	}
}
