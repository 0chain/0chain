package logging

import (
	"fmt"

	"go.uber.org/zap"
)

var (
	Logger *zap.Logger
)

/*InitLogging - intialize logging system */
func InitLogging(mode string) {
	Logger = zap.NewNop()
	LoggerInit(mode, "log/0chain.log")
}

/*LoggerInit - initialize the logger */
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
		if logMode == "production" {
			conf.OutputPaths = []string{logFile}
		} else {
			conf.OutputPaths = append(conf.OutputPaths, logFile)
		}
	}
	var err error
	Logger, err = conf.Build()
	if err != nil {
		//panic(fmt.Sprintf("error initializing the logging system: %v", err))
		fmt.Printf("error initializing logging: %v", err)
	}
}
