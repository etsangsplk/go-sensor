package opentracing

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	config "github.com/uber/jaeger-client-go/config"

	"github.com/splunk/ssc-observation/logging"
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

	tracer, closer := NewTracer("testNew", logger)
	defer closer.Close()

	span := tracer.StartSpan("testNewSpan")
	span.SetOperationName("TestNewTracerWithTraceLogger")
	span.LogKV("event", "test NewTracerWithTraceLogger", "type", "unit test", "waited.millis", 1500)
	// Must finish span before validation.
	span.Finish()

	s := StopLogCapturing(outC, w)
	fmt.Printf("s: %#v", s)
	assert.NotNil(t, tracer)
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

	tracer, closer := newTracer("some service", sampler, reporter, logger)
	defer closer.Close()

	span := tracer.StartSpan("testNewSpan")
	span.SetOperationName("some operation")
	span.LogKV("event", "test NewTracerWithTraceLogger", "type", "unit test", "waited.millis", 1500)
	// Must finish span before validation.
	span.Finish()

	s := StopLogCapturing(outC, w)
	fmt.Printf("s: %#v", s)

	assert.NotNil(t, tracer)
	assert.NotEmpty(t, span.Context())
	// Since there is no API to get SpanID, use the following to infer spanID is nonempty
	spanCtx := fmt.Sprintf("%#v \n", span.Context())
	fmt.Printf(spanCtx)
	assert.NotContains(t, spanCtx, `spanID:""`)
	assert.Contains(t, spanCtx, `spanID:`)
	// This is the first span, so got no parentID.
	assert.Contains(t, spanCtx, `parentID:0x0`)
	assert.Contains(t, s[0], `"message":"Initializing logging reporter\n"`)
	assert.Contains(t, s[0], `"service":"testlogger"`)
}
