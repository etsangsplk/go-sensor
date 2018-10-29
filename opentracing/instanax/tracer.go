package instana

import (
	"context"

	instana "github.com/instana/golang-sensor"
	opentracing "github.com/opentracing/opentracing-go"

	"cd.splunkdev.com/libraries/go-observation/logging"
)

var globalTracer = &opentracing.NoopTracer{}

// NewTracer returns an instance of Tracer associated with serviceName, and all spans
// from this tracer will be tagged with this service name.
// It sends all spans to reporter and writes the reported spans to logger.
// You need 1 global tracer per microservice.
func NewTracer(serviceName string) opentracing.Tracer {
	logger := logging.Global()
	err := LoadConfig()
	if err != nil {
		logger.Error(err, "error loading tracer config")
		return globalTracer
	}
	return NewTracerWithOptions(serviceName)
}

// NewTracer returns an instance of Tracer associated with serviceName, and all spans
// from this tracer will be tagged with this service name.
// It sends all spans to reporter and writes the reported spans to logger.
// You need 1 global tracer per microservice.
func NewTracerWithOptions(serviceName string, opts ...Option) opentracing.Tracer {
	// setup event handler first to catch startup errors
	options := &Options{}
	for _, opt := range opts {
		opt(options)
	}
	WithServiceName(serviceName)(options)
	return instana.NewTracerWithOptions(&options.Opts)
}

// Flush noop for Instana. Instana does not offers Flush.
func Flush(ctx context.Context) {
	return
}

// Close is noop for Instana. Instana does not offers Close.
func Close(ctx context.Context) {
	return
}
