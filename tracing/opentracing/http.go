package opentracing

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strconv"

	opentracing "github.com/opentracing/opentracing-go"
	tag "github.com/opentracing/opentracing-go/ext"

	// Because of referencing logging to get the requestID etc,
	// to avoid (import cycle), I now put everything under opentracing folder
	// instead if just tracing. Suggestion?
	"github.com/splunk/ssc-observation/logging"
	"github.com/splunk/ssc-observation/tracing"
)

// RequestFunc is a function to inject outgoing HTTP request data to the initiating span,
// so that the initiating span can choose to tag about information like peer service hostname etc.
type RequestFunc func(req *http.Request) *http.Request

// OutboundHTTPRequest returns a RequestFunc that injects a span so the server serving this rquest
// will have the information to continue the trace (create child span etc).
// If no such Span can be found, the RequestFunc is a noop.
func OutboundHTTPRequest(tracer opentracing.Tracer) RequestFunc {
	return func(req *http.Request) *http.Request {
		// Retrieve the Span from request context.
		ctx := req.Context()
		// This does not create a new Span.
		span := opentracing.SpanFromContext(ctx)
		if span != nil {
			// We are going to use this span in a client request, so mark as such.
			tag.SpanKindRPCClient.Set(span)
			// Add some standard OpenTracing tags, useful in an HTTP request.
			tag.HTTPMethod.Set(span, req.Method)
			tag.HTTPUrl.Set(
				span,
				fmt.Sprintf("%s://%s%s", req.URL.Scheme, req.URL.Host, req.URL.Path),
			)

			// Add information on the peer service we're about to contact.
			host, portString, err := net.SplitHostPort(req.URL.Host)
			if err == nil {
				tag.PeerHostname.Set(span, host)
				if port, err := strconv.Atoi(portString); err != nil {
					tag.PeerPort.Set(span, uint16(port))
				}
			} else {
				tag.PeerHostname.Set(span, req.URL.Host)
			}

			// Inject the Span context into the outgoing HTTP Request.
			// Sicne we are sending an HTTP request, will use the HTTP headers as carrier.
			err = tracer.Inject(
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
			req = req.WithContext(spanCtx)
		}
		return req
	}
}

// InboundHTTPRequest .
func InboundHTTPRequest(operationName string, r *http.Request) (opentracing.Span, error) {
	// Assume that there a tracer is already setup by service
	// defaulted as noop tracer.
	tracer := Global()

	// Recreate parent spancontext for child span creation.
	parentSpanContext, err := tracer.Extract(
		opentracing.HTTPHeaders,
		opentracing.HTTPHeadersCarrier(r.Header),
	)

	if err != nil && err != opentracing.ErrSpanContextNotFound {
		return nil, err
	}
	// Attach to parent span.
	// Create child span from incoming request
	span := tracer.StartSpan(operationName, tag.RPCServerOption(parentSpanContext), opentracing.ChildOf(parentSpanContext))
	tag.HTTPMethod.Set(span, r.Method)
	tag.HTTPUrl.Set(span, r.URL.String())

	// update request context to include our opentracing context
	return span, nil
}

// httpOpentracingHandler provides http middleware to construct
// opentracing trace span.
type httpOpentracingHandler struct {
	next http.Handler
}

// NewHTTPOpentracingHandler constructs a new middleware that instance to
// create a new span when serve an incoming http request. If the incoming request
// is associated with a span, that will be the parent span; if not a new span will be created.
func NewHTTPOpentracingHandler(next http.Handler) http.Handler {
	return &httpOpentracingHandler{next: next}
}

func (h *httpOpentracingHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	rw := tracing.NewHTTPResponseWriter(w)

	ctx := r.Context()
	operationName := tracing.OperationIDFrom(ctx)

	span, err := InboundHTTPRequest(operationName, r)
	defer func() {
		if span != nil {
			span.Finish()
		}
	}()
	if err != nil {
		logger := logging.Global()
		logger.Error(err, "error extract span from request")
	}

	tagCurrentSpan(ctx, span)

	r = r.WithContext(opentracing.ContextWithSpan(r.Context(), span))
	// serve the real operation.
	h.next.ServeHTTP(rw, r)
	tag.HTTPStatusCode.Set(span, uint16(rw.StatusCode()))
}

// Tag the current span with addtitional information extracted from current http request context.
func tagCurrentSpan(ctx context.Context, span opentracing.Span) {
	reqId := tracing.RequestIDFrom(ctx)
	tenant := tracing.TenantIDFrom(ctx)
	span.SetTag(tracing.RequestIDKey, reqId)
	span.SetTag(tracing.TenantKey, tenant)
}
