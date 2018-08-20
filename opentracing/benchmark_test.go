package opentracing

import (
	"context"
	"errors"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"

	opentracing "github.com/opentracing/opentracing-go"

	"cd.splunkdev.com/libraries/go-observation/logging"
	"cd.splunkdev.com/libraries/go-observation/opentracing/lightstepx"
	"cd.splunkdev.com/libraries/go-observation/tracing"
)

// Provided solely as a relative value to compare against
func BenchmarkStringsRepeatBaseline(b *testing.B) {
	b.ResetTimer()
	b.ReportAllocs()
	for n := 0; n < b.N; n++ {
		strings.Repeat("LongMessage", 4096)
	}
	return
}

func BenchmarkShortMessage(b *testing.B) {
	logger := logging.NewWithOutput("span with short message", ioutil.Discard)
	logging.SetGlobalLogger(logger)
	tracer := lightstepx.NewTracer("test new tracer")
	defer lightstepx.Close(context.Background())
	spanLogger := NewSpanLoggerWithSpan(logger, nil)
	b.ResetTimer()
	b.ReportAllocs()
	for n := 0; n < b.N; n++ {
		span := tracer.StartSpan("operation")
		spanLogger := spanLogger.WithSpan(span)
		spanLogger.Info("event", "ShortMessage")
		span.Finish()
	}
	return
}

func BenchmarkLongMessage(b *testing.B) {
	logger := logging.NewWithOutput("span with long message", ioutil.Discard)
	logging.SetGlobalLogger(logger)
	tracer := lightstepx.NewTracer("test new tracer")
	defer lightstepx.Close(context.Background())
	spanLogger := NewSpanLoggerWithSpan(logger, nil)
	b.ResetTimer()
	b.ReportAllocs()
	for n := 0; n < b.N; n++ {
		span := tracer.StartSpan("operation")
		spanLogger = spanLogger.WithSpan(span)
		spanLogger.Info("event", strings.Repeat("LongMessage", 4096))
		span.Finish()
	}
	return
}

func BenchmarkShortErrorMessage(b *testing.B) {
	logger := logging.NewWithOutput("span with short error message", ioutil.Discard)
	logging.SetGlobalLogger(logger)
	tracer := lightstepx.NewTracer("test new tracer")
	defer lightstepx.Close(context.Background())
	spanLogger := NewSpanLoggerWithSpan(logger, nil)
	b.ResetTimer()
	b.ReportAllocs()
	for n := 0; n < b.N; n++ {
		span := tracer.StartSpan("operation")
		spanLogger = spanLogger.WithSpan(span)
		logger.Error(errors.New("ShortMessage"), "")
		span.Finish()
	}
	return
}

func BenchmarkLongErrorMessage(b *testing.B) {
	logger := logging.NewWithOutput("span with long error message", ioutil.Discard)
	logging.SetGlobalLogger(logger)
	tracer := lightstepx.NewTracer("test new tracer")
	defer lightstepx.Close(context.Background())
	spanLogger := NewSpanLoggerWithSpan(logger, nil)
	b.ResetTimer()
	b.ReportAllocs()
	for n := 0; n < b.N; n++ {
		span := tracer.StartSpan("operation")
		spanLogger = spanLogger.WithSpan(span)
		logger.Error(errors.New(strings.Repeat("LongMessage", 4096)), "")
		span.Finish()
	}
	return
}

func BenchmarkInjectHTTPRequestWithSpan(b *testing.B) {
	logger := logging.NewWithOutput("inject http request with span", ioutil.Discard)
	logging.SetGlobalLogger(logger)
	tracer := lightstepx.NewTracer("test inject http request with span")
	defer lightstepx.Close(context.Background())
	b.ResetTimer()
	b.ReportAllocs()

	topSpan := tracer.StartSpan("operation")
	spanContext := opentracing.ContextWithSpan(context.Background(), topSpan)
	r, _ := http.NewRequest("GET", "/tenant1/benchmark", nil)
	r.Header.Add(tracing.XRequestID, "abcde")
	for n := 0; n < b.N; n++ {
		span := tracer.StartSpan("operation")
		InjectHTTPRequestWithSpan(r.WithContext(spanContext))
		span.Finish()
	}
	return
}
