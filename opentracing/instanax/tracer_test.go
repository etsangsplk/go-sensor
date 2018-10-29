package instanax

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"

	instana "github.com/instana/golang-sensor"
	opentracing "github.com/opentracing/opentracing-go"
	"github.com/stretchr/testify/assert"

	"cd.splunkdev.com/libraries/go-observation/logging"
	ot "cd.splunkdev.com/libraries/go-observation/opentracing"
)

func StartLogCapturing() (chan string, *os.File) {
	r, w, _ := os.Pipe()
	outC := make(chan string)

	// copy the output in a separate goroutine so printing can't block indefinitely
	go func() {
		var buf bytes.Buffer
		_, e := io.Copy(&buf, r)
		if e != nil {
			fmt.Printf("write stream panic: %v \n", e)
			panic(e)
		}
		outC <- buf.String()
	}()
	return outC, w
}

func StopLogCapturing(outChannel chan string, writeStream *os.File) []string {
	// back to normal state
	if e := writeStream.Close(); e != nil {
		fmt.Printf("closing write stream panic: %v \n", e)
		panic(e)
	}

	logOutput := <-outChannel

	// Verify call stack contains information we care about
	s := strings.Split(logOutput, "\n")
	return s
}

func SetupTestEnvironmentVar() {
	os.Setenv(string(EnvInstanaAgentHost), "127.0.0.1")
	os.Setenv(string(EnvInstanaAgentPort), "8080")
}

func MockInstanaTracer(serviceName string, recorder instana.SpanRecorder) opentracing.Tracer {
	opts := &Options{}
	agentHost := getenvRequired(EnvInstanaAgentHost)
	agentPort := int(getenvRequiredInt64(EnvInstanaAgentPort))
	WithServiceName(serviceName)(opts)

	WithAgentEndpoint(agentHost, agentPort)(opts)
	return instana.NewTracerWithEverything(&opts.Opts, recorder)
}

func TestNewTracerWithTraceLogger(t *testing.T) {
	env := StashEnv()
	defer PopEnv(env)
	SetupTestEnvironmentVar()
	g := SaveGlobalTracer()
	defer RestoreGlobalTracer(g)

	serviceName := "test new tracer"
	outC, w := StartLogCapturing()
	logger := logging.NewWithOutput(serviceName, w)
	logging.SetGlobalLogger(logger)

	recorder := instana.NewTestRecorder()

	tracer := MockInstanaTracer(serviceName, recorder)

	ot.SetGlobalTracer(tracer)
	hostname, _ := os.Hostname()
	span := tracer.StartSpan("testNewSpan", opentracing.Tags{logging.HostnameKey: hostname})
	span.SetOperationName("TestNewTracerWithTraceLogger")
	span.LogKV("event", "test NewTracerWithTraceLogger", "type", "unit test", "waited.millis", 1500)
	// Must finish span before validation.
	span.Finish()

	spans := recorder.GetQueuedSpans()
	StopLogCapturing(outC, w)

	assert.NotNil(t, tracer)
	// Check that some signs of reporter being initialized and that
	// a span ie being reported. SpanID is changing so just do inference.
	recordedSpan := spans[0]
	assert.NotZero(t, recordedSpan.SpanID, "Missing span ID for this span")
	assert.NotZero(t, recordedSpan.TraceID, "Missing trace ID for this span")
	assert.NotZero(t, recordedSpan.Timestamp, "Missing timestamp for this span")
	assert.NotNil(t, recordedSpan.Duration, "Duration is nil")
	// There is something wrong with recorder from instana
	// assert.Equal(t, serviceName, recordedSpan.Data.Service, "Missing span service name")

	assert.Equal(t, "TestNewTracerWithTraceLogger", recordedSpan.Data.SDK.Name, "Missing span operation name")
	assert.Contains(t, recordedSpan.Data.SDK.Custom.Tags[logging.HostnameKey], hostname, "Missing hostname")
}

// If you have no GlobalTracer, SpanFromContext will not work, internally it depends on global tracer.
func SaveGlobalTracer() opentracing.Tracer {
	return ot.Global()
}

// Restore the Global tracer or else other tests will be affected.
func RestoreGlobalTracer(t opentracing.Tracer) {
	ot.SetGlobalTracer(t)
}
