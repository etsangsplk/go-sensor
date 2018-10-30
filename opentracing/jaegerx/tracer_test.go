package jaegerx

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"testing"

	opentracing "github.com/opentracing/opentracing-go"
	opentracingLog "github.com/opentracing/opentracing-go/log"
	"github.com/stretchr/testify/assert"
	jaeger "github.com/uber/jaeger-client-go"
	"github.com/uber/jaeger-lib/metrics"

	"cd.splunkdev.com/libraries/go-observation/logging"
	ot "cd.splunkdev.com/libraries/go-observation/opentracing"

	"cd.splunkdev.com/libraries/go-observation/opentracing/testutil"
)

func SetupTestEnvironmentVar() {
	os.Setenv(string(EnvJaegerAgentHost), "127.0.0.1")
	os.Setenv(string(EnvJaegerAgentPort), "8080")
}

func MockJaegerTracer(serviceName string) (opentracing.Tracer, io.Closer) {
	logger := logging.Global()
	log := newLogger(logger)
	metrics := jaeger.NewMetrics(metrics.NewLocalFactory(0), nil)
	observer := testSpanobserver{logger: logger}
	tracer, closer := jaeger.NewTracer(serviceName,
		jaeger.NewConstSampler(true),
		jaeger.NewLoggingReporter(log),
		jaeger.TracerOptions.ContribObserver(observer),
		jaeger.TracerOptions.Metrics(metrics),
	)
	return tracer, closer
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

	tracer, closer := MockJaegerTracer(serviceName)

	ot.SetGlobalTracer(tracer)
	hostname, _ := os.Hostname()
	span := tracer.StartSpan("testNewSpan", opentracing.Tags{logging.HostnameKey: hostname})
	span.SetOperationName("TestNewTracerWithTraceLogger")
	span.LogKV("event", "test NewTracerWithTraceLogger", "type", "unit test", "waited.millis", 1500)
	// Must finish span before validation.
	span.Finish()

	closer.Close()
	spans := testutil.StopLogCapturing(outC, w)

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

type testSpanobserver struct {
	jaeger.ContribSpanObserver
	logger *logging.Logger
}

func (o *testSpanobserver) OnStartSpan(sp opentracing.Span, operationName string, options opentracing.StartSpanOptions) (jaeger.ContribSpanObserver, bool) {
	o.logger.Info("span operation name", "operationname", operationName)
	for _, v := range options.Tags {
		t := fmt.Sprintf("#%v", v)
		o.logger.Info("span operation tags", "tags", t)
	}

	return o, true
}

func (o *testSpanobserver) OnSetOperationName(operationName string) {
	o.logger.Info("span operation name", "operationname", operationName)
}

func (o *testSpanobserver) OnSetTag(key string, value interface{}) {
	v := fmt.Sprintf("#%v", value)
	o.logger.Info("span tag", "tag", key, "value", v)
}

func (o *testSpanobserver) OnFinish(options opentracing.FinishOptions) {
	logRecords := options.LogRecords
	for _, v := range logRecords {
		data, err := MaterializeWithJSON(v.Fields)
		if err != nil {
			o.logger.Info("span log", "log record", string(data))
		} else {
			o.logger.Error(err, "error reading span log record")
		}
	}
}

type fieldsAsMap map[string]string

// MaterializeWithJSON converts log Fields into JSON string
// TODO refactor into pluggable materializer
func MaterializeWithJSON(logFields []opentracingLog.Field) ([]byte, error) {
	fields := fieldsAsMap(make(map[string]string, len(logFields)))
	for _, field := range logFields {
		field.Marshal(fields)
	}
	if event, ok := fields["event"]; ok && len(fields) == 1 {
		return []byte(event), nil
	}
	return json.Marshal(fields)
}

func (ml fieldsAsMap) EmitString(key, value string) {
	ml[key] = value
}

func (ml fieldsAsMap) EmitBool(key string, value bool) {
	ml[key] = fmt.Sprintf("%t", value)
}

func (ml fieldsAsMap) EmitInt(key string, value int) {
	ml[key] = fmt.Sprintf("%d", value)
}

func (ml fieldsAsMap) EmitInt32(key string, value int32) {
	ml[key] = fmt.Sprintf("%d", value)
}

func (ml fieldsAsMap) EmitInt64(key string, value int64) {
	ml[key] = fmt.Sprintf("%d", value)
}

func (ml fieldsAsMap) EmitUint32(key string, value uint32) {
	ml[key] = fmt.Sprintf("%d", value)
}

func (ml fieldsAsMap) EmitUint64(key string, value uint64) {
	ml[key] = fmt.Sprintf("%d", value)
}

func (ml fieldsAsMap) EmitFloat32(key string, value float32) {
	ml[key] = fmt.Sprintf("%f", value)
}

func (ml fieldsAsMap) EmitFloat64(key string, value float64) {
	ml[key] = fmt.Sprintf("%f", value)
}

func (ml fieldsAsMap) EmitObject(key string, value interface{}) {
	ml[key] = fmt.Sprintf("%+v", value)
}

func (ml fieldsAsMap) EmitLazyLogger(value opentracingLog.LazyLogger) {
	value(ml)
}
