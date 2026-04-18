package config

import (
	"fmt"

	"github.com/spf13/viper"
	"go.uber.org/zap"
)

func NewLogger(config *viper.Viper) *zap.SugaredLogger {
	var zapConfig zap.Config
	var log *zap.Logger
	var err error

	if config.GetString("APP_MODE") == "prod" {
		zapConfig = zap.NewProductionConfig()
		
		zapConfig.OutputPaths = []string{
			"stdout",
		}

		log, err = zapConfig.Build(zap.AddStacktrace(zap.WarnLevel))
		if err != nil {
			panic(fmt.Errorf("failed to instantiate logger: %v", err))
		}
	} else {
		zapConfig = zap.NewDevelopmentConfig()
		zapConfig.OutputPaths = []string{
			"stdout",
		}

		log, err = zapConfig.Build(zap.AddStacktrace(zap.ErrorLevel))
		if err != nil {
			panic(fmt.Errorf("failed to instantiate logger: %v", err))
		}
	}

	return log.Sugar()
}
