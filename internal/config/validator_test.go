package config

import (
	"testing"

	"github.com/spf13/viper"
)

func TestNewValidatorRegistersUserRole(t *testing.T) {
	validate := NewValidator(viper.New())

	var good struct {
		Role string `validate:"userrole"`
	}
	good.Role = "Admin"
	if err := validate.Struct(good); err != nil {
		t.Fatalf("Admin role validation error: %v", err)
	}

	var bad struct {
		Role string `validate:"userrole"`
	}
	bad.Role = "Viewer"
	if err := validate.Struct(bad); err == nil {
		t.Fatal("expected invalid user role error")
	}
}
