package jaegerx

import (
	"io"
	// "io/ioutil"
	"context"

	opentracing "github.com/opentracing/opentracing-go"
	jaegerConfig "github.com/uber/jaeger-client-go/config"

	"cd.splunkdev.com/libraries/go-observation/logging"
)

var globalTracer = &opentracing.NoopTracer{}

type nullCloser struct{}

func (*nullCloser) Close() error { return nil }

// NewTracer returns an instance of Tracer associated with serviceName, and all spans
// from this tracer will be tagged with this service name.
// It sends all spans to reporter and writes the reported spans to logger.
// You need 1 global tracer per microservice.
func NewTracer(serviceName string) (opentracing.Tracer, io.Closer, error) {
	logger := logging.Global()
	config, err := jaegerConfig.FromEnv()
	if err != nil {
		logger.Error(err, "error loading tracer config")
		return globalTracer, &nullCloser{}, err
	}
	if len(serviceName) > 0 {
		config.ServiceName = serviceName
	}

	return config.New(serviceName)
}

// NewTracer returns an instance of Tracer associated with serviceName, and all spans
// from this tracer will be tagged with this service name.
// It sends all spans to reporter and writes the reported spans to logger.
// You need 1 global tracer per microservice.
func NewTracerWithOptions(serviceName string, opts ...jaegerConfig.Option) (opentracing.Tracer, io.Closer, error) {
	c := &jaegerConfig.Configuration{}
	// setup event handler first to catch startup errors
	closer, err := c.InitGlobalTracer(serviceName, opts...)
	t := opentracing.GlobalTracer()
	return t, closer, err
}

// Flush Jaeger.
func Flush(c io.Closer, ctx context.Context) {
	Close(c, ctx)
}

// Close Jaeger.
func Close(c io.Closer, ctx context.Context) {
	c.Close()
}
