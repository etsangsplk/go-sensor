package opentracing

import (
	"context"
	"net/http"
	"testing"

	opentracing "github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/opentracing/opentracing-go/mocktracer"
	"github.com/stretchr/testify/assert"
)

func TestNewTransportWithSpanTagHandlers(t *testing.T) {
	// Initiate a mock tracer and a top level span
	g := SaveGlobalTracer()
	defer RestoreGlobalTracer(g)
	tracer := mocktracer.New()
	span := tracer.StartSpan("topspan").(*mocktracer.MockSpan)
	topSpanContext := opentracing.ContextWithSpan(context.Background(), span)

	h := NewHandlerList()
	transport := NewTransportWithSpanTagHandlers(h)
	httpClient := &http.Client{
		Transport: transport,
	}
	req, err := NewRequest(topSpanContext, "GET", "http://test.biz/tenant1/foo?spantype=old", nil)
	httpClient.Do(req)

	tags := span.Tags()

	assert.False(t, h.IsFull(), false, "default capacity as 5")
	assert.False(t, h.IsEmpty(), false, "there should be 1 default handler to inject default set of tags")
	assert.Equal(t, h.size(), 1, "1 default handler")
	assert.NoError(t, err)
	assert.NotNil(t, tags)
	assert.NotEmpty(t, tags)
	assert.Equal(t, http.MethodGet, tags[string(ext.HTTPMethod)], "span tags from default handler")

	// nil handler will not panic and reduce to default handler case.
	h = nil
	transport = NewTransportWithSpanTagHandlers(h)
	httpClient = &http.Client{
		Transport: transport,
	}
	req, err = NewRequest(topSpanContext, "GET", "http://test.biz/tenant1/foo?spantype=old", nil)
	httpClient.Do(req)

	assert.NoError(t, err)
	assert.NotNil(t, tags)
	assert.NotEmpty(t, tags)
	assert.Equal(t, http.MethodGet, tags[string(ext.HTTPMethod)], "span tags from default handler")
}
