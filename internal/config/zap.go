package config

import (
	"fmt"

	"go.uber.org/zap"
)

func NewLogger() *zap.SugaredLogger {
	log, err := zap.NewDevelopment()
	if err != nil {
		panic(fmt.Errorf("failed to instantiate logger: %v", err))
	}

	return log.Sugar()
}
