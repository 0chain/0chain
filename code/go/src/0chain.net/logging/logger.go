package logging

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	lumberjack "gopkg.in/natefinch/lumberjack.v2"
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
	var conf zapcore.EncoderConfig
	if logMode == "development" {
		conf = zap.NewDevelopmentEncoderConfig()
	} else if logMode == "production" {
		conf = zap.NewProductionEncoderConfig()
	} else {
		conf = zap.NewDevelopmentEncoderConfig()
	}
	/*
		conf = zapcore.EncoderConfig{
			TimeKey:        "@timestamp",
			LevelKey:       "level",
			NameKey:        "logger",
			CallerKey:      "caller",
			MessageKey:     "msg",
			StacktraceKey:  "stacktrace",
			EncodeLevel:    zapcore.LowercaseLevelEncoder,
			EncodeTime:     zapcore.ISO8601TimeEncoder,
			EncodeDuration: zapcore.NanosDurationEncoder,
		} */
	writer := zapcore.AddSync(&lumberjack.Logger{
		Filename:   logFile,
		MaxSize:    10, // megabytes
		MaxBackups: 5,
		MaxAge:     28, // days
	})
	if logMode == "development" {
		core := zapcore.NewCore(
			zapcore.NewJSONEncoder(conf),
			writer,
			zap.DebugLevel,
		)
		Logger = zap.New(core, zap.AddCaller())
	} else {
		core := zapcore.NewCore(
			zapcore.NewJSONEncoder(conf),
			writer,
			zap.InfoLevel,
		)
		Logger = zap.New(core)
	}
}
