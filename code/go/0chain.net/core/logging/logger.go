package logging

import (
	"os"

	"go.uber.org/zap/zapcore"

	"github.com/spf13/viper"
	"go.uber.org/zap"
	lumberjack "gopkg.in/natefinch/lumberjack.v2"
)

var (
	Logger   *zap.Logger
	N2n      *zap.Logger
	MemUsage *zap.Logger

	mLogger    *MemLogger
	mHCLogger  *MemLogger
	mN2nLogger *MemLogger
	mMLogger   *MemLogger

	// Health-Check logger. Currently only used for sharder.
	HCLogger *zap.Logger
)

//InitLogging - initialize the logging submodule
func InitLogging(mode string) {
	var logName = "log/0chain.log"
	var n2nLogName = "log/n2n.log"
	var memLogName = "log/memUsage.log"
	var hcLogName = "log/hc.log"

	var logWriter = getWriteSyncer(logName)
	var n2nLogWriter = getWriteSyncer(n2nLogName)
	var memLogWriter = getWriteSyncer(memLogName)
	var hcWriter = getWriteSyncer(hcLogName)

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
			memLogWriter = zapcore.NewMultiWriteSyncer(zapcore.AddSync(os.Stdout), memLogWriter)
		}
	}
	cfg.Level.UnmarshalText([]byte(viper.GetString("logging.level")))
	cfg.Encoding = "console"
	cfg.EncoderConfig.TimeKey = "timestamp"
	cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	mlcfg := zap.NewProductionConfig()
	if mode != "development" {
		mlcfg.Level.SetLevel(zapcore.ErrorLevel)
	} else {
		mlcfg.Level.SetLevel(zapcore.DebugLevel)
	}
	mLogger = createMemLogger(mlcfg)
	option := createOptionFromCores(createZapCore(logWriter, cfg), mLogger.GetCore())
	l, err := cfg.Build(option)
	if err != nil {
		panic(err)
	}

	mn2ncfg := zap.NewProductionConfig()
	mn2ncfg.Level.SetLevel(zapcore.InfoLevel)
	mN2nLogger = createMemLogger(mn2ncfg)
	option = createOptionFromCores(createZapCore(n2nLogWriter, cfg), mN2nLogger.GetCore())
	ls, err := cfg.Build(option)
	if err != nil {
		panic(err)
	}

	mucfg := zap.NewProductionConfig()
	mucfg.Level.SetLevel(zapcore.InfoLevel)
	mMLogger = createMemLogger(mucfg)
	option = createOptionFromCores(createZapCore(memLogWriter, cfg), mMLogger.GetCore())
	lu, err := cfg.Build(option)
	if err != nil {
		panic(err)
	}

	// Create health-check writer.
	mhclcfg := zap.NewProductionConfig()
	mhclcfg.Level.SetLevel(zapcore.ErrorLevel)
	mHCLogger = createMemLogger(mhclcfg)
	option = createOptionFromCores(createZapCore(hcWriter, cfg), mHCLogger.GetCore())
	hcl, err := cfg.Build(option)
	if err != nil {
		panic(err)
	}

	Logger = l
	HCLogger = hcl
	N2n = ls
	MemUsage = lu
}

func createZapCore(ws zapcore.WriteSyncer, conf zap.Config) zapcore.Core {
	enc := getEncoder(conf)
	return zapcore.NewCore(enc, ws, conf.Level)
}

func createMemLogger(conf zap.Config) *MemLogger {
	enc := getEncoder(conf)
	return NewMemLogger(enc, conf.Level)
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
		MaxSize:    100, // MB
		MaxBackups: 5,   // number of backups
		MaxAge:     28,  //days
		LocalTime:  false,
		Compress:   false, // disabled by default
	}
	ioWriter.Rotate()
	return zapcore.AddSync(ioWriter)
}
