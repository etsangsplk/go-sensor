package opentracing

import (
	"fmt"

	"github.com/splunk/ssc-observation/logging"
	jlog "github.com/uber/jaeger-client-go/log"
)

// This is a adaptor for ssc logging to jaeger reporter logger.
// so that same logging format is used.
type Logger struct {
	jlog.Logger
	logger *logging.Logger
}

// NewLogger creates a new Logger for tracer logger reporter.
// Information like SpanId will be sent to our ssc logging
// for record.
func NewLogger(logger *logging.Logger) *Logger {
	return &Logger{logger: logger}
}

// Error logs an error message
func (l *Logger) Error(msg string) {
	l.logger.Error(fmt.Errorf(msg), msg)
}

// Infof logs a info message
func (l *Logger) Infof(msg string, fields ...interface{}) {
	l.logger.Info(fmt.Sprintf(msg, fields...))
}
