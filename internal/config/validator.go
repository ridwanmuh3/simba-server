package config

import (
	"github.com/go-playground/validator/v10"
	"github.com/spf13/viper"

	"github.com/ridwanmuh3/simba-server/internal/util"
)

func NewValidator(viper *viper.Viper) *validator.Validate {
	validate := validator.New(validator.WithRequiredStructEnabled())
	validate.RegisterValidation("userrole", util.ValidateUserRole)
	return validate
}
