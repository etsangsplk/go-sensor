package opentracing

import (
	"context"
	"net/http"
	"testing"

	opentracing "github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/opentracing/opentracing-go/mocktracer"
	"github.com/stretchr/testify/assert"

	"cd.splunkdev.com/libraries/go-observation/opentracing/testutil"
)

func TestNewTransport(t *testing.T) {
	// Initiate a mock tracer and a top level span
	g := testutil.GetGlobalTracer()
	defer testutil.RestoreGlobalTracer(g)
	tracer := mocktracer.New()
	span := tracer.StartSpan("topspan").(*mocktracer.MockSpan)
	topSpanContext := opentracing.ContextWithSpan(context.Background(), span)
	transport := NewTransport()
	httpClient := &http.Client{
		Transport: transport,
	}

	req, err := NewRequest(topSpanContext, "GET", "http://test.biz/tenant1/foo?spantype=old", nil)
	httpClient.Do(req)

	tags := span.Tags()

	span.Finish()
	tracer.FinishedSpans()

	assert.NoError(t, err)
	assert.NotNil(t, tags)
	assert.NotEmpty(t, tags)
	assert.Equal(t, http.MethodGet, tags[string(ext.HTTPMethod)])
	assert.Contains(t, tags[string(ext.HTTPUrl)], "tenant1/foo")
	assert.Contains(t, tags[string(ext.PeerHostname)], "test.biz")
}

// Test that we can register custom roundtripper to a standard http client
// and annotate all the meta data about the downstream service will also be
// reflected in the span at client side.
func TestRoundtripper(t *testing.T) {
	// Initiate a mock tracer and a top level span
	g := testutil.GetGlobalTracer()
	defer testutil.RestoreGlobalTracer(g)
	tracer := mocktracer.New()
	SetGlobalTracer(tracer)
	span := tracer.StartSpan("topspan").(*mocktracer.MockSpan)
	topSpanContext := opentracing.ContextWithSpan(context.Background(), span)

	spanTagHandlers := NewHandlerList()
	spanTagHandlers.PushBackNamed(NamedHandler{
		Name: "TestHander",
		Func: func(req *http.Request, span opentracing.Span) {
			span.SetTag("testname", t.Name())
		},
	})

	transport := NewTransportWithSpanTagHandlers(spanTagHandlers)
	httpClient := &http.Client{
		Transport: transport,
	}

	req, err := NewRequest(topSpanContext, "GET", "http://test.biz/tenant1/foo?spantype=old", nil)
	httpClient.Do(req)

	tags := span.Tags()
	assert.NoError(t, err)
	assert.NotNil(t, tags)
	assert.NotEmpty(t, tags)
	assert.Equal(t, http.MethodGet, tags[string(ext.HTTPMethod)])
	assert.Contains(t, tags[string(ext.HTTPUrl)], "tenant1/foo")
	assert.Equal(t, spanTagHandlers.size(), 2, "default and user defined handler")
	assert.Contains(t, tags[string(ext.PeerHostname)], "test.biz")

	httpClient.Get("http://test.biz/tenant2/foo?spantype=new")

	// We have to cleanse/close all the spans and tracer resource, or behavior is not defined.
	span.Finish()
	//Flush(context.Background())
	spans := tracer.FinishedSpans()
	// We should get 3 span *records*, 1 top span, 1 span annotated the first span,
	// 1 new Span (1 request context is not derived from any span).
	// Span records are not Span.
	if len(spans) != 7 {
		t.Fail()
	}
}
