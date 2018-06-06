package logging

import (
	"context"

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
	if l, ok := ctx.Value(loggerContextKey).(*Logger); ok {
		return l
	}
	return Global()
}

// NewContext extends context ctx with logger as a context value.
// Use this to add a customized logger to a context and then flow that context
// through your function calls. If optional fields are provided the supplied
// logger will be first extended with those fields via logger.With().
// After a logger is added to a context it can be retrieved using logging.From(ctx).
func NewContext(ctx context.Context, logger *Logger, fields ...interface{}) context.Context {
	if len(fields) > 0 {
		logger = logger.With(fields...)
	}

	return context.WithValue(ctx, loggerContextKey, logger)
}

// newContextWithField extends the logger found in ctx with the field specified by key and field. If the optional
// fields is specified then those are added to the logger as well. The new logger is then added to ctx and
// that is returned. If no logger is found then nil is returned.
func newContextWithField(ctx context.Context, key string, field string, fields ...interface{}) context.Context {
	logger := From(ctx)
	if logger == nil {
		return nil
	}
	logger = logger.With(key, field)
	return NewContext(ctx, logger, fields...)
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
	return newContextWithField(ctx, RequestIdKey, requestID, fields...)
}

// NewTenantContext extends the context ctx with a new logger with the tenant field
// with value tenantID. The new logger is
// derived from the logger found by calling logging.From(ctx). If no logger is found
// then nil is returned. If optional fields are provided the supplied
// logger will be first extended with those fields via logger.With().
func NewTenantContext(ctx context.Context, tenantID string, fields ...interface{}) context.Context {
	return newContextWithField(ctx, TenantKey, tenantID, fields...)
}

// NewComponentContext creates a new context that includes a component logger. A
// component logger will trace the field {"component": component}. The parent
// logger is acquired by calling logging.From(ctx). If no logger is found then
// nil is returned. If optional fields are provided the supplied
// logger will be first extended with those fields via logger.With().
func NewComponentContext(ctx context.Context, component string, fields ...interface{}) context.Context {
	return newContextWithField(ctx, ComponentKey, component, fields...)
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
