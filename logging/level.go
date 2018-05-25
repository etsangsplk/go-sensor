package logging

import (
	"go.uber.org/zap/zapcore"
)

type Level zapcore.Level

const (
	DebugLevel = Level(zapcore.DebugLevel)
	InfoLevel  = Level(zapcore.InfoLevel)
	WarnLevel  = Level(zapcore.WarnLevel)
	ErrorLevel = Level(zapcore.ErrorLevel)
	FatalLevel = Level(zapcore.FatalLevel)
)

// String converts log severity level to string
func (level Level) String() string {
	return zapcore.Level(level).String()
}

// ParseLevel converts a level string into a Level value.
// returns an error if the input string does not match known values.
func ParseLevel(levelStr string) (Level, error) {
	var level Level
	err := level.UnmarshalText([]byte(levelStr))
	return level, err
}

// MarshalText marshals the level into its string representation and returns it as a byte array.
// Text strings are one of the following: "debug", "info", "warn", "error", "fatal"
func (level *Level) MarshalText() (text []byte, err error) {
	return zapcore.Level(*level).MarshalText()
}

// UnmarshalText unmarshals a byte array representing a log string into a log level object and assigns it to the level.
// Text strings must be one of the following: "debug", "info", "warn", "error", "fatal"
func (level *Level) UnmarshalText(text []byte) error {
	zapLvl := zapcore.Level(*level)
	err := zapLvl.UnmarshalText(text)
	*level = Level(zapLvl)
	return err
}
