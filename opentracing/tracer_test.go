package opentracing

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	config "github.com/uber/jaeger-client-go/config"

	"cd.splunkdev.com/libraries/go-observation/logging"
)

func StartLogCapturing() (chan string, *os.File) {
	r, w, _ := os.Pipe()
	outC := make(chan string)

	// copy the output in a separate goroutine so printing can't block indefinitely
	go func() {
		var buf bytes.Buffer
		_, e := io.Copy(&buf, r)
		if e != nil {
			panic(e)
		}
		outC <- buf.String()
	}()
	return outC, w
}

func StopLogCapturing(outChannel chan string, writeStream *os.File) []string {
	// back to normal state
	if e := writeStream.Close(); e != nil {
		panic(e)
	}

	logOutput := <-outChannel

	// Verify call stack contains information we care about
	s := strings.Split(logOutput, "\n")
	return s
}

func TestNewTracerWithTraceLogger(t *testing.T) {
	outC, w := StartLogCapturing()
	logger := logging.NewWithOutput("testlogger", w)

	tracer, closer := NewTracer("test new tracer", logger)
	defer closer.Close()

	span := tracer.StartSpan("testNewSpan")
	span.SetOperationName("TestNewTracerWithTraceLogger")
	span.LogKV("event", "test NewTracerWithTraceLogger", "type", "unit test", "waited.millis", 1500)
	// Must finish span before validation.
	span.Finish()

	s := StopLogCapturing(outC, w)
	assert.NotNil(t, tracer)
	// Check that some signs of reporter being initialized and that
	// a span ie being reported. SpanID is changing so just do inference.
	assert.Contains(t, s[0], `"message":"Initializing logging reporter\n"`)
	assert.Contains(t, s[0], `"service":"testlogger"`)
}

func TestNewTracerWithSamplerReporter(t *testing.T) {
	outC, w := StartLogCapturing()
	logger := logging.NewWithOutput("testlogger", w)

	sampler := &config.SamplerConfig{
		Type:  "const",
		Param: 1, // This reports 100%. Need to let user choose. Functional Options?
	}
	reporter := &config.ReporterConfig{
		LogSpans: true,
	}

	tracer, closer := newTracer("test span creation", sampler, reporter, logger)
	defer closer.Close()

	span := tracer.StartSpan("test new span")
	span.LogKV("event", "new pan created", "type", "unit test", "waited.millis", 1500)
	// Must finish span before validation.
	span.Finish()

	s := StopLogCapturing(outC, w)

	assert.NotNil(t, tracer)
	assert.NotEmpty(t, span.Context())
	// Since there is no API to get SpanID, use the following to infer spanID is nonempty
	spanCtx := fmt.Sprintf("%#v \n", span.Context())
	assert.NotContains(t, spanCtx, `spanID:""`)
	assert.Contains(t, spanCtx, `spanID:`)
	// This is the first span, so got no parentID.
	assert.Contains(t, spanCtx, `parentID:0x0`)
	// Check that some signs of reporter being initialized and that
	// a span ie being reported. SpanID is changing so just do inference.
	assert.Contains(t, s[0], `"message":"Initializing logging reporter\n"`)
	assert.Contains(t, s[0], `"service":"testlogger"`)
}

func TestSpanCreation(t *testing.T) {
	outC, w := StartLogCapturing()
	logger := logging.NewWithOutput("testlogger", w)

	sampler := &config.SamplerConfig{
		Type:  "const",
		Param: 1, // This reports 100%. Need to let user choose. Functional Options?
	}
	reporter := &config.ReporterConfig{
		LogSpans: true,
	}

	// Important, if you are using a MockTracer, childSpan will return as noopSpan
	// internally the tracer looks for Global() tracer for creation after finding a parent
	// exists from context. Make sure all mockTracer is not set as Global()
	tracer, closer := newTracer("test multiple span creation", sampler, reporter, logger)
	SetGlobalTracer(tracer)
	defer closer.Close()

	ctx := context.Background()
	parentSpan := tracer.StartSpan("parent span")
	parentSpan.LogKV("event", "parent span created")

	// create a child span from parent span context
	parentCtx := ContextWithSpan(ctx, parentSpan)
	childSpan, _ := StartSpanFromContext(parentCtx, "child span")
	childSpan.LogKV("event", "child span created")
	// Must finish span before validation.
	parentSpan.Finish()
	childSpan.Finish()

	s := StopLogCapturing(outC, w)

	// Since there is no API to get SpanID, use the following to infer spanID is nonempty
	parentSpanCtxStr := fmt.Sprintf("%#v \n", parentSpan.Context())
	childSpanCtxStr := fmt.Sprintf("%#v \n", childSpan.Context())

	assert.NotContains(t, parentSpanCtxStr, `spanID:""`)
	assert.Contains(t, parentSpanCtxStr, `spanID:`)
	// This is the first span, so parentID should be 0x0.
	assert.Contains(t, parentSpanCtxStr, `parentID:0x0`)

	// Child SpanId should contain some value
	assert.NotContains(t, childSpanCtxStr, `spanID:""`)
	assert.Contains(t, childSpanCtxStr, `spanID:`)
	// ParentID should not be 0x0
	assert.NotContains(t, childSpanCtxStr, `parentID:0x0`)

	// Check that some signs of reporter being initialized and that
	// a span ie being reported. SpanID is changing so just do inference.
	assert.Contains(t, s[0], `"message":"Initializing logging reporter\n"`)
	assert.Contains(t, s[0], `"service":"testlogger"`)
	// we should have reported 2 spans
	assert.Contains(t, s[1], `"message":"Reporting span`)
	assert.Contains(t, s[2], `"message":"Reporting span`)
}
