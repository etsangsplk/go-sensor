package opencensus

import (
	"context"
	"errors"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"

	opentracing "github.com/opentracing/opentracing-go"
	ot "github.com/opentracing/opentracing-go"

	"cd.splunkdev.com/libraries/go-observation/logging"
	"cd.splunkdev.com/libraries/go-observation/opentracing/instanax"
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
	tracer := getTracer("test short message")
	defer closeTracer(context.Background())
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

	tracer := getTracer("test long message")
	defer closeTracer(context.Background())

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

	tracer := getTracer("test short error messsage")
	defer closeTracer(context.Background())

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

	tracer := getTracer("test long error messsage tracer")
	defer closeTracer(context.Background())

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

	tracer := getTracer("test inject http request with span")
	defer closeTracer(context.Background())

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

func getTracer(serviceName string) ot.Tracer {
	if lightstepx.Enabled() && instanax.Enabled() {
		logger := logging.Global()
		logger.Fatal(errors.New("cannot enable both Lighstep and Instana"), "use either Lightstep or Instana")
	}
	if lightstepx.Enabled() {
		return lightstepx.NewTracer(serviceName)
	}
	if instanax.Enabled() {
		return instanax.NewTracer(serviceName)
	}
	return ot.GlobalTracer()
}

func closeTracer(ctx context.Context) {
	if lightstepx.Enabled() {
		lightstepx.Close(ctx)
	}
	if instanax.Enabled() {
		instanax.Close(ctx)
	}
}