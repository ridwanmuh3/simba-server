package util

import (
	"slices"

	"github.com/go-playground/validator/v10"
)

func ValidateSliceOfStruct[T any](validate *validator.Validate, slice []T) error {
	var err error
	for _, entity := range slice {
		err = validate.Struct(entity)
	}
	return err
}

func ValidateUserRole(fl validator.FieldLevel) bool {
	return slices.Contains([]string{"Admin", "Super Admin"}, fl.Field().String())
}
