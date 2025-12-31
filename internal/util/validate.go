package util

import (
	"slices"

	"github.com/go-playground/validator/v10"

	"github.com/ridwanmuh3/simba-server/internal/model"
)

func ValidateUserRole(fl validator.FieldLevel) bool {
	return slices.Contains(model.UserRoles, fl.Field().String())
}
