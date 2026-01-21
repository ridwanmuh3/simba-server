package config

import (
	"fmt"

	"go.uber.org/zap"
)

func NewLogger() *zap.SugaredLogger {
	// cwd, _ := os.Getwd()
	// logDir := filepath.Join(cwd, "logs")
	// if err := os.MkdirAll(logDir, 0o755); err != nil {
	// 	panic(fmt.Errorf("failed to create log directory: %v", err))
	// }

	// logFileName := filepath.Join(logDir, fmt.Sprintf("simba-server-%d.log", time.Now().Unix()))

	// file, err := os.OpenFile(logFileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	// if err != nil {
	// 	panic(fmt.Errorf("failed to create log file: %v", err))
	// }
	// defer file.Close()

	zapConfig := zap.NewDevelopmentConfig()
	zapConfig.OutputPaths = []string{
		// logFileName,
		"stderr",
	}

	log, err := zapConfig.Build()
	if err != nil {
		panic(fmt.Errorf("failed to instantiate logger: %v", err))
	}

	return log.Sugar()
}
