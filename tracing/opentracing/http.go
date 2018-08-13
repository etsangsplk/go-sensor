package opentracing

import (
	"fmt"
	"net"
	"net/http"
	"strconv"

	opentracing "github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"

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
		span := opentracing.SpanFromContext(ctx)
		if span != nil {
			// We are going to use this span in a client request, so mark as such.
			ext.SpanKindRPCClient.Set(span)
			// Add some standard OpenTracing tags, useful in an HTTP request.
			ext.HTTPMethod.Set(span, req.Method)
			ext.HTTPUrl.Set(
				span,
				fmt.Sprintf("%s://%s%s", req.URL.Scheme, req.URL.Host, req.URL.Path),
			)

			// Add information on the peer service we're about to contact.
			host, portString, err := net.SplitHostPort(req.URL.Host)
			if err == nil {
				ext.PeerHostname.Set(span, host)
				if port, err := strconv.Atoi(portString); err != nil {
					ext.PeerPort.Set(span, uint16(port))
				}
			} else {
				ext.PeerHostname.Set(span, req.URL.Host)
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
				ext.Error.Set(span, true)
				logger := logging.Global()
				logger.Error(err, "error inject span to request")
			}
			spanCtx := opentracing.ContextWithSpan(ctx, span)
			req = req.WithContext(spanCtx)
		}
		return req
	}
}

// InboundHTTPRequest .
func InboundHTTPRequest(tracer opentracing.Tracer, operationName string, r *http.Request) (opentracing.Span, error) {
	// Recreate parent spancontext for child span creation.
	parentSpanContext, err := tracer.Extract(
		opentracing.HTTPHeaders,
		opentracing.HTTPHeadersCarrier(r.Header),
	)

	if err != nil && err != opentracing.ErrSpanContextNotFound {
		return nil, err
	}
	// Attach to parent span
	// parentSpanReference := opentracing.ChildOf(parentSpanContext)
	// Create child span from incoming request
	span := tracer.StartSpan(operationName, ext.RPCServerOption(parentSpanContext))
	ext.HTTPMethod.Set(span, r.Method)
	ext.HTTPUrl.Set(span, r.URL.String())

	// These two items will flow to the child span
	reqID := span.BaggageItem(tracing.RequestIDKey)
	tenant := span.BaggageItem(tracing.TenantKey)
	span.LogKV(tracing.RequestIDKey, reqID)
	span.LogKV(tracing.TenantKey, tenant)

	// update request context to include our opentracing context
	return span, nil
}

// httpOpentracingHandler provides http middleware to construct
// opentracing trace span.
type httpOpentracingHandler struct {
	// tracer
	// logger
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
	operationName := tracing.OperationIDFrom(r.Context())
	reqId := tracing.RequestIDFrom(r.Context())
	tenant := tracing.TenantIDFrom(r.Context())
	// Assume that there a tracer is already setup by service
	// defaulted as noop tracer.
	tracer := Global() //TODO? get from opentracinghandler even it is global  per service?
	span, err := InboundHTTPRequest(tracer, operationName, r)
	defer func() {
		if span != nil {
			span.Finish()
		}
	}()
	if err != nil {
		logger := logging.Global()
		logger.Error(err, "error extract span from request")
	}
	// Tagging the current span
	span.SetTag(tracing.RequestIDKey, reqId)
	span.SetTag(tracing.TenantKey, tenant)

	// These two items will flow to the child span
	span = span.SetBaggageItem(tracing.RequestIDKey, reqId)
	span = span.SetBaggageItem(tracing.TenantKey, tenant)

	r = r.WithContext(opentracing.ContextWithSpan(r.Context(), span))
	// serve the real operation.
	h.next.ServeHTTP(rw, r)
	ext.HTTPStatusCode.Set(span, uint16(rw.StatusCode()))
}
