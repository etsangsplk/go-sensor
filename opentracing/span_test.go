package opentracing

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"

	lightstep "github.com/lightstep/lightstep-tracer-go"
	cpb "github.com/lightstep/lightstep-tracer-go/collectorpb"
	cpbfakes "github.com/lightstep/lightstep-tracer-go/collectorpb/collectorpbfakes"
	opentracing "github.com/opentracing/opentracing-go"
	"github.com/stretchr/testify/assert"

	"cd.splunkdev.com/libraries/go-observation/logging"
	"cd.splunkdev.com/libraries/go-observation/opentracing/lightstepx"
)

func TestChildSpanCreation(t *testing.T) {
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
	tracer := MockLightStepTracer(serviceName, logger)
	SetGlobalTracer(tracer)

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

	// Flush tracer before validation.
	// If tracer is not a lightStep tracer, flush is noop.
	lightstepx.Flush(context.Background())
	s := StopLogCapturing(outC, w)

	assert.NotNil(t, tracer)
	// Validate parent span.
	assert.Contains(t, s[0], `"service":"test multiple span creation`)
	assert.Contains(t, s[0], `"operation":"parent span"`)
	assert.Contains(t, s[0], `"parentSpanID":0`) // This is a new Span so no parent
	assert.NotContains(t, s[0], `"SpanID":""`)
	assert.NotContains(t, s[0], `"TraceID":""`)

	// Validate child span.
	assert.Contains(t, s[1], `"service":"test multiple span creation`)
	assert.Contains(t, s[1], `"operation":"child span"`)
	assert.NotContains(t, s[1], `"parentSpanID":0`) // This is a child Span so has parent
	assert.NotContains(t, s[1], `"spanID":""`)
	assert.NotContains(t, s[1], `"traceID":""`)
}

func SetupTestEnvironmentVar() {
	os.Setenv(string(lightstepx.EnvCollectorEndpointHostPort), "127.0.0.1:8080")
	os.Setenv(string(lightstepx.EnvCollectorEndpointSendPlainText), "true")
	os.Setenv(string(lightstepx.EnvLightStepAPIHostPort), "127.0.0.5:8083")
	os.Setenv(string(lightstepx.EnvLightStepAPISendPlainText), "false")
	os.Setenv(string(lightstepx.EnvLightStepAccessToken), "test")
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

// StashEnv stashes the current environment variables and returns an array of
// all environment values as key=val strings.
func StashEnv() []string {
	// Rip https://github.com/aws/aws-sdk-go/awstesting/util.go
	env := os.Environ()
	os.Clearenv()
	return env
}

// PopEnv takes the list of the environment values and injects them into the
// process's environment variable data. Clears any existing environment values
// that may already exist.
func PopEnv(env []string) {
	// Rip https://github.com/aws/aws-sdk-go/awstesting/util.go
	os.Clearenv()
	for _, e := range env {
		p := strings.SplitN(e, "=", 2)
		k, v := p[0], ""
		if len(p) > 1 {
			v = p[1]
		}
		os.Setenv(k, v)
	}
}

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
