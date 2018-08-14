package opentracing

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	opentracing "github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/opentracing/opentracing-go/mocktracer"
	"github.com/stretchr/testify/assert"
)

// Test that when we make an outbound http request within a span at client side
// all the meta data about the downstream service will also be
// refected in the span at client side.
func TestOutboundHTTPRequest(t *testing.T) {
	// Initiate a mock tracer and a top level span
	tracer := mocktracer.New()
	SetGlobalTracer(tracer)
	span := tracer.StartSpan("to_inject").(*mocktracer.MockSpan)
	defer span.Finish()

	// This top span also has some extra meta data as "baggage" to propagate to
	// any potential children span.
	span.SetBaggageItem("key1", "value1")
	span.SetBaggageItem("key2", "value2")

	// Client send a request out to some mock server.
	req := httptest.NewRequest("GET", "http://test.biz/tenant1/foo?param1=value1", nil)

	// Wrap Span and context
	topSpanContext := opentracing.ContextWithSpan(context.Background(), span)
	httpClient := NewHTTPClient(topSpanContext)
	httpClient.Do(req)

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

func TestNewTracingHttpClientFrom(t *testing.T) {
	// Initiate a mock tracer and a top level span
	tracer := mocktracer.New()
	SetGlobalTracer(tracer)
	span := tracer.StartSpan("to_inject").(*mocktracer.MockSpan)
	defer span.Finish()

	// Wrap Span and context
	topSpanContext := opentracing.ContextWithSpan(context.Background(), span)

	// Test each http api in turn
	httpClient := NewHTTPClient(topSpanContext)
	respGet, errGet := httpClient.Get("http://google.com")
	// Test HEAD
	respHead, errHead := httpClient.Head("http://google.com")
	// Test POSTFORM, which also test POST
	respPostForm, errPostForm := httpClient.PostForm("http://google.com", url.Values{"a": []string{"b"}})

	assert.NotNil(t, httpClient)
	assert.NoError(t, errGet)
	assert.NotNil(t, respGet)
	assert.NoError(t, errHead)
	assert.NotNil(t, respHead)
	assert.NoError(t, errPostForm)
	assert.NotNil(t, respPostForm)
	// clean up resources
	span.Finish()

	spanCtx := fmt.Sprintf("%#v", span.Context())
	// MockTracer is not jaeger Tracer so the key is different
	// So this is important info for splunk app.
	assert.NotContains(t, spanCtx, `SpanID:""`)
	assert.Contains(t, spanCtx, `SpanID:`)
	assert.NotContains(t, spanCtx, `TraceID:""`)
	assert.Contains(t, spanCtx, `TraceID:`)
}
