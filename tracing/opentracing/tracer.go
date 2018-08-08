package opentracing

import (
	"fmt"
	"io"

	opentracing "github.com/opentracing/opentracing-go"
	jaeger "github.com/uber/jaeger-client-go"
	config "github.com/uber/jaeger-client-go/config"
	jaegerLogger "github.com/uber/jaeger-client-go/log"
)

const (
	TraceContextSpanTraceIDKey = "ssc-trace-id"
	TraceBaggageHeaderPrefix   = "sscctx-"
)

type Tracer opentracing.Tracer
type TraceLogger jaegerLogger.Logger

var (
	defaultLogger = jaegerLogger.NullLogger
	globalTracer  = &opentracing.NoopTracer{}
)

// SetGlobalTracer sets the global logger.
// By default tracer is a no-op tracer. Passing nil tracer will panic.
func SetGlobalTracer(t Tracer) {
	if t == nil {
		panic("The global tracer can not be nil")
	}
	opentracing.SetGlobalTracer(t)
}

// Global returns the global tracer. The default is a no-op tracer.
func Global() Tracer {
	return opentracing.GlobalTracer()
}

// NewTracer returns an instance of Tracer associated with serviceName, and all spans
// from this tracer will be tagged with this service name.
// It sends all spans to reporter and writes the reported spans to logger.
// You need 1 global tracer per microservice.
// It also returns a closer func to be used to flush buffers before shutdown.
func NewTracer(serviceName string, logger TraceLogger) (Tracer, io.Closer) {
	sampler := &config.SamplerConfig{
		Type:  "const",
		Param: 1, // This reports 100%. Need to let user choose. Functional Options?
	}
	reporter := &config.ReporterConfig{
		LogSpans: true,
	}

	return newTracer(serviceName, sampler, reporter, logger)
}

// newTracer returns an instance of Tracer associated with serviceName.
// It also allows configuration for sampler to only subet of spans out, reporter for different types of reporters.
// It also returns a closer func to be used to flush buffers before shutdown.
func newTracer(serviceName string, sampler *config.SamplerConfig, reporter *config.ReporterConfig, logger TraceLogger) (Tracer, io.Closer) {
	cfg := &config.Configuration{
		Sampler:  sampler,
		Reporter: reporter,
		// Override jaegers default of tagging using uber.
		Headers: &jaeger.HeadersConfig{
			TraceContextHeaderName:   TraceContextSpanTraceIDKey,
			TraceBaggageHeaderPrefix: TraceBaggageHeaderPrefix,
		},
	}
	tracer, closer, err := cfg.New(serviceName, config.Logger(logger))
	if err != nil {
		panic(fmt.Sprintf("cannot init Jaeger error: %v\n", err))
	}
	return tracer, closer

}
