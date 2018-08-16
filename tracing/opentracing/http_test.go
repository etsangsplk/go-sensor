package opentracing

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"testing"

	opentracing "github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/opentracing/opentracing-go/mocktracer"
	"github.com/stretchr/testify/assert"

	"github.com/splunk/ssc-observation/tracing"
)

// Test that when we make an outbound http GET request within a span at client side
// all the meta data about the downstream service will also be
// refected in the span at client side.
func TestNewRequestGET(t *testing.T) {
	// Initiate a mock tracer and a top level span
	tracer := mocktracer.New()
	SetGlobalTracer(tracer)
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
	req.Header.Add(tracing.XRequestID, "abcde")
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
	// MockTracer is not jaeger Tracer so the key is different
	// So this is important info for splunk app.
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
// refected in the span at client side.
func TestNewRequestPOST(t *testing.T) {
	// Initiate a mock tracer and a top level span
	tracer := mocktracer.New()
	SetGlobalTracer(tracer)
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
	req.Header.Add(tracing.XRequestID, "abcde")
	httpClient.Do(req)

	assert.NoError(t, err2)
	assert.NotNil(t, span.Tags())
	assert.NotEmpty(t, span.Tags())

	// Check that the current span annotated with the request info.
	tags := span.Tags()
	assert.Equal(t, http.MethodPost, tags[string(ext.HTTPMethod)])
	assert.Equal(t, "google.com", tags[string(ext.PeerHostname)])

	spanCtx := fmt.Sprintf("%#v", span.Context())
	// MockTracer is not jaeger Tracer so the key is different
	// So this is important info for splunk app.
	assert.NotContains(t, spanCtx, `SpanID:""`)
	assert.Contains(t, spanCtx, `SpanID:`)
	assert.NotContains(t, spanCtx, `TraceID:""`)
	assert.Contains(t, spanCtx, `TraceID:`)
	assert.Contains(t, spanCtx, `"key1":"value1"`)
	assert.Contains(t, spanCtx, `"key2":"value2"`)
	assert.Contains(t, spanCtx, `"key2":"value2"`)
}
