package opencensus

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	opentracing "github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/opentracing/opentracing-go/mocktracer"
	"github.com/stretchr/testify/assert"

	"cd.splunkdev.com/libraries/go-observation/logging"
	"cd.splunkdev.com/libraries/go-observation/opentracing/testutil"
	"cd.splunkdev.com/libraries/go-observation/tracing"
)

// Test that when we make an outbound http GET request within a span at client side
// all the meta data about the downstream service will also be
// reflected in the span at client side.
func TestNewRequestGET(t *testing.T) {
	// Initiate a mock tracer and a top level span
	tracer := mocktracer.New()
	span := tracer.StartSpan("to_inject").(*mocktracer.MockSpan)
	defer span.Finish()

	// This top span also has some extra meta data as "baggage" to propagate to
	// any potential children span.
	span.SetBaggageItem("key1", "value1")
	span.SetBaggageItem("key2", "value2")

	// Wrap Span and context
	topSpanContext := opentracing.ContextWithSpan(context.Background(), span)

	// Send GET request
	httpClient := &http.Client{}
	req, err1 := NewRequest(topSpanContext, "GET", "http://test.biz/tenant1/foo?param1=value1", nil)
	httpClient.Do(req)

	assert.NoError(t, err1)
	assert.NotNil(t, span.Tags())
	assert.NotEmpty(t, span.Tags())

	// Check that the current span annotated with the request info.
	tags := span.Tags()
	assert.Equal(t, http.MethodGet, tags[string(ext.HTTPMethod)])
	assert.Contains(t, tags[string(ext.HTTPUrl)], "tenant1/foo")
	assert.Equal(t, "test.biz", tags[string(ext.PeerHostname)])

	spanCtx := fmt.Sprintf("%#v", span.Context())
	// MockTracer is not Jaeger Tracer so the key is different
	// So this is important info for Splunk app.
	assert.NotContains(t, spanCtx, `SpanID:""`)
	assert.Contains(t, spanCtx, `SpanID:`)
	assert.NotContains(t, spanCtx, `TraceID:""`)
	assert.Contains(t, spanCtx, `TraceID:`)
	assert.Contains(t, spanCtx, `"key1":"value1"`)
	assert.Contains(t, spanCtx, `"key2":"value2"`)
	assert.Contains(t, spanCtx, `"key2":"value2"`)
}

// Test that when we make an outbound http POST request within a span at client side
// all the meta data about the downstream service will also be
// reflected in the span at client side.
func TestNewRequestPOST(t *testing.T) {
	// Initiate a mock tracer and a top level span
	tracer := mocktracer.New()
	span := tracer.StartSpan("to_inject").(*mocktracer.MockSpan)
	defer span.Finish()

	// This top span also has some extra meta data as "baggage" to propagate to
	// any potential children span.
	span.SetBaggageItem("key1", "value1")
	span.SetBaggageItem("key2", "value2")

	// Wrap Span and context
	topSpanContext := opentracing.ContextWithSpan(context.Background(), span)

	// Send POST request
	httpClient := &http.Client{}
	var jsonStr = []byte(`{"title":"Buy cheese and bread for breakfast."}`)
	req, err2 := NewRequest(topSpanContext, "POST", "http://google.com", bytes.NewBuffer(jsonStr))
	req.Header.Set("Content-Type", "application/json")
	// This propagate X-Request-ID to another microservice
	httpClient.Do(req)

	assert.NoError(t, err2)
	assert.NotNil(t, span.Tags())
	assert.NotEmpty(t, span.Tags())

	// Check that the current span annotated with the request info.
	tags := span.Tags()
	assert.Equal(t, http.MethodPost, tags[string(ext.HTTPMethod)])
	assert.Equal(t, "google.com", tags[string(ext.PeerHostname)])

	spanCtx := fmt.Sprintf("%#v", span.Context())
	// MockTracer is not Jaeger Tracer so the key is different
	// So this is important info for Splunk app.
	assert.NotContains(t, spanCtx, `SpanID:""`)
	assert.Contains(t, spanCtx, `SpanID:`)
	assert.NotContains(t, spanCtx, `TraceID:""`)
	assert.Contains(t, spanCtx, `TraceID:`)
	assert.Contains(t, spanCtx, `"key1":"value1"`)
	assert.Contains(t, spanCtx, `"key2":"value2"`)
	assert.Contains(t, spanCtx, `"key2":"value2"`)
}

func TestHttpOpentracingHandler(t *testing.T) {
	var e error
	// Initiate a mock tracer and a top level span
	g := testutil.GetGlobalTracer()
	defer testutil.RestoreGlobalTracer(g)

	tracer := mocktracer.New()
	topSpan := tracer.StartSpan("TestHttpOpentracing").(*mocktracer.MockSpan)
	opentracing.SetGlobalTracer(tracer)
	var h http.Handler
	h = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		_, e = w.Write([]byte(`"Success"`))
		if e != nil {
			t.Fatal(e)
		}
	})

	outC, logWriter := testutil.StartLogCapturing()
	logger := logging.NewWithOutput("TestHttpOpentracing", logWriter)

	h = tracing.NewRequestContextHandler(
		logging.NewRequestLoggerHandler(logger,
			NewHTTPOpenTracingHandler((h))))

	// Prepare request with span context

	spanContext := opentracing.ContextWithSpan(context.Background(), topSpan)

	r := httptest.NewRequest("GET", "/tenant1/operationha", nil)
	r.Header.Add(tracing.XRequestID, "abcde")
	r = InjectHTTPRequestWithSpan(r.WithContext(spanContext))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)

	// Release/Flush resources
	topSpan.Finish()

	spans := tracer.FinishedSpans()
	testutil.StopLogCapturing(outC, logWriter)

	assert.Equal(t, 2, len(spans), "should have 2 span records. TopSpan and a child span")
	assert.Equal(t, spans[0].ParentID, spans[1].SpanContext.SpanID, "child span should have top span as parent id")
}

// NewRequest returns a new request from upstream ctx context.
func NewRequest(ctx context.Context, method string, url string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}
	// This propagate X-Request-ID to another microservice
	req.Header.Add(tracing.XRequestID, "abcde")
	req = InjectHTTPRequestWithSpan(req.WithContext(ctx))
	return req, err
}
