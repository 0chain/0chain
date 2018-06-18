package logging

import (
	"io"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	lumberjack "gopkg.in/natefinch/lumberjack.v2"
)

// var (
// 	Logger  *zap.Logger
// 	sLogger *zap.SugaredLogger
// )
//
// /*InitLogging - intialize logging system */
// func InitLogging(mode string) {
// 	Logger = zap.NewNop()
// 	LoggerInit(mode, "log/0chain.log")
// }
//
// // Logs return sugared zap logger
// func Logs(mode string) {
// 	sLogger = Logger.Sugar()
// 	sLoggerInit(mode, "log/n2n.log")
// 	// return sLogger
// }
//
// func sLoggerInit(logMode, logFile string) {
// 	var conf .Config
// 	if logMode == "development" {
// 		conf = zap.NewDevelopmentConfig()
// 	} else if logMode == "production" {
// 		conf = zap.NewProductionConfig()
// 	} else {
// 		conf = zap.NewDevelopmentConfig()
// 	}
//
// 	if logFile != "" {
// 		conf.OutputPaths = append(conf.OutputPaths, logFile)
// 	}
//
// 	sLogger, _ = conf.Build()
// }
//
// /*LoggerInit - initialize the logger */
// func LoggerInit(logMode, logFile string) {
// 	var conf zapcore.EncoderConfig
// 	if logMode == "development" {
// 		conf = zap.NewDevelopmentEncoderConfig()
// 	} else if logMode == "production" {
// 		conf = zap.NewProductionEncoderConfig()
// 	} else {
// 		conf = zap.NewDevelopmentEncoderConfig()
// 	}
// 	/*
// 		conf = zapcore.EncoderConfig{
// 			TimeKey:        "@timestamp",
// 			LevelKey:       "level",
// 			NameKey:        "logger",
// 			CallerKey:      "caller",
// 			MessageKey:     "msg",
// 			StacktraceKey:  "stacktrace",
// 			EncodeLevel:    zapcore.LowercaseLevelEncoder,
// 			EncodeTime:     zapcore.ISO8601TimeEncoder,
// 			EncodeDuration: zapcore.NanosDurationEncoder,
// 		} */
// 	writer := zapcore.AddSync(&lumberjack.Logger{
// 		Filename:   logFile,
// 		MaxSize:    10, // megabytes
// 		MaxBackups: 5,
// 		MaxAge:     28, // days
// 	})
// 	if logMode == "development" {
// 		core := zapcore.NewCore(
// 			zapcore.NewJSONEncoder(conf),
// 			writer,
// 			zap.DebugLevel,
// 		)
// 		Logger = zap.New(core, zap.AddCaller())
// 	} else {
// 		core := zapcore.NewCore(
// 			zapcore.NewJSONEncoder(conf),
// 			writer,
// 			zap.InfoLevel,
// 		)
// 		Logger = zap.New(core)
// 	}
// }

var (
	Logger *zap.Logger
	N2n    *zap.Logger
)

type WriteSyncer struct {
	io.Writer
}

func (ws WriteSyncer) Sync() error {
	return nil
}

func InitLogging(mode string) {
	var cfg zap.Config
	var logName = "log/0chain.log"
	var slogName = "log/n2n.log"

	if mode == "production" {
		cfg = zap.NewProductionConfig()
		cfg.DisableCaller = true
	} else {
		cfg = zap.NewDevelopmentConfig()
		cfg.EncoderConfig.LevelKey = "level"
		cfg.EncoderConfig.NameKey = "name"
		cfg.EncoderConfig.MessageKey = "msg"
		cfg.EncoderConfig.CallerKey = "caller"
		cfg.EncoderConfig.StacktraceKey = "stacktrace"
	}

	cfg.Encoding = "json"
	cfg.EncoderConfig.TimeKey = "timestamp"
	cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	cfg.OutputPaths = []string{logName}
	//cfg.ErrorOutputPaths = []string{"logs/error.log"}

	sw := getWriteSyncer(logName)
	swSugar := getWriteSyncer(slogName)

	l, err := cfg.Build(SetOutput(sw, cfg))
	if err != nil {
		panic(err)
	}
	defer l.Sync()

	ls, err := cfg.Build(SetOutput(swSugar, cfg))
	if err != nil {
		panic(err)
	}
	defer ls.Sync()

	Logger = l
	N2n = ls

}

// SetOutput replaces existing Core with new, that writes to passed WriteSyncer.
func SetOutput(ws zapcore.WriteSyncer, conf zap.Config) zap.Option {
	var enc zapcore.Encoder
	switch conf.Encoding {
	case "json":
		enc = zapcore.NewJSONEncoder(conf.EncoderConfig)
	case "console":
		enc = zapcore.NewConsoleEncoder(conf.EncoderConfig)
	default:
		panic("unknown encoding")
	}

	return zap.WrapCore(func(core zapcore.Core) zapcore.Core {
		return zapcore.NewCore(enc, ws, conf.Level)
	})
}

func getWriteSyncer(logName string) zapcore.WriteSyncer {
	var ioWriter = &lumberjack.Logger{
		Filename:   logName,
		MaxSize:    10, // MB
		MaxBackups: 3,  // number of backups
		MaxAge:     28, //days
		LocalTime:  true,
		Compress:   false, // disabled by default
	}
	var sw = WriteSyncer{
		ioWriter,
	}
	return sw
}
