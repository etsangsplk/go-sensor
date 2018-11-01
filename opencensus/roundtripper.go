package opencensus

import (
	"io"
	"net/http"

	"cd.splunkdev.com/libraries/go-observation/tracing"
	"github.com/opentracing/opentracing-go"
	tag "github.com/opentracing/opentracing-go/ext"
)

// This is for use with routing framework like swagger where http client
// is already wrapped deep inside the framework.
// Transport wraps a RoundTripper. If a request is being traced with
// Tracer, Transport will inject the current span into the headers,
// and set HTTP related tags on the span.
type Transport struct {
	// A list of callbacks that allow user to add additional span tags.
	handlers *HandlerList
	// The actual RoundTripper to use for the request. A nil
	// RoundTripper defaults to http.DefaultTransport.
	Base http.RoundTripper
}

// NewTansport create a new transport with default http.RoundTripper.
func NewTransport() *Transport {
	h := NewHandlerList()
	return NewTransportWithSpanTagHandlers(h)
}

// NewTransportWithSpanTagHandlers create a new transport with custom handler.
func NewTransportWithSpanTagHandlers(handlers *HandlerList) *Transport {
	return NewTransportWithRoundTripper(handlers, http.DefaultTransport)
}

// NewTransportWithRoundTripper create a new transport with custom roundTripper.
func NewTransportWithRoundTripper(handlers *HandlerList, roundTripper http.RoundTripper) *Transport {
	return &Transport{
		handlers: handlers,
		Base:     roundTripper,
	}
}

// RoundTrip implements the RoundTripper interface.
func (t *Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	rt := t.base()
	ctx := req.Context()

	var span opentracing.Span
	operationName := tracing.OperationIDFrom(ctx)
	span, ctx = opentracing.StartSpanFromContext(ctx, operationName)
	defer span.Finish()

	if component := tracing.ComponentFrom(ctx); len(component) > 0 {
		span.SetTag("component", component)
		// TODO: does lightstep not support the standard "component" tag key?
		span.SetTag("lightstep.component_name", component)
	}

	// X-REQUEST-ID is the request ID header
	requestID := req.Header.Get(tracing.XRequestID)
	span.SetTag(tracing.RequestIDKey, requestID)

	// There is a span retrieved from request context.
	// We are going to use this span in a client request, so mark as such.
	tag.SpanKindRPCClient.Set(span)

	// execute handlers to tag the span.
	if t.handlers != nil && !t.handlers.IsEmpty() {
		t.handlers.Run(req, span)
	} else {
		tagHTTPClientRequest(span, req)
	}

	// Inject the Span context into the outgoing HTTP Request.
	// Sicne we are sending an HTTP request, will use the HTTP headers as carrier.
	tracer := Global()
	err := tracer.Inject(
		span.Context(),
		opentracing.HTTPHeaders,
		opentracing.HTTPHeadersCarrier(req.Header),
	)
	if err != nil {
		// Indicate span resulted in failed operation.
		// We are just marking the Span as failed. The real request will still continue.
		tag.Error.Set(span, true)
	}
	// Now inject the span into request context for downstream propagation.
	spanCtx := opentracing.ContextWithSpan(ctx, span)
	req = req.WithContext(spanCtx)

	// Send the request out.
	resp, err := rt.RoundTrip(req)

	// Close the span if for any error.
	// We are not tagging span a failed because error can be from http side.
	if err != nil {
		span.Finish()
		return resp, err
	}
	// Tag the span with response.
	tag.HTTPStatusCode.Set(span, uint16(resp.StatusCode))
	if req.Method == http.MethodHead {
		span.Finish()
	} else {
		resp.Body = closeTracker{resp.Body, span}
	}
	return resp, nil
}

// base returns any custom transport or returns http.DefaultTransport as default.
func (t *Transport) base() http.RoundTripper {
	if t == nil || t.Base == nil {
		return http.DefaultTransport
	}
	return t.Base
}

//
type closeTracker struct {
	io.ReadCloser
	sp opentracing.Span
}

// Close the http response body and span
func (c closeTracker) Close() error {
	err := c.ReadCloser.Close()
	c.sp.LogKV("event", "ClosedBody")
	c.sp.Finish()
	return err
}
