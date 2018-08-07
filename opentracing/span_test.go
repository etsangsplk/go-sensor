package opentracing

import (
	"context"
	"errors"
	"os"
	"strconv"
	"testing"

	instanaSensor "github.com/instana/golang-sensor"
	lightstep "github.com/lightstep/lightstep-tracer-go"
	cpb "github.com/lightstep/lightstep-tracer-go/collectorpb"
	cpbfakes "github.com/lightstep/lightstep-tracer-go/collectorpb/collectorpbfakes"
	opentracing "github.com/opentracing/opentracing-go"
	"github.com/stretchr/testify/assert"

	"cd.splunkdev.com/libraries/go-observation/logging"
	"cd.splunkdev.com/libraries/go-observation/opentracing/instanax"
	"cd.splunkdev.com/libraries/go-observation/opentracing/lightstepx"
)

// Need to enable either Lighstep or Instana
func TestChildSpanCreation(t *testing.T) {
	if lightstepx.Enabled() {
		env := StashEnv()
		defer PopEnv(env)
		SetupTestEnvironmentVar()

		serviceName := "test multiple span creation"
		outC, w := StartLogCapturing()
		logger := logging.NewWithOutput(serviceName, w)
		logging.SetGlobalLogger(logger)

		// Important, if you are using a MockTracer, childSpan will return as noopSpan
		// internally the tracer looks for Global() tracer for creation after finding a parent
		// exists from context. Make sure all mockTracer is not set as Global()
		g := SaveGlobalTracer()
		defer RestoreGlobalTracer(g)

		recorder := instanaSensor.NewTestRecorder()
		tracer := MockInstanaTracer(serviceName, recorder)
		SetGlobalTracer(tracer)

		ctx := context.Background()
		// OMG this is so omg .. why I have to pass in static meta like hostname on each span.
		parentSpan := tracer.StartSpan("parent span", opentracing.Tags{logging.HostnameKey: hostname})
		parentSpan.LogKV("event", "parent span created")

		// create a child span from parent span context
		parentCtx := ContextWithSpan(ctx, parentSpan)
		childSpan, _ := StartSpanFromContext(parentCtx, "child span")
		childSpan.LogKV("event", "child span created")
		// Must finish span before validation.
		parentSpan.Finish()
		childSpan.Finish()

		spans := recorder.GetQueuedSpans()
		StopLogCapturing(outC, w)

		assert.NotNil(t, tracer)

		assert.Equal(t, 2, len(spans))

		// Validate parent span.
		assert.NotZero(t, spans[0].SpanID, "Missing span ID for this span")
		assert.NotZero(t, spans[0].TraceID, "Missing trace ID for this span")
		assert.Zero(t, spans[0].ParentID, "Parent Span should not have parent ID")
		assert.NotNil(t, spans[0].Duration, "Duration is nil")
		assert.Equal(t, serviceName, spans[0].Data.Service, "Missing span service name")
		assert.Equal(t, "parent span", spans[0].Data.SDK.Name, "Missing span operation name")
		assert.Contains(t, spans[0].Data.SDK.Custom.Tags[logging.HostnameKey], hostname, "Missing hostname")

		// Validate child span.
		assert.NotZero(t, spans[1].SpanID, "Missing span ID for this span")
		assert.NotZero(t, spans[1].TraceID, "Missing trace ID for this span")
		assert.NotZero(t, spans[1].ParentID, "Child Span should have parent ID")
		assert.NotNil(t, spans[1].Duration, "Duration is nil")
		assert.Equal(t, serviceName, spans[1].Data.Service, "Missing span service name")
		assert.Equal(t, "child span", spans[1].Data.SDK.Name, "Missing span operation name")
		// Since instana does not create tracer with tags, and opentracinig mandate tags are local to each span and no inheritance,
		// we have no tags. hostname is alwayws added.
		// assert.Equal(t, 1, len(spans[1].Data.SDK.Custom.Tags), "tags are local to a span and no inheritance is allowed")
	}
}

