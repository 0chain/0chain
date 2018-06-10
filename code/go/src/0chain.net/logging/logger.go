package logging

import (
	"go.uber.org/zap"
)

var (
	Logger *zap.Logger
)

func init() {
	Logger = zap.NewNop()
}

func LoggerInit(logMode, logFile string) {
	var conf zap.Config
	if logMode == "development" {
		conf = zap.NewDevelopmentConfig()
	} else if logMode == "production" {
		conf = zap.NewProductionConfig()
	} else {
		conf = zap.NewDevelopmentConfig()
	}

	if logFile != "" {
		conf.OutputPaths = append(conf.OutputPaths, logFile)
	}

	Logger, _ = conf.Build()
}
