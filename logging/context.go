package logging

import (
	"context"
	"net/http"

	"gopkg.in/mgo.v2/bson"
)

// Custom type used for context keys, avoid conflicts with other libraries.
type keyType int

const (
	loggerContextKey keyType = iota
)

// From returns the logger in ctx. If no logger is found then the global
// logger is returned. For example if you pass context.Background() then
// you will get back the global logger. From will panic if ctx is nil.
func From(ctx context.Context) *Logger {
	l := ctx.Value(loggerContextKey).(*Logger)
	if l != nil {
		return l
	}
	return Global()
}

// NewContext creates a new context that includes logger as a context value.
// Use this to add a customized logger to a context and then flow that context
// through your function calls. If optional fields are provided the supplied
// logger will be first extended with those fields via logger.With().
// After a logger is added to a context it can be retrieved using log.From(ctx).
func NewContext(ctx context.Context, logger *Logger, fields ...interface{}) context.Context {
	if len(fields) > 0 {
		logger = logger.With(fields...)
	}

	return context.WithValue(ctx, loggerContextKey, logger)
}

// NewRequestContext creates a context with a new request logger.  The request
// logger will include the requestId in each trace. The request logger is
// derived from the logger found by calling logging.From(ctx). If no logger is found
// then nil is returned. If requestId is zero length then a new globally
// unique bson hex id will be generated. If optional fields are provided the supplied
// logger will be first extended with those fields via logger.With().
func NewRequestContext(ctx context.Context, requestID string, fields ...interface{}) context.Context {
	if len(requestID) == 0 {
		requestID = bson.NewObjectId().Hex()
	}
	logger := From(ctx)
	logger = logger.With(RequestIdKey, requestID)
	return NewContext(ctx, logger, fields...)
}

// NewComponentContext creates a new context that includes a component logger. A
// component logger will trace the field {"component": component}. The parent
// logger is acquired by calling logging.From(ctx). If no logger is found then
// nil is returned. If optional fields are provided the supplied
// logger will be first extended with those fields via logger.With().
func NewComponentContext(ctx context.Context, component string, fields ...interface{}) context.Context {
	logger := From(ctx)
	logger = logger.With(ComponentKey, component)
	return NewContext(ctx, logger, fields...)
}

// NewTestContext is a convenience function for use in unit tests when
// calling functions that expect a context with a logger. Call NewTestContext
// like this: logging.NewTestContext(t.Name()).
// A new context is created that contains a logger with testName as the service name.
// The context is derived from context.Background(). If optional fields are provided the supplied
// logger will be first extended with those fields via logger.With().
func NewTestContext(testName string, fields ...interface{}) context.Context {
	ctx := context.Background()
	logger := New(testName)
	return NewContext(ctx, logger, fields...)
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

// ServeHTTP implements the http.Handler interface.
func (self *RequestHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// r.Header.Get("X-REQUEST-ID") will return a uuid requestId if there is one
	// or else an empty string, that can be forwarded downstream.
	// This is useful for tracing and/or correlation.
	requestID := r.Header.Get("X-REQUEST-ID")
	ctx := r.Context()
	if self.parentLogger != nil {
		ctx = NewContext(ctx, self.parentLogger)
	}
	ctx = NewRequestContext(ctx, requestID)
	r = r.WithContext(ctx)
	self.handler.ServeHTTP(w, r)
}
