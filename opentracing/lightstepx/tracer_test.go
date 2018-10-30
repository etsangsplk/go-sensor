package lightstepx

import (
	"context"
	"fmt"
	"os"
	"testing"

	lightstep "github.com/lightstep/lightstep-tracer-go"
	cpb "github.com/lightstep/lightstep-tracer-go/collectorpb"
	cpbfakes "github.com/lightstep/lightstep-tracer-go/collectorpb/collectorpbfakes"
	opentracing "github.com/opentracing/opentracing-go"
	"github.com/stretchr/testify/assert"

	"cd.splunkdev.com/libraries/go-observation/logging"
	ot "cd.splunkdev.com/libraries/go-observation/opentracing"
	"cd.splunkdev.com/libraries/go-observation/opentracing/testutil"
)

func SetupTestEnvironmentVar() {
	os.Setenv(string(EnvCollectorEndpointHostPort), "localhost:32776")
	os.Setenv(string(EnvCollectorEndpointSendPlainText), "false")
	os.Setenv(string(EnvLightStepAPIHostPort), "127.0.0.5:8083")
	os.Setenv(string(EnvLightStepAPISendPlainText), "false")
	os.Setenv(string(EnvLightStepAccessToken), "test")
}

func MockLightStepTracer(serviceName string, logger *logging.Logger) opentracing.Tracer {
	fakeClient := new(cpbfakes.FakeCollectorServiceClient)
	fakeClient.ReportReturns(&cpb.ReportResponse{}, nil)
	fakeConn := fakeConnection(fakeClient)
	return NewTracerWithOptions(serviceName,
		logger,
		WithAccessToken("fakeAccessToken"),
		withConnection(fakeConn),
		WithLogRecorder(logger),
	)
}

func TestNewTracerWithTraceLogger(t *testing.T) {
	env := testutil.StashEnv()
	defer testutil.PopEnv(env)
	SetupTestEnvironmentVar()
	serviceName := "test new tracer"

	outC, w := testutil.StartLogCapturing()
	logger := logging.NewWithOutput(serviceName, w)
	logging.SetGlobalLogger(logger)

	g := testutil.GetGlobalTracer()
	defer testutil.RestoreGlobalTracer(g)
	tracer := MockLightStepTracer(serviceName, logger)
	ot.SetGlobalTracer(tracer)

	span := tracer.StartSpan("testNewSpan")
	span.SetOperationName("TestNewTracerWithTraceLogger")
	span.LogKV("event", "test NewTracerWithTraceLogger", "type", "unit test", "waited.millis", 1500)
	// Must finish span before validation.
	span.Finish()

	Flush(context.Background()) // Flush the tracer before validation.
	s := testutil.StopLogCapturing(outC, w)

	assert.NotNil(t, tracer)
	// Check that some signs of reporter being initialized and that
	// a span ie being reported. SpanID is changing so just do inference.
	hostname, _ := os.Hostname()
	assert.Contains(t, s[0], fmt.Sprintf(`"hostname":"%s"`, hostname))
	assert.Contains(t, s[0], `"service":"test new tracer"`)
	assert.Contains(t, s[0], `"operation":"TestNewTracerWithTraceLogger"`)
	assert.Contains(t, s[0], `"parentSpanID":0`) // This is a new Span so no parent
	assert.NotContains(t, s[0], `"spanID":""`)
	assert.NotContains(t, s[0], `"traceID":""`)
}

// withConnection intercepts the outgoing connection from lightstep client.
// This funciton is only used in testing.
func withConnection(fakeConnection lightstep.ConnectorFactory) Option {
	return func(c *Options) {
		if c != nil {
			c.ConnFactory = fakeConnection
		}
	}
}

// The following is mocking outgoing connection form lightstep client. Strictly for testing.
func getReportedSpans(fakeClient *cpbfakes.FakeCollectorServiceClient) []*cpb.Span {
	callCount := fakeClient.ReportCallCount()
	spans := make([]*cpb.Span, 0)
	for i := 0; i < callCount; i++ {
		_, report, _ := fakeClient.ReportArgsForCall(i)
		spans = append(spans, report.GetSpans()...)
	}
	return spans
}

type dummyConnection struct{}

func (*dummyConnection) Close() error { return nil }

func fakeConnection(fakeClient *cpbfakes.FakeCollectorServiceClient) lightstep.ConnectorFactory {
	return func() (interface{}, lightstep.Connection, error) {
		return fakeClient, new(dummyConnection), nil
	}
}
