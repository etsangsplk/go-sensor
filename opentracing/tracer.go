package opentracing

import (
	opentracing "github.com/opentracing/opentracing-go"
)

// SetGlobalTracer sets the global logger.
// By default tracer is a no-op tracer. Passing nil tracer will panic.
func SetGlobalTracer(t opentracing.Tracer) {
	if t == nil {
		panic("The global tracer can not be nil")
	}
	opentracing.SetGlobalTracer(t)
}

// Global returns the global tracer. The default is a no-op tracer.
func Global() opentracing.Tracer {
	return opentracing.GlobalTracer()
}
