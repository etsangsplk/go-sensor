package opencensus

import (
	"errors"

	opentracing "github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/log"

	"cd.splunkdev.com/libraries/go-observation/logging"
)

// SpanLogger is a convenience wrapper so user can log to tracer and logging library at the same time.
// It is by no means to replace logger from logging library. It is also NOT a structured level logger.
// Since logging to a span only concerns information that is useful about a span, which is not same as application logging,
// only INFO and ERROR are offered.
// Also note that logging to a span only handles basic types and does not handle the time duration or complex number type like
// application logger.
type SpanLogger struct {
	base *logging.Logger
	span opentracing.Span
}

// NewSpanLoggerWithSpan constructs a new SpanLogger and writes to both tracer and logging
// library logger.
func NewSpanLoggerWithSpan(logger *logging.Logger, span opentracing.Span) *SpanLogger {
	if logger == nil {
		// Sets to Global so won't panic on nil.
		logger = logging.Global()
	}
	return &SpanLogger{
		base: logger,
		span: span,
	}
}

// Info logs informational event about a span.
func (s *SpanLogger) Info(msg string, fields ...interface{}) {
	s.logToSpan(logging.InfoLevel.String(), msg, nil, fields...)
	s.base.Info(msg, fields...)
}

// Warn logs error event that does not lead to a span in error state.
func (s *SpanLogger) Warn(msg string, fields ...interface{}) {
	s.logToSpan(logging.WarnLevel.String(), msg, nil, fields...)
	s.base.Warn(msg, fields...)
}

// Error logs error event that leads to a span in error state.
// It will set the span in Error state.
func (s *SpanLogger) Error(err error, msg string, fields ...interface{}) {
	s.logToSpan(logging.ErrorLevel.String(), msg, err, fields...)
	SetSpanError(s.span)
	s.base.Error(err, msg, fields...)
}

// WithSpan sets span instance to a spanLogger and returns the resultant SpanLogger instance.
// Subsequent call to SpanLogger will act on this new span.
func (s *SpanLogger) WithSpan(span opentracing.Span) *SpanLogger {
	s.span = span
	return s
}

// logToSpan constructs a set of log entries that is of key/value pair before sending to underlying span logKV.
func (s *SpanLogger) logToSpan(level string, msg string, err error, fields ...interface{}) {
	if s.span == nil {
		s.base.Error(errors.New("span cannot be nil"), "logger", "spanLogger")
		return
	}

	l := 0
	// Delegate to opentracing to check for proper KV fields.
	opentracingFields, err1 := log.InterleavedKVToFields(fields...)

	if err1 != nil {
		s.base.Error(err, "invalid fields for span logger")
		return
	}

	// add error, error message entry pair.
	if err != nil {
		l = len(opentracingFields) + 2
	}

	l = len(opentracingFields)
	fa := make([]log.Field, 0, 2+l)

	if err != nil {
		// opentracing recommended way of logging error event
		fa = append(fa, log.Error(err))
	}
	// opentracing recommended way of handling any event that is not error.Error
	// record the event and the logging level.
	fa = append(fa, log.String("event", msg), log.String("level", level))
	fa = append(fa, opentracingFields...)

	s.span.LogFields(fa...)
}
