package instanax

import (
	"os"
	"testing"

	instana "github.com/instana/golang-sensor"
	opentracing "github.com/opentracing/opentracing-go"
	"github.com/stretchr/testify/assert"

	"cd.splunkdev.com/libraries/go-observation/logging"
	ot "cd.splunkdev.com/libraries/go-observation/opentracing"
	"cd.splunkdev.com/libraries/go-observation/opentracing/testutil"
)

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
	env := testutil.StashEnv()
	defer testutil.PopEnv(env)
	SetupTestEnvironmentVar()
	g := testutil.GetGlobalTracer()
	defer testutil.RestoreGlobalTracer(g)

	serviceName := "test new tracer"
	outC, w := testutil.StartLogCapturing()
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
	testutil.StopLogCapturing(outC, w)

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
