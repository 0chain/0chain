package logging

import (
	"os"

	"github.com/spf13/viper"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	lumberjack "gopkg.in/natefinch/lumberjack.v2"
)

var (
	Logger *zap.Logger
	N2n    *zap.Logger
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
		logWriter = zapcore.NewMultiWriteSyncer(zapcore.AddSync(os.Stdout), logWriter)
		n2nLogWriter = zapcore.NewMultiWriteSyncer(zapcore.AddSync(os.Stdout), n2nLogWriter)
	}
	cfg.Level.UnmarshalText([]byte(viper.GetString("logging.level")))
	cfg.Encoding = "console"
	cfg.EncoderConfig.TimeKey = "timestamp"
	cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	l, err := cfg.Build(SetOutput(logWriter, cfg))
	if err != nil {
		panic(err)
	}

	ls, err := cfg.Build(SetOutput(n2nLogWriter, cfg))
	if err != nil {
		panic(err)
	}

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
	ioWriter.Rotate()
	return zapcore.AddSync(ioWriter)
}
