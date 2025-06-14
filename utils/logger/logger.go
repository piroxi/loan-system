package logger

import (
	"runtime"
	"slices"
	"sync"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	instance *zap.Logger
	once     sync.Once
	output   = []string{"stdout"}
)

func GetLogger() *zap.Logger {
	once.Do(func() {
		instance = newLogger()
	})
	return instance
}

func getLogLevel() zapcore.Level {
	return zap.DebugLevel
}

func newLogger() *zap.Logger {
	loggerConfig := zap.NewProductionConfig()
	loggerConfig.Level = zap.NewAtomicLevelAt(getLogLevel())
	loggerConfig.Encoding = "json"
	loggerConfig.DisableCaller = true
	loggerConfig.OutputPaths = output
	loggerConfig.EncoderConfig.FunctionKey = "func"
	if logger, err := loggerConfig.Build(); err != nil {
		panic(err)
	} else {
		return logger.WithOptions(zap.WithFatalHook(zapcore.CheckWriteAction(zap.PanicLevel)))
	}
}

func EnableCSVLogging() {
	if slices.Contains(output, "./logs.csv") {
		return
	}

	output = append(output, "./logs.csv")
	instance = newLogger()
}

func DisableCSVLogging() {
	output = []string{"stdout"}
	instance = newLogger()
}

// Debug logs the message at debug level with additional fields, if any
func Debug(message string, fields ...zap.Field) {
	GetLogger().Debug(message, fields...)
}

// Error logs the message at error level and prints stacktrace with additional fields, if any
func Error(message string, fields ...zap.Field) {
	fields = append(fields, zap.String("caller", getCallerFunctionName()))
	GetLogger().Error(message, fields...)
}

// Fatal logs the message at fatal level with additional fields, if any and exits
func Fatal(message string, fields ...zap.Field) {
	fields = append(fields, zap.String("caller", getCallerFunctionName()))
	GetLogger().Fatal(message, fields...)
}

// Info logs the message at info level with additional fields, if any
func Info(message string, fields ...zap.Field) {
	fields = append(fields, zap.String("caller", getCallerFunctionName()))
	GetLogger().Info(message, fields...)
}

// Warn logs the message at warn level with additional fields, if any
func Warn(message string, fields ...zap.Field) {
	fields = append(fields, zap.String("caller", getCallerFunctionName()))
	GetLogger().Warn(message, fields...)
}

func getCallerFunctionName() string {
	pc, _, _, _ := runtime.Caller(2)
	callerFunctionName := runtime.FuncForPC(pc).Name()
	return callerFunctionName
}
