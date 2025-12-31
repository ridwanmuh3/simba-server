package config

import (
	"fmt"
	"os"

	"github.com/spf13/viper"
)

func NewViper() *viper.Viper {

	_, err := os.Stat(".development.env")
	if err != nil {
		panic(fmt.Errorf("failed to locate env file: %v", err))
	}

	config := viper.New()

	config.SetConfigName(".development")
	config.SetConfigType("env")
	config.AddConfigPath("./")
	config.AddConfigPath("./../")

	err = config.ReadInConfig()
	if err != nil {
		panic(fmt.Errorf("failed to read config on env file: %v", err))
	}

	return config
}
