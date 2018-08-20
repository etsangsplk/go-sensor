package tracing

import (
    "github.com/aws/aws-sdk-go/aws/request"
    "github.com/aws/aws-sdk-go/aws/session"
    "github.com/opentracing/opentracing-go/ext"

    "cd.splunkdev.com/libraries/go-observation/opentracing"
    "cd.splunkdev.com/libraries/go-observation/tracing"
)

// When you initiate any resource client from AWS session, session carries the configuration to make the request
// as well as a set of default request handlers, that will be passed to resource client.
// AWS Client calls a list of request handlers before sending out a raw http request.
// 
// For set of request handlers see: https://github.com/aws/aws-sdk-go/blob/master/aws/request/handlers.go
//
// We can create 2 handlers Send and Complete for Span creation and span tagging.

type handlers struct{}

// Session wraps a session.Session, causing requests and responses to be traced.
func Session(s *session.Session) *session.Session {
    s = s.Copy()
    h := &handlers{}
    s.Handlers.Send.PushFrontNamed(request.NamedHandler{
        Name: "opentracing.Send",
        Fn:   h.Send,
    })
    s.Handlers.Complete.PushBackNamed(request.NamedHandler{
        Name: "opentracing.Complete",
        Fn:   h.Complete,
    })
    return s
}

func (h *handlers) Send(req *request.Request) {
    span, ctx := opentracing.StartSpanFromContext(req.Context(), h.operationName(req))
    ext.SpanKindRPCClient.Set(span)
    span = span.SetTag("aws.serviceName", h.serviceName(req))
    span = span.SetTag("aws.resource", h.resourceName(req))
    span = span.SetTag("aws.agent", h.awsAgent(req))
    span = span.SetTag("aws.operation", h.awsOperation(req))
    span = span.SetTag("aws.region", h.awsRegion(req))
    span = span.SetTag("aws.requestID", h.awsRequestID(req))
    // Upstream needs to set these in request context or empty string.
    span = span.SetTag("tenant", tracing.TenantIDFrom(req.Context()))
    span = span.SetTag("requestID", tracing.RequestIDFrom(req.Context()))
    ext.HTTPMethod.Set(span, req.Operation.HTTPMethod)
    ext.HTTPUrl.Set(span, req.HTTPRequest.URL.String())

    req.SetContext(ctx)
}

func (h *handlers) Complete(req *request.Request) {
    span := opentracing.SpanFromContext(req.Context())
    defer span.Finish()
    defer opentracing.FailIfError(span, req.Error)
    if req.HTTPResponse != nil {
        ext.HTTPStatusCode.Set(span, uint16(req.HTTPResponse.StatusCode))
    }
}

func (h *handlers) operationName(req *request.Request) string {
    return h.awsService(req) + ".command"
}

func (h *handlers) resourceName(req *request.Request) string {
    return h.awsService(req) + "." + req.Operation.Name
}

func (h *handlers) serviceName(req *request.Request) string {
    return "aws." + h.awsService(req)
}

func (h *handlers) awsAgent(req *request.Request) string {
    agent := req.HTTPRequest.Header.Get("User-Agent")
    if agent != "" {
        return agent
    }
    return "aws-sdk-go"
}

func (h *handlers) awsOperation(req *request.Request) string {
    return req.Operation.Name
}

func (h *handlers) awsRegion(req *request.Request) string {
    return req.ClientInfo.SigningRegion
}

func (h *handlers) awsService(req *request.Request) string {
    return req.ClientInfo.ServiceName
}

func (h *handlers) awsRequestID(req *request.Request) string {
    return req.RequestID
}
