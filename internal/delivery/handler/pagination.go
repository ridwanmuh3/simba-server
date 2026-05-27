package handler

import (
	"math"

	"github.com/ridwanmuh3/simba-server/internal/model"
)

func newPageMetadata(page, size int, total int64) *model.PageMetadata {
	if page < 1 {
		page = 1
	}
	if size < 1 {
		size = 10
	}

	totalPage := int64(0)
	if total > 0 {
		totalPage = int64(math.Ceil(float64(total) / float64(size)))
	}

	return &model.PageMetadata{
		Page:      page,
		Size:      size,
		TotalItem: total,
		TotalPage: totalPage,
	}
}
