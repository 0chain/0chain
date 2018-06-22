package logging

import (
	"io"
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
	cfg.Level.UnmarshalText([]byte(viper.GetString("logging.level")))

	cfg.Encoding = "console"
	cfg.EncoderConfig.TimeKey = "timestamp"
	cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	sw := zapcore.NewMultiWriteSyncer(zapcore.AddSync(os.Stdout), getWriteSyncer(logName))
	swSugar := zapcore.NewMultiWriteSyncer(zapcore.AddSync(os.Stdout), getWriteSyncer(slogName))

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
	ioWriter.Rotate()
	var sw = WriteSyncer{
		ioWriter,
	}
	return sw
}
