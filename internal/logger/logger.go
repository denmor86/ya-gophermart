package logger

import (
	"sync"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	once     sync.Once
	instance *zap.SugaredLogger = nil
)

// Initialize - инициализирует синглтон логера с необходимым уровнем логирования.
func Initialize(level string) error {
	var initErr error
	once.Do(func() {
		// преобразуем текстовый уровень логирования в zap.AtomicLevel
		lvl, err := zap.ParseAtomicLevel(level)
		if err != nil {
			initErr = err
			return
		}
		// создаём новую конфигурацию логера
		cfg := zap.NewProductionConfig()
		// устанавливаем уровень
		cfg.Level = lvl
		cfg.EncoderConfig.TimeKey = "time"
		cfg.EncoderConfig.EncodeTime = zapcore.TimeEncoderOfLayout("2006-01-02 15:04:05")
		// создаём логер на основе конфигурации
		logger, err := cfg.Build()
		if err != nil {
			initErr = err
			return
		}
		// устанавливаем синглтон
		instance = logger.Sugar()
	})
	return initErr
}

// Get - метод получения объекта логгера из синглтона
func Get() *zap.SugaredLogger {
	if instance == nil {
		// Возвращаем no-op логгер вместо паники
		return zap.NewNop().Sugar()
	}
	return instance
}

// Sync - метод синхронизации буфферов
func Sync() error {
	if instance != nil {
		return instance.Sync()
	}
	return nil
}

// Debug — обертка над методом логирования уровня Debug
func Debug(args ...interface{}) {
	Get().Debugln(args...)
}

// Info — обертка над методом логирования уровня Info
func Info(args ...interface{}) {
	Get().Infoln(args...)
}

// Warn — обертка над методом логирования уровня Warn
func Warn(args ...interface{}) {
	Get().Warnln(args...)
}

// Error — обертка над методом логирования уровня Error
func Error(args ...interface{}) {
	Get().Errorln(args...)
}

// Panic — обертка над методом логирования уровня Panic
func Panic(args ...interface{}) {
	Get().Panicln(args...)
}
