package logging

import (
	"io"
	"os"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

    "cd.splunkdev.com/libraries/go-observation/tracing"
)

// Constants for standard key names
const (
	CallstackKey = "callstack"
	ComponentKey = "component"
	ErrorKey     = "error"
	FileKey      = "location" // Deprecated: use LocationKey
	HostnameKey  = "hostname"
	LevelKey     = "level"
	LocationKey  = "location"
	MessageKey   = "message"
	RequestIdKey = tracing.RequestIdKey
	RequestIDKey = tracing.RequestIDKey
	ServiceKey   = "service"
	TimeKey      = "time"
	TenantKey    = tracing.TenantKey
	UrlKey       = "url" // Deprecated: use URLKey
	URLKey       = "url"
)

var globalLogger = NewNoOp()

// SetGlobalLogger sets the global logger. By default it is
// a no-op logger. Passing nil will panic.
func SetGlobalLogger(l *Logger) {
	if l == nil {
		panic("The global logger can not be nil")
	}
	globalLogger = l
}

// Global returns the global logger. The default logger
// is a no-op logger.
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
	return NewWithOutput(serviceName, os.Stdout)
}

// NewWithOutput constructs a new logger and writes output to writer.
// writer is an io.writer that also supports concurrent writes.
// The writer will be wrapped with zapcore.AddSync()
func NewWithOutput(serviceName string, writer io.Writer) *Logger {
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
	atomLevel := zap.NewAtomicLevelAt(zapcore.InfoLevel)
	stacktrace := zap.AddStacktrace(zap.FatalLevel)
	core := zapcore.NewCore(zapcore.NewJSONEncoder(encoderCfg), lockWriter(writer), &atomLevel)
	hostname, _ := os.Hostname()
	requiredFields := zap.Fields(
		zap.String(ServiceKey, serviceName),
		zap.String(HostnameKey, hostname))
	logger := zap.New(core, requiredFields, stacktrace, zap.AddCaller(), zap.AddCallerSkip(1))
	return &Logger{logger.Sugar(), &atomLevel}
}

// NewNoOp returns a no-op Logger that doesn't emit any logs. This is
// the default global logger.
func NewNoOp() *Logger {
	// Does not matter what level as this is NoOp.
	atomLevel := zap.NewAtomicLevelAt(zapcore.InfoLevel)
	logger := zap.New(nil)
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

// Flush ensures that all buffered messages are written. Normally it only needs to be called
// before program exit.
func (l *Logger) Flush() error {
	return l.sugared.Sync()
}

// Debug logs a message at DebugLevel
func (l *Logger) Debug(msg string, fields ...interface{}) {
	l.sugared.Debugw(msg, fields...)
}

// Info logs a message at InfoLevel
func (l *Logger) Info(msg string, fields ...interface{}) {
	l.sugared.Infow(msg, fields...)
}

// Error logs a message at ErrorLevel. err is traced as {"error": err.Error()}
func (l *Logger) Error(err error, msg string, fields ...interface{}) {
	fields = append(fields, ErrorKey, err)
	l.sugared.Errorw(msg, fields...)
}

// Warn logs a message at WarnLevel
func (l *Logger) Warn(msg string, fields ...interface{}) {
	l.sugared.Warnw(msg, fields...)
}

// Fatal logs a message at FatalLevel and then calls os.Exit(1). err is traced
// as {"error": err.Error()}
func (l *Logger) Fatal(err error, msg string, fields ...interface{}) {
	fields = append(fields, ErrorKey, err)
	l.sugared.Fatalw(msg, fields...)
}

// DebugEnabled returns true if the logger level is debug or lower.
// It is a shortcut for Enabled(logging.DebugLevel)
func (l *Logger) DebugEnabled() bool {
	return l.level.Enabled(zapcore.DebugLevel)
}

// Enabled returns true if the specified log level is enabled
func (l *Logger) Enabled(level Level) bool {
	return l.level.Enabled(zapcore.Level(level))
}

// SetLevel sets the specified log level. The level will be modified for all loggers
// cloned from the same root parent logger.
func (l *Logger) SetLevel(level Level) {
	l.level.SetLevel(zapcore.Level(level))
}

// Level gets the log level
func (l *Logger) Level() Level {
	return Level(l.level.Level())
}

// With constructs a clone of logger with the added fields.. fields are
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

// lockWriter converts anything that implements io.Writer to WriteSyncer.
// If input already implements Sync(), it will just pass through.
func lockWriter(w io.Writer) zapcore.WriteSyncer {
	// If w already is a WriteSyncer, it won't wrap that again.
	writer := zapcore.AddSync(w)
	return zapcore.Lock(writer)
}
