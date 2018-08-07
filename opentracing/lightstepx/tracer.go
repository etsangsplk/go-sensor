package lightstepx

import (
	"context"
	"os"

	lightstep "github.com/lightstep/lightstep-tracer-go"
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
	return NewTracerWithOptions(serviceName, logger)
}

// NewTracer returns an instance of Tracer associated with serviceName, and all spans
// from this tracer will be tagged with this service name.
// It sends all spans to reporter and writes the reported spans to logger.
// You need 1 global tracer per microservice.
func NewTracerWithOptions(serviceName string, logger *logging.Logger, opts ...Option) opentracing.Tracer {
	// setup event handler first to catch startup errors
	// logger := logging.Global()
	lightstep.SetGlobalEventHandler(logTracerEventHandler)
	for _, opt := range opts {
		opt(options)

	}

	o := lightstep.Options(*options)
	err := o.Initialize()

	if err != nil {
		logger.Warn("fail to configure tracer", "message", err.Error())
		return globalTracer
	}
	// Tag the informtion about server data to the tracer, for easier data correlation with splunk search.
	hostname, _ := os.Hostname()
	requiredTags := []opentracing.Tag{
		opentracing.Tag{lightstep.ComponentNameKey, serviceName},
		opentracing.Tag{logging.HostnameKey, hostname},
		opentracing.Tag{logging.ServiceKey, serviceName},
	}
	o.Tags = make(map[string]interface{}, len(requiredTags))
	for _, tag := range requiredTags {
		o.Tags[tag.Key] = tag.Value
	}
	return lightstep.NewTracer(o)
}

// Flush flushes buffer of tracer derived from context.
// If no tracer is found, it is an noop.
func Flush(ctx context.Context) {
	t := opentracing.GlobalTracer()
	l, ok := t.(lightstep.Tracer)
	if !ok {
		return
	}
	l.Flush(ctx)
}

// Close flush buffer of tracer derived from context and terminates collector.
// If no tracer is found or calling it multiple times, it is an noop.
func Close(ctx context.Context) {
	t := opentracing.GlobalTracer()
	l, ok := t.(lightstep.Tracer)
	if !ok {
		return
	}
	l.Close(ctx)
}
