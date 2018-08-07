package opentracing

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	opentracing "github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"

	// Because of referencing logging to get the requestID etc,
	// to avoid (import cycle), I now put everything under opentracing folder
	// instead if just tracing. Suggestion?
	"github.com/splunk/ssc-observation/logging"
	"github.com/splunk/ssc-observation/tracing"
)

type Client struct {
	http.Client
	httpClient     *http.Client
	requestContext context.Context
	tracer         Tracer
	traceRequest   RequestFunc // Request wrapper
}

// NewHTTPClient returns a new client instance from upstream ctx context.
// It wraps http.client and add opentracing meta data into outbound http.request.
func NewHTTPClient(ctx context.Context, tracer opentracing.Tracer) *Client {
	return &Client{
		requestContext: ctx,
		httpClient:     &http.Client{},
		tracer:         tracer,
		traceRequest:   OutboundHTTPRequest(tracer),
	}
}

// Do adds opentracing http header instrumentation before making a http Do request.
func (c *Client) Do(req *http.Request) (*http.Response, error) {
	ctx := c.requestContext
	req = c.traceRequest(req.WithContext(ctx))
	return c.httpClient.Do(req)
}

// Get adds opentracing http header instrumentation before making a http Get request.
func (c *Client) Get(url string) (resp *http.Response, err error) {
	ctx := c.requestContext
	// Since we need to annotate the http header, we need to use NewRequest
	// Looks like golang also just use nil for last parameter.
	// https://github.com/golang/go/blob/1b9f66330b123b042e738eff5e47e869dd301a98/src/net/http/client.go#L369
	req, err := http.NewRequest(http.MethodGet, url, nil)
	req = c.traceRequest(req.WithContext(ctx))
	return c.httpClient.Do(req)
}

// Head adds opentracing http header instrumentation before making a http Head request.
func (c *Client) Head(url string) (resp *http.Response, err error) {
	ctx := c.requestContext
	req, err := http.NewRequest(http.MethodHead, url, nil)
	req = c.traceRequest(req.WithContext(ctx))
	return c.httpClient.Do(req)
}

// Post adds opentracing http header instrumentation before making a http Post request.
func (c *Client) Post(url string, contentType string, body io.Reader) (resp *http.Response, err error) {
	ctx := c.requestContext
	// https://github.com/golang/go/blob/1b9f66330b123b042e738eff5e47e869dd301a98/src/net/http/client.go#L736
	req, err := http.NewRequest(http.MethodPost, url, body)
	req = c.traceRequest(req.WithContext(ctx))
	return c.httpClient.Do(req)
}

// PostForm adds opentracing http header instrumentation before making a http PostForm request.
func (c *Client) PostForm(url string, data url.Values) (resp *http.Response, err error) {
	// https://github.com/golang/go/blob/1b9f66330b123b042e738eff5e47e869dd301a98/src/net/http/client.go#L774
	return c.Post(url, "application/x-www-form-urlencoded", strings.NewReader(data.Encode()))
}

// RequestFunc is a function to inject outgoing HTTP request data to the initiating span,
// so that the initiating span can choose to tag about information like peer service hostname etc.
type RequestFunc func(req *http.Request) *http.Request

// OutboundHTTPRequest returns a RequestFunc that injects a span so the server serving this rquest
// will have the information to continue the trace (create child span etc).
// If no such Span can be found, the RequestFunc is a noop. A new span starts when you make the call.
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
			if err := tracer.Inject(
				span.Context(),
				opentracing.HTTPHeaders,
				opentracing.HTTPHeadersCarrier(req.Header),
			); err != nil {
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
	//parentSpanReference := opentracing.ChildOf(parentSpanContext)
	// Create child span from incoming request
	span := tracer.StartSpan(operationName, ext.RPCServerOption(parentSpanContext))
	ext.HTTPMethod.Set(span, r.Method)
	ext.HTTPUrl.Set(span, r.URL.String())

	// These two items will flow to the child span
	reqID := span.BaggageItem(logging.RequestIDKey)
	tenant := span.BaggageItem(logging.TenantKey)
	span.LogKV(logging.RequestIDKey, reqID)
	span.LogKV(logging.TenantKey, tenant)

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
	rw := newHTTPResponseWriter(w)
	operationName := tracing.OperationIDFrom(r.Context())
	reqId := tracing.RequestIDFrom(r.Context())
	tenant := tracing.TenantIDFrom(r.Context())
	// Assume that there a tracer is already setup by service
	// defaulted as noop tracer.
	tracer := GlobalTracer()
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
	span.SetTag(logging.RequestIDKey, reqId)
	span.SetTag(logging.TenantKey, tenant)

	// These two items will flow to the child span
	span = span.SetBaggageItem(logging.RequestIDKey, reqId)
	span = span.SetBaggageItem(logging.TenantKey, tenant)

	r = r.WithContext(opentracing.ContextWithSpan(r.Context(), span))
	// serve the real operation.
	h.next.ServeHTTP(rw, r)
	ext.HTTPStatusCode.Set(span, uint16(rw.StatusCode()))
}
