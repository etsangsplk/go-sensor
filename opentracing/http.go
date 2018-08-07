package opentracing

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strconv"

	"cd.splunkdev.com/libraries/go-observation/logging"
	"cd.splunkdev.com/libraries/go-observation/tracing"
	opentracing "github.com/opentracing/opentracing-go"
	tag "github.com/opentracing/opentracing-go/ext"
)

// InjectHTTPRequest injects the span from the HTTP request context into the HTTP request headers.
// If there is no span in the req.Context() then no span injection is done.
func InjectHTTPRequestWithSpan(req *http.Request) *http.Request {
	ctx := req.Context()
	span := opentracing.SpanFromContext(ctx)
	if span == nil {
		return req
	}
	tag.SpanKindRPCClient.Set(span)
	tagHTTPClientRequest(span, req)
	tagCurrentSpan(ctx, span)

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
	spanCtx := opentracing.ContextWithSpan(ctx, span)
	return req.WithContext(spanCtx)
}

// inboundHTTPRequest is used in http handler. When a http request arrived to server, if a request is being traced,
// the function will extract the current span from the http headers, and set HTTP related tags on the span. The span
// will be returned for upper layer to use.
// If the incoming http request is not created form any span, the server will create a new span.
// Also note that we tag the span with requestID and tenantID to make it easier to correlate
// with Splunk search.
func inboundHTTPRequest(ctx context.Context, operationName string, r *http.Request) opentracing.Span {
	// Recreate parent spancontext for child span creation.
	tracer := Global()
	parentSpanContext, err := tracer.Extract(
		opentracing.HTTPHeaders,
		opentracing.HTTPHeadersCarrier(r.Header),
	)

	if err != nil && err != opentracing.ErrSpanContextNotFound {
		logging.From(ctx).Warn("Error extracing span from HTTP request", logging.ErrorKey, err)
	}

	// Attach to parent span. In the case of an error it is ok if parentSpanContext is nil
	// Create child span from incoming request
	span := tracer.StartSpan(operationName, tag.RPCServerOption(parentSpanContext), opentracing.ChildOf(parentSpanContext))
	tag.HTTPMethod.Set(span, r.Method)
	tag.HTTPUrl.Set(span, r.URL.String())
	tagCurrentSpan(r.Context(), span)
	return span
}

// tagHTTPClientRequest adds common HTTP tags for method, host url, and hostname to the
// HTTP request span.
func tagHTTPClientRequest(span opentracing.Span, req *http.Request) {
	// Add some standard OpenTracing tags, useful in an HTTP request
	tag.HTTPMethod.Set(span, req.Method)
	tag.HTTPUrl.Set(
		span,
		fmt.Sprintf("%s://%s%s", req.URL.Scheme, req.URL.Host, req.URL.Path),
	)

	// Add information on the peer service we're about to contact.
	host, portString, err := net.SplitHostPort(req.URL.Host)
	if err == nil {
		tag.PeerHostname.Set(span, host)
		port, err := strconv.Atoi(portString)
		if err != nil {
			tag.PeerPort.Set(span, uint16(port))
		}
	} else {
		tag.PeerHostname.Set(span, req.URL.Host)
	}
}

// tagCurrentSpan tags the current span with additional information extracted from current http request context.
func tagCurrentSpan(ctx context.Context, span opentracing.Span) {
	reqID := tracing.RequestIDFrom(ctx)
	tenantID := tracing.TenantIDFrom(ctx)
	span.SetTag(tracing.RequestIDKey, reqID)
	span.SetTag(tracing.TenantKey, tenantID)
}

// httpOpenTracingHandler provides http middleware to construct
// opentracing trace span.
type httpOpenTracingHandler struct {
	next http.Handler
}

// NewHTTPOpentracingHandler constructs a new middleware that instance to
// create a new span when serve an incoming http request. If the incoming request
// is associated with a span, that will be the parent span; if not a new span will be created.
func NewHTTPOpenTracingHandler(next http.Handler) http.Handler {
	return &httpOpenTracingHandler{next: next}
}

func (h *httpOpenTracingHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	rw := tracing.NewHTTPResponseWriter(w)
	ctx := r.Context()
	operationName := tracing.OperationIDFrom(ctx)

	// Extract parent span context and start new span
	span := inboundHTTPRequest(ctx, operationName, r)
	defer span.Finish()

	// Put new span into request context
	r = r.WithContext(opentracing.ContextWithSpan(r.Context(), span))

	h.next.ServeHTTP(rw, r)
	tag.HTTPStatusCode.Set(span, uint16(rw.StatusCode()))
}
