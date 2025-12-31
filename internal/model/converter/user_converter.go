package converter

import (
	"github.com/ridwanmuh3/simba-server/internal/entity"
	"github.com/ridwanmuh3/simba-server/internal/model"
)

func UserToResponse(user *entity.User) *model.UserResponse {
	return &model.UserResponse{
		ID:         int(user.ID),
		Username:   user.Username,
		Fullname:   user.Fullname,
		Role:       user.Role,
		Token:      user.Token,
		IsActive:   user.IsActive,
		LastActive: user.LastActive,
		CreatedAt:  user.CreatedAt,
		UpdatedAt:  user.UpdatedAt,
	}
}
