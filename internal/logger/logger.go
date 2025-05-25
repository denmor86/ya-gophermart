package logger

import (
	"sync"

	"go.uber.org/zap"
)

var (
	once     sync.Once
	instance *zap.SugaredLogger = nil
)

// Initialize - инициализирует синглтон логера с необходимым уровнем логирования.
func Initialize(level string) error {
	// преобразуем текстовый уровень логирования в zap.AtomicLevel
	lvl, err := zap.ParseAtomicLevel(level)
	if err != nil {
		return err
	}
	// создаём новую конфигурацию логера
	cfg := zap.NewProductionConfig()
	// устанавливаем уровень
	cfg.Level = lvl
	// создаём логер на основе конфигурации
	logger, err := cfg.Build()
	if err != nil {
		return err
	}
	// устанавливаем синглтон
	instance = logger.Sugar()
	return nil
}

// Get - метод получения объекта логгера из синглтона
func Get() *zap.SugaredLogger {
	if instance == nil {
		panic("logger not initialized, call Initialize()")
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
