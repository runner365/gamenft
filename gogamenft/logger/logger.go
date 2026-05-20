package logger

import (
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	rl "github.com/lestrrat-go/file-rotatelogs"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

var (
	logger *zap.SugaredLogger
	once   sync.Once
)

// Config 日志配置
type Config struct {
	Level      string
	Filename   string
	MaxSize    int
	MaxBackups int
	MaxAge     int
	Compress   bool
	// DailyRotate when true will rotate logs by day producing files like
	// <Filename>.YYYY.MM.DD and keep a symlink at <Filename> pointing to
	// the current file.
	DailyRotate bool
}

// InitLogger 初始化日志
func InitLogger(cfg Config) {
	once.Do(func() {
		// 1. 级别设置
		level := zap.NewAtomicLevelAt(zapcore.InfoLevel)
		switch cfg.Level {
		case "debug":
			level = zap.NewAtomicLevelAt(zapcore.DebugLevel)
		case "info":
			level = zap.NewAtomicLevelAt(zapcore.InfoLevel)
		case "warn":
			level = zap.NewAtomicLevelAt(zapcore.WarnLevel)
		case "error":
			level = zap.NewAtomicLevelAt(zapcore.ErrorLevel)
		}

		// 2. 写入目标
		var writeSyncer zapcore.WriteSyncer
		if cfg.Filename == "" {
			writeSyncer = zapcore.AddSync(os.Stdout)
		} else if cfg.DailyRotate {
			// Use time-based rotation with pattern: <filename>.YYYY.MM.DD
			pattern := cfg.Filename + ".%Y.%m.%d"
			rotator, err := rl.New(
				pattern,
				rl.WithLinkName(cfg.Filename),
				rl.WithRotationTime(24*time.Hour),
				rl.WithMaxAge(time.Duration(cfg.MaxAge)*24*time.Hour),
			)
			if err != nil {
				// fallback to size-based lumberjack when rotator fails
				writeSyncer = zapcore.AddSync(&lumberjack.Logger{
					Filename:   cfg.Filename,
					MaxSize:    cfg.MaxSize,
					MaxBackups: cfg.MaxBackups,
					MaxAge:     cfg.MaxAge,
					Compress:   cfg.Compress,
				})
			} else {
				writeSyncer = zapcore.AddSync(rotator)
			}
		} else {
			writeSyncer = zapcore.AddSync(&lumberjack.Logger{
				Filename:   cfg.Filename,
				MaxSize:    cfg.MaxSize,
				MaxBackups: cfg.MaxBackups,
				MaxAge:     cfg.MaxAge,
				Compress:   cfg.Compress,
			})
		}

		// 3. 编码器
		encoderConfig := zap.NewProductionEncoderConfig()
		encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
		encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder

		core := zapcore.NewCore(
			zapcore.NewJSONEncoder(encoderConfig),
			writeSyncer,
			level,
		)

		// 4. 构建
		zapLogger := zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1))
		logger = zapLogger.Sugar()

		// 5. 【核心改动】注册自动关闭钩子
		// 监听 Ctrl+C (SIGINT) 和 Kill (SIGTERM) 信号
		go func() {
			c := make(chan os.Signal, 1)
			signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
			<-c
			// 收到信号后，先同步日志，再退出
			if logger != nil {
				logger.Sync()
			}
			// 如果需要优雅退出其他资源，可以在这里处理
			// 最后强制退出，防止程序挂起
			os.Exit(1)
		}()
	})
}

// Close 显式关闭日志（供 defer 使用）
func Close() {
	if logger != nil {
		logger.Sync()
	}
}

// --- 对外 API ---

func LogInfof(format string, args ...interface{}) {
	if logger == nil {
		InitLogger(Config{Level: "info"})
	}
	logger.Infof(format, args...)
}

func LogDebugf(format string, args ...interface{}) {
	if logger == nil {
		InitLogger(Config{Level: "debug"})
	}
	logger.Debugf(format, args...)
}

func LogWarnf(format string, args ...interface{}) {
	if logger == nil {
		InitLogger(Config{Level: "info"})
	}
	logger.Warnf(format, args...)
}

func LogErrorf(format string, args ...interface{}) {
	if logger == nil {
		InitLogger(Config{Level: "info"})
	}
	logger.Errorf(format, args...)
}
