package logging

import (
	"os"
    "time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Level = zapcore.Level
type WriteSyncer = zapcore.WriteSyncer

const (
	DebugLevel = zapcore.DebugLevel
	InfoLevel  = zapcore.InfoLevel
	WarnLevel  = zapcore.WarnLevel
	ErrorLevel = zapcore.ErrorLevel
	FatalLevel = zapcore.FatalLevel
)

// Constants for standard field key names
const (
	CallstackKey = "callstack"
	ComponentKey = "component"
	ErrorKey     = "error"
	FileKey      = "file"
	LevelKey     = "level"
	MessageKey   = "message"
	RequestIdKey = "requestId"
	ServiceKey   = "service"
	TimeKey      = "time"
	TenantKey    = "tenant"
	UrlKey       = "url"
)

var globalLogger *Logger

// SetGlobalLobber sets the global logger
func SetGlobalLogger(l *Logger) {
	globalLogger = l
}

// Global returns the global logger
func Global() *Logger {
	return globalLogger
}

// A Logger provides performant, leveled, structured logging.
type Logger struct {
	sugared *zap.SugaredLogger
	level   *zap.AtomicLevel
}

// New constructs a new logger using the default stdout for output.
// The serviceName argument will be traced as the standard "service"
// field on every trace.
func New(serviceName string) *Logger {
	return NewWithOutput(serviceName, Lock(os.Stdout))
}

// NewWithOutput constructs a new logger and writes output to writer
func NewWithOutput(serviceName string, writer WriteSyncer) *Logger {
	encoderCfg := zapcore.EncoderConfig{
		MessageKey:     MessageKey,
		LevelKey:       LevelKey,
		NameKey:        "logger",
		TimeKey:        TimeKey,
		StacktraceKey:  CallstackKey,
		CallerKey:      FileKey,
		EncodeLevel:    zapcore.CapitalLevelEncoder,
		EncodeTime:     utcTimeEncoder,
		EncodeDuration: zapcore.StringDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}
	atomLevel := zap.NewAtomicLevelAt(InfoLevel)
	core := zapcore.NewCore(zapcore.NewJSONEncoder(encoderCfg), writer, &atomLevel)
	requiredFields := zap.Fields(
		zap.String("service", serviceName),
		zap.String("hostname", os.Getenv("HOSTNAME")))
	stacktrace := zap.AddStacktrace(zap.FatalLevel)
	logger := zap.New(core, requiredFields, stacktrace, zap.AddCaller(), zap.AddCallerSkip(1))
	return &Logger{logger.Sugar(), &atomLevel}
}

// utcTimeEncoder encodes the time as a UTC ISO8601 timestamp
func utcTimeEncoder(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString(t.UTC().Format("2006-01-02T15:04:05.000Z"))
}

// Make a shallow copy of Logger
func (l *Logger) clone() *Logger {
	copy := *l
	return &copy
}

// SetCallstackSkip returns a clone of the logger with the callstack skip set to skip. Use this
// when wrapping the logger.
func (l *Logger) SetCallstackSkip(skip int) *Logger {
	newLogger := l.sugared.Desugar().WithOptions(zap.AddCallerSkip(skip))
	return &Logger{newLogger.Sugar(), l.level}
}

// Flush ensures that all buffered messages are written.
func (l *Logger) Flush() {
	l.sugared.Sync()
}

// Debug logs a message at DebugLevel
func (l *Logger) Debug(msg string, fields ...interface{}) {
	l.sugared.Debugw(msg, fields...)
}

// Info logs a message at InfoLevel
func (l *Logger) Info(msg string, fields ...interface{}) {
	l.sugared.Infow(msg, fields...)
}

// Error logs a message at ErrorLevel. The err is traced as {"error": err.Error()}
func (l *Logger) Error(err error, msg string, fields ...interface{}) {
	fields = append(fields, ErrorKey, err)
	l.sugared.Errorw(msg, fields...)
}

// Warn logs a message at WarnLevel
func (l *Logger) Warn(msg string, fields ...interface{}) {
	l.sugared.Warnw(msg, fields...)
}

// Fatal logs a message at FatalLevel and then calls os.Exit(1). The err is traced
// as {"error": err.Error()}
func (l *Logger) Fatal(err error, msg string, fields ...interface{}) {
	fields = append(fields, ErrorKey, err)
	l.sugared.Fatalw(msg, fields...)
}

// DebugEnabled returns true if the debug log level or lower is enabled.
// It is a shortcut for Enabled(logging.DebugLevel)
func (l *Logger) DebugEnabled() bool {
	return l.level.Enabled(DebugLevel)
}

// Enabled returns true if the specified log level is enabled
func (l *Logger) Enabled(level Level) bool {
	return l.level.Enabled(level)
}

// SetLevel sets the specified log level. The level will be modified for all loggers
// cloned from the same root parent logger.
func (l *Logger) SetLevel(level Level) {
	l.level.SetLevel(level)
}

// Level gets the log level
func (l *Logger) Level() Level {
	return l.level.Level()
}

// With constructs a clone of logger with the addition of fields which are
// key-value pairs that will be included in each trace.
// The return value is the new logger which shares the same atomic level
// as the parent.
func (l *Logger) With(fields ...interface{}) *Logger {
	if len(fields) == 0 {
		return l
	}
	child := l.clone()
	child.sugared = l.sugared.With(fields...)
	return child
}