func TestInstanaChildSpanCreation(t *testing.T) {
	serviceName := "test multiple span creation"
	outC, w := StartLogCapturing()
	logger := logging.NewWithOutput(serviceName, w)
	logging.SetGlobalLogger(logger)
	if instanax.Enabled() {
		env := StashEnv()
		defer PopEnv(env)
		SetupTestEnvironmentVar()
		// Important, if you are using a MockTracer, childSpan will return as noopSpan
		// internally the tracer looks for Global() tracer for creation after finding a parent
		// exists from context. Make sure all mockTracer is not set as Global()
		g := SaveGlobalTracer()
		defer RestoreGlobalTracer(g)

		recorder := instanaSensor.NewTestRecorder()
		tracer := MockInstanaTracer(serviceName, recorder)
		SetGlobalTracer(tracer)

		ctx := context.Background()
		// OMG this is so omg .. why I have to pass in static meta like hostname on each span.
		parentSpan := tracer.StartSpan("parent span", opentracing.Tags{logging.HostnameKey: hostname})
		parentSpan.LogKV("event", "parent span created")

		// create a child span from parent span context
		parentCtx := ContextWithSpan(ctx, parentSpan)
		childSpan, _ := StartSpanFromContext(parentCtx, "child span")
		childSpan.LogKV("event", "child span created")
		// Must finish span before validation.
		parentSpan.Finish()
		childSpan.Finish()

		spans := recorder.GetQueuedSpans()
		StopLogCapturing(outC, w)

		assert.NotNil(t, tracer)

		assert.Equal(t, 2, len(spans))

		// Validate parent span.
		assert.NotZero(t, spans[0].SpanID, "Missing span ID for this span")
		assert.NotZero(t, spans[0].TraceID, "Missing trace ID for this span")
		assert.Zero(t, spans[0].ParentID, "Parent Span should not have parent ID")
		assert.NotNil(t, spans[0].Duration, "Duration is nil")
		assert.Equal(t, serviceName, spans[0].Data.Service, "Missing span service name")
		assert.Equal(t, "parent span", spans[0].Data.SDK.Name, "Missing span operation name")
		assert.Contains(t, spans[0].Data.SDK.Custom.Tags[logging.HostnameKey], hostname, "Missing hostname")

		// Validate child span.
		assert.NotZero(t, spans[1].SpanID, "Missing span ID for this span")
		assert.NotZero(t, spans[1].TraceID, "Missing trace ID for this span")
		assert.NotZero(t, spans[1].ParentID, "Child Span should have parent ID")
		assert.NotNil(t, spans[1].Duration, "Duration is nil")
		assert.Equal(t, serviceName, spans[1].Data.Service, "Missing span service name")
		assert.Equal(t, "child span", spans[1].Data.SDK.Name, "Missing span operation name")
		// Since instana does not create tracer with tags, and opentracinig mandate tags are local to each span and no inheritance,
		// we have no tags. hostname is alwayws added.
		// assert.Equal(t, 1, len(spans[1].Data.SDK.Custom.Tags), "tags are local to a span and no inheritance is allowed")
	}
}

func SetupTestEnvironmentVar() {
	logger := logging.Global()
	if lightstepx.Enabled() && instanax.Enabled() {
		logger.Fatal(errors.New("cannot enable both Lighstep and Instana"), "use either Lightstep or Instana")
	}

	if lightstepx.Enabled() {
		os.Setenv(string(lightstepx.EnvCollectorEndpointHostPort), "127.0.0.1:8080")
		os.Setenv(string(lightstepx.EnvCollectorEndpointSendPlainText), "true")
		os.Setenv(string(lightstepx.EnvLightStepAPIHostPort), "127.0.0.5:8083")
		os.Setenv(string(lightstepx.EnvLightStepAPISendPlainText), "false")
		os.Setenv(string(lightstepx.EnvLightStepAccessToken), "test")
	}
	if instanax.Enabled() {
		os.Setenv(string(instanax.EnvInstanaAgentHost), "127.0.0.1:")
		os.Setenv(string(instanax.EnvInstanaAgentPort), "8080")
	}
	logger.Warn("neither Lighstep nor Instana is enabled")
}

func MockInstanaTracer(serviceName string, recorder instanaSensor.SpanRecorder) opentracing.Tracer {
	opts := &instanax.Options{}
	agentHost, _ := os.LookupEnv(string(instanax.EnvInstanaAgentHost))
	agentPort, _ := os.LookupEnv(string(instanax.EnvInstanaAgentPort))
	port, _ := strconv.Atoi(agentPort)

	instanax.WithServiceName(serviceName)(opts)
	instanax.WithAgentEndpoint(agentHost, port)(opts)

	return instanaSensor.NewTracerWithEverything(&opts.Opts, recorder)
}

func MockLightStepTracer(serviceName string, logger *logging.Logger) opentracing.Tracer {
	fakeClient := new(cpbfakes.FakeCollectorServiceClient)
	fakeClient.ReportReturns(&cpb.ReportResponse{}, nil)
	fakeConn := fakeConnection(fakeClient)
	return lightstepx.NewTracerWithOptions(serviceName,
		logger,
		lightstepx.WithAccessToken("fakeAccessToken"),
		lightstepx.WithConnection(fakeConn),
		lightstepx.WithLogRecorder(logging.Global()),
	)
}

func flushTracer(ctx context.Context) {
	if lightstepx.Enabled() {
		lightstepx.Flush(ctx)
	}
	if instanax.Enabled() {
		instanax.Flush(ctx)
	}
}

// The following is mocking outgoing connection form Lightstep client. Strictly for testing.
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
