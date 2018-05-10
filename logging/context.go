package logging

import (
	"context"
	"net/http"

	"gopkg.in/mgo.v2/bson"
)

type keyType int

const (
	loggerContextKey = iota
)

// From returns the logger in ctx. If no logger is found then the global
// logger is returned. For example if you pass context.Background() then
// you will get back the global logger. From will panic if ctx is nil.
func From(ctx context.Context) *Logger {
	if l, ok := ctx.Value(loggerContextKey).(*Logger); ok {
		return l
	}
	return Global()
}

// NewContext creates a new context that includes logger as a value.
// The logger can be retrieved using log.From(ctx)
func NewContext(ctx context.Context, logger *Logger) context.Context {
	return context.WithValue(ctx, loggerContextKey, logger)
}

// NewRequestContext creates a context with a new request logger.  The request
// logger will include the requestId in each trace. The request logger is
// derived from the logger found by calling logging.From(ctx). If no logger is found
// then nil is returned. If requestId is zero length then a new globally
// unique bson hex id will be generated.
func NewRequestContext(ctx context.Context, requestId string) context.Context {
	if len(requestId) == 0 {
		requestId = bson.NewObjectId().Hex()
	}
	logger := From(ctx)
	if logger == nil {
		return nil
	}
	logger = logger.With(RequestIdKey, requestId)
	return NewContext(ctx, logger)
}

// NewComponentContext creates a new context that includes a component logger. A
// component logger will trace the field {"component": component}. The parent
// logger is acquired by calling logging.From(ctx). If no logger is found then
// nil is returned.
func NewComponentContext(ctx context.Context, component string) context.Context {
	logger := From(ctx)
	if logger == nil {
		return nil
	}
	logger = logger.With(ComponentKey, component)
	return NewContext(ctx, logger)
}

// NewTestContext is a convenience function for use in unit tests when
// calling functions that expect a context with a logger. Call NewTestContext
// like this: logging.NewTestContext(t.Name()).
// A new context is created that contains a logger with testName as the service name.
// The context is derived from context.Background()
func NewTestContext(testName string) context.Context {
	ctx := context.Background()
	logger := New(testName)
	return NewContext(ctx, logger)
}

// Middleware HTTP Handlers

// RequestHandler provides an http.Handler that will extend the http context
// with a context created using NewRequestContext() to do requestId tracing.
type RequestHandler struct {
	handler      http.Handler // the next handler in the chain
	parentLogger *Logger
}

// NewRequestHandler returns a logging request handler that implements the
// http.HttpHandler interface. If parentLogger is nil then the global logger
// will be used as the parent logger.
func NewRequestHandler(parentLogger *Logger, handler http.Handler) *RequestHandler {
	return &RequestHandler{
		handler:      handler,
		parentLogger: parentLogger,
	}
}

func (self *RequestHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	requestId := r.Header.Get("X-Request-ID")
	ctx := r.Context()
	if self.parentLogger != nil {
		ctx = NewContext(ctx, self.parentLogger)
	}
	ctx = NewRequestContext(ctx, requestId)
	r = r.WithContext(ctx)
	self.handler.ServeHTTP(w, r)
}
