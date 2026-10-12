package util

import (
	"slices"

	"github.com/go-playground/validator/v10"
)

func ValidateSliceOfStruct[T any](validate *validator.Validate, slice []T) error {
	var allErrors validator.ValidationErrors
	for _, entity := range slice {
		if err := validate.Struct(entity); err != nil {
			if validationErrors, ok := err.(validator.ValidationErrors); ok {
				allErrors = append(allErrors, validationErrors...)
			} else {
				return err
			}
		}
	}
	if len(allErrors) > 0 {
		return allErrors
	}
	return nil
}

func ValidateUserRole(fl validator.FieldLevel) bool {
	return slices.Contains([]string{"Admin", "Super Admin"}, fl.Field().String())
}

func ValidateStrongPassword(fl validator.FieldLevel) bool {
	return IsStrongPassword(fl.Field().String())
}

func IsStrongPassword(password string) bool {
	if len(password) < 8 || len(password) > 30 {
		return false
	}

	hasUpper := false
	hasNumber := false
	hasSymbol := false

	for i := 0; i < len(password); i++ {
		ch := password[i]
		if ch < 0x20 || ch > 0x7E {
			return false
		}

		switch {
		case ch >= 'A' && ch <= 'Z':
			hasUpper = true
		case ch >= '0' && ch <= '9':
			hasNumber = true
		case !(ch >= 'a' && ch <= 'z'):
			hasSymbol = true
		}
	}

	return hasUpper && hasNumber && hasSymbol
}
