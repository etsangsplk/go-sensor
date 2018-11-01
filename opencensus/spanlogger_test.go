package opentracing

import (
	"errors"
	"testing"

	"github.com/opentracing/opentracing-go/mocktracer"
	"github.com/stretchr/testify/assert"

	"cd.splunkdev.com/libraries/go-observation/logging"
	"cd.splunkdev.com/libraries/go-observation/opentracing/testutil"
)

func TestSpanLoggerInfo(t *testing.T) {
	env := testutil.StashEnv()
	defer testutil.PopEnv(env)
	SetupTestEnvironmentVar()

	serviceName := "test span logger at info level"
	outC, w := testutil.StartLogCapturing()
	logger := logging.NewWithOutput(serviceName, w)
	logging.SetGlobalLogger(logger)

	g := testutil.GetGlobalTracer()
	defer testutil.RestoreGlobalTracer(g)
	tracer := mocktracer.New()
	SetGlobalTracer(tracer)

	span := tracer.StartSpan("a span")
	spanLogger := NewSpanLoggerWithSpan(logger, span)

	spanLogger.Info("message 1")
	spanLogger.Info("message 2", "keyWithNoValue")
	spanLogger.Info("message 3", "sql statement", "SELECT tenantinfo, subscription from tenant where tenantID=1234")

	span.Finish()

	// Flush tracer before validation.
	// If tracer is not a lightStep tracer, flush is noop.
	spans := tracer.FinishedSpans()

	testutil.StopLogCapturing(outC, w)

	// We 1 span with 2 span logs. span log with dangling  key is rejected.
	assert.Equal(t, 1, len(spans), "should have 1 span records")
	logRecords := spans[0].Logs()
	assert.Equal(t, 2, len(logRecords), "should have 2 span log records")

	assert.Contains(t, logRecords[0].Fields[0].Key, "event")
	assert.Contains(t, logRecords[0].Fields[0].ValueString, "message 1")
	assert.Contains(t, logRecords[0].Fields[1].Key, "level")
	assert.Contains(t, logRecords[0].Fields[1].ValueString, "info")

	assert.Contains(t, logRecords[1].Fields[0].Key, "event")
	assert.Contains(t, logRecords[1].Fields[0].ValueString, "message 3")
	assert.Contains(t, logRecords[1].Fields[1].Key, "level")
	assert.Contains(t, logRecords[1].Fields[1].ValueString, "info")

	assert.Contains(t, logRecords[1].Fields[2].Key, "sql statement")
	assert.Contains(t, logRecords[1].Fields[2].ValueString, "SELECT tenantinfo, subscription from tenant where tenantID=1234")
}

func TestSpanLoggerError(t *testing.T) {
	env := testutil.StashEnv()
	defer testutil.PopEnv(env)
	SetupTestEnvironmentVar()

	serviceName := "test span logger at error level"
	outC, w := testutil.StartLogCapturing()
	logger := logging.NewWithOutput(serviceName, w)
	logging.SetGlobalLogger(logger)
	spanLogger := NewSpanLoggerWithSpan(logger, nil)

	g := testutil.GetGlobalTracer()
	defer testutil.RestoreGlobalTracer(g)
	tracer := mocktracer.New()

	SetGlobalTracer(tracer)

	span := tracer.StartSpan("a span")
	spanLogger = spanLogger.WithSpan(span)

	spanLogger.Error(errors.New("no fields"), "somethig went wrong")
	spanLogger.Error(errors.New("key with no value"), "somethig went wrong", "keyWithNoValue")
	spanLogger.Error(errors.New("crash error"), "somethig went wrong again", "tenantID", "123")

	span.Finish()
	// Flush tracer before validation.
	// If tracer is not a lightStep tracer, flush is noop.
	spans := tracer.FinishedSpans()

	testutil.StopLogCapturing(outC, w)

	assert.Equal(t, 1, len(spans), "should have 1 span records")
	logRecords := spans[0].Logs()
	assert.Equal(t, 2, len(logRecords), "should have 2 span log records")

	assert.Contains(t, logRecords[0].Fields[0].Key, "error")
	assert.Contains(t, logRecords[0].Fields[0].ValueString, "no fields")
	assert.Contains(t, logRecords[0].Fields[1].Key, "event")
	assert.Contains(t, logRecords[0].Fields[1].ValueString, "somethig went wrong")
	assert.Contains(t, logRecords[0].Fields[2].Key, "level")
	assert.Contains(t, logRecords[0].Fields[2].ValueString, "error")

	assert.Contains(t, logRecords[1].Fields[0].Key, "error")
	assert.Contains(t, logRecords[1].Fields[0].ValueString, "crash error")
	assert.Contains(t, logRecords[1].Fields[1].Key, "event")
	assert.Contains(t, logRecords[1].Fields[1].ValueString, "somethig went wrong again")
	assert.Contains(t, logRecords[0].Fields[2].Key, "level")
	assert.Contains(t, logRecords[0].Fields[2].ValueString, "error")
	assert.Contains(t, logRecords[1].Fields[3].Key, "tenantID")
	assert.Contains(t, logRecords[1].Fields[3].ValueString, "123")
}
