package opentracing

import (
	"context"
	//"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	opentracing "github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/opentracing/opentracing-go/mocktracer"
	"github.com/stretchr/testify/assert"
	//jaeger "github.com/uber/jaeger-client-go"
	jaegerLogger "github.com/uber/jaeger-client-go/log"
)

func TestOutboundHTTPRequest(t *testing.T) {
	// Test that as we are sending out a http request, the current span should
	// annotate that information into the current span.
	tracer := mocktracer.New()
	// Initialize the ctx with a Span to inject.
	span := tracer.StartSpan("to_inject").(*mocktracer.MockSpan)
	defer span.Finish()

	// Create a top span with some extra meta data as "baggage" (if you want to propagate to any child spans)
	span.SetBaggageItem("key1", "value1")
	span.SetBaggageItem("key2", "value2")
	req := httptest.NewRequest("GET", "http://test.biz/tenant1/foo?param1=value1", nil)

	// Wrap Span and context
	topSpanContext := opentracing.ContextWithSpan(context.Background(), span)
	httpClient := NewHTTPClient(topSpanContext, tracer)
	httpClient.Do(req)

	assert.NotNil(t, span.Tags())
	assert.NotEmpty(t, span.Tags())

	// Check that the current span annotated with the request info.
	tags := span.Tags()
	assert.Equal(t, http.MethodGet, tags[string(ext.HTTPMethod)])
	assert.Contains(t, tags[string(ext.HTTPUrl)], "tenant1/foo")
	assert.Equal(t, "test.biz", tags[string(ext.PeerHostname)])
}

func TestNewTracingHttpClientFrom(t *testing.T) {
	bufferLogger := &jaegerLogger.BytesBufferLogger{}
	defer bufferLogger.Flush()
	//reporter := jaeger.NewInMemoryReporter()

	tracer, closer := NewTracer("ExampleRequestEndToEndTracing", bufferLogger)
	// Make sure close is called for any resource clean up related to tracer
	defer closer.Close()

	// Create a top span with some extra meta data as "baggage" (if you want to propagate to any child spans)
	span, topSpanContext := opentracing.StartSpanFromContext(context.Background(), "TestNewTracingHttpClientFrom")
	span.SetBaggageItem("key1", "value1")
	span.SetBaggageItem("key2", "value2")
	//
	httpClient := NewHTTPClient(topSpanContext, tracer)
	respGet, errGet := httpClient.Get("http://google.com")
	// Normal http response resource  closure.
	defer func() {
		if respGet != nil {
			respGet.Body.Close()
		}
	}()
	// Need to log something to span
	span.LogKV("http", "get")
	// Try out each Http call
	respHead, errHead := httpClient.Head("http://google.com")
	defer func() {
		if respHead != nil {
			respHead.Body.Close()
		}
	}()
	span.LogKV("http", "head")
	respPostForm, errPostForm := httpClient.PostForm("http://google.com", url.Values{"a": []string{"b"}})
	defer func() {
		if respPostForm != nil {
			respPostForm.Body.Close()
		}
	}()
	span.LogKV("http", "postform")
	assert.NotNil(t, httpClient)
	assert.NoError(t, errGet)
	assert.NotNil(t, respGet)
	assert.NoError(t, errHead)
	assert.NotNil(t, respHead)
	assert.NoError(t, errPostForm)
	assert.NotNil(t, respPostForm)

	// clean up resources
	span.Finish()
	// reporter.Close()

	//fmt.Printf("spans SpansSubmited %#v\n", reporter.SpansSubmitted())
	// fmt.Printf("reported spans: %#v\n", reporter.GetSpans())
}
