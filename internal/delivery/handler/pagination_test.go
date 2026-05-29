package handler

import "testing"

func TestNewPageMetadata(t *testing.T) {
	tests := []struct {
		name          string
		page          int
		size          int
		total         int64
		wantPage      int
		wantSize      int
		wantTotalPage int64
	}{
		{
			name:          "empty total keeps zero total page",
			page:          1,
			size:          10,
			total:         0,
			wantPage:      1,
			wantSize:      10,
			wantTotalPage: 0,
		},
		{
			name:          "rounds up partial page",
			page:          2,
			size:          10,
			total:         21,
			wantPage:      2,
			wantSize:      10,
			wantTotalPage: 3,
		},
		{
			name:          "normalizes invalid page and size",
			page:          0,
			size:          0,
			total:         1,
			wantPage:      1,
			wantSize:      10,
			wantTotalPage: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := newPageMetadata(tt.page, tt.size, tt.total)
			if got.Page != tt.wantPage {
				t.Fatalf("Page = %d, want %d", got.Page, tt.wantPage)
			}
			if got.Size != tt.wantSize {
				t.Fatalf("Size = %d, want %d", got.Size, tt.wantSize)
			}
			if got.TotalItem != tt.total {
				t.Fatalf("TotalItem = %d, want %d", got.TotalItem, tt.total)
			}
			if got.TotalPage != tt.wantTotalPage {
				t.Fatalf("TotalPage = %d, want %d", got.TotalPage, tt.wantTotalPage)
			}
		})
	}
}
