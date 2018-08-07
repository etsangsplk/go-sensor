package opentracing

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	config "github.com/uber/jaeger-client-go/config"
	jaegerLogger "github.com/uber/jaeger-client-go/log"
)

func TestNewTracerWithTraceLogger(t *testing.T) {
	bufferLogger := &jaegerLogger.BytesBufferLogger{}
	tracer, closer := NewTracer("testNew", bufferLogger)
	defer closer.Close()

	span := tracer.StartSpan("testNewSpan")
	span.SetOperationName("TestNewTracerWithTraceLogger")
	span.LogKV("event", "test NewTracerWithTraceLogger", "type", "unit test", "waited.millis", 1500)
	// Must finish span before validation.
	span.Finish()
	assert.NotNil(t, tracer)
	assert.Contains(t, bufferLogger.String(), "Reporting span")

}

func TestNewTracerWithSamplerReporter(t *testing.T) {
	bufferLogger := &jaegerLogger.BytesBufferLogger{}

	sampler := &config.SamplerConfig{
		Type:  "const",
		Param: 1, // This reports 100%. Need to let user choose. Functional Options?
	}
	reporter := &config.ReporterConfig{
		LogSpans: true,
	}

	tracer, closer := newTracer("some service", sampler, reporter, bufferLogger)
	defer closer.Close()

	span := tracer.StartSpan("testNewSpan")
	span.SetOperationName("some operation")
	span.LogKV("event", "test NewTracerWithTraceLogger", "type", "unit test", "waited.millis", 1500)
	// Must finish span before validation.
	span.Finish()

	assert.NotNil(t, tracer)
	//fmt.Printf("bufferLogger: %#v \n", bufferLogger.String())
	assert.Contains(t, bufferLogger.String(), "Reporting span")
	assert.NotEmpty(t, span.Context())
	// Since there is no API to get SpanID, use the following to infer spanID is nonempty
	spanCtx := fmt.Sprintf("%#v \n", span.Context())
	fmt.Printf(spanCtx)
	assert.NotContains(t, spanCtx, `spanID:""`)
	assert.Contains(t, spanCtx, `spanID:`)
	// This is the first span, so got no parentID.
	assert.Contains(t, spanCtx, `parentID:0x0`)
}
