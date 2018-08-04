package logging

import (
	"os"

	"go.uber.org/zap/zapcore"

	"github.com/spf13/viper"
	"go.uber.org/zap"
	lumberjack "gopkg.in/natefinch/lumberjack.v2"
)

var (
	Logger *zap.Logger
	N2n    *zap.Logger

	MLogger    *MemLogger
	N2NMLogger *MemLogger
)

func InitLogging(mode string) {
	var logName = "log/0chain.log"
	var n2nLogName = "log/n2n.log"

	var logWriter = getWriteSyncer(logName)
	var n2nLogWriter = getWriteSyncer(n2nLogName)

	var cfg zap.Config
	if mode != "development" {
		cfg = zap.NewProductionConfig()
		cfg.DisableCaller = true
	} else {
		cfg = zap.NewDevelopmentConfig()
		cfg.EncoderConfig.LevelKey = "level"
		cfg.EncoderConfig.NameKey = "name"
		cfg.EncoderConfig.MessageKey = "msg"
		cfg.EncoderConfig.CallerKey = "caller"
		cfg.EncoderConfig.StacktraceKey = "stacktrace"
		if viper.GetBool("logging.console") {
			logWriter = zapcore.NewMultiWriteSyncer(zapcore.AddSync(os.Stdout), logWriter)
			n2nLogWriter = zapcore.NewMultiWriteSyncer(zapcore.AddSync(os.Stdout), n2nLogWriter)
		}
	}
	cfg.Level.UnmarshalText([]byte(viper.GetString("logging.level")))
	cfg.Encoding = "console"
	cfg.EncoderConfig.TimeKey = "timestamp"
	cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	level := zap.ErrorLevel

	MLogger = createMemLogger(level, cfg)
	option := createOptionFromCores(createZapCore(logWriter, cfg), MLogger.GetCore())
	l, err := cfg.Build(option)
	if err != nil {
		panic(err)
	}

	N2NMLogger = createMemLogger(level, cfg)
	option = createOptionFromCores(createZapCore(n2nLogWriter, cfg), N2NMLogger.GetCore())
	ls, err := cfg.Build(option)
	if err != nil {
		panic(err)
	}

	Logger = l
	N2n = ls
}

func createMemLogger(level zapcore.Level, conf zap.Config) *MemLogger {
	enc := getEncoder(conf)
	return NewMemLogger(enc, level)
}

func createZapCore(ws zapcore.WriteSyncer, conf zap.Config) zapcore.Core {
	enc := getEncoder(conf)
	return zapcore.NewCore(enc, ws, conf.Level)
}

func createOptionFromCores(cores ...zapcore.Core) zap.Option {
	return zap.WrapCore(func(core zapcore.Core) zapcore.Core {
		return zapcore.NewTee(cores...)
	})

}

func getEncoder(conf zap.Config) zapcore.Encoder {
	var enc zapcore.Encoder
	switch conf.Encoding {
	case "json":
		enc = zapcore.NewJSONEncoder(conf.EncoderConfig)
	case "console":
		enc = zapcore.NewConsoleEncoder(conf.EncoderConfig)
	default:
		panic("unknown encoding")
	}
	return enc
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
	ioWriter.Rotate()
	return zapcore.AddSync(ioWriter)
}
