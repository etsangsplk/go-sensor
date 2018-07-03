package tracing

import (
	"context"
)

type keyType int

const (
	operationIDContextKey keyType = iota
	requestIDContextKey
	tenantIDContextKey
)

// WithOperationID creates a new context that includes the value operationID.
// Use OperationIDFrom() to get the value back out.
func WithOperationID(ctx context.Context, operationID string) context.Context {
	return context.WithValue(ctx, operationIDContextKey, operationID)
}

// OperationIDFrom returns the operationID value from context. If that value
// has not been set in the context then the empty string is returned.
func OperationIDFrom(ctx context.Context) string {
	if v, ok := ctx.Value(operationIDContextKey).(string); ok {
		return v
	}
	return ""
}

// WithRequestID creates a new context that includes the value requestID.
// Use RequestIDFrom() to get the value back out.
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDContextKey, requestID)
}

// RequestIDFrom returns the tenantID value from context. If that value
// has not been set in the context then the empty string is returned.
func RequestIDFrom(ctx context.Context) string {
	if v, ok := ctx.Value(requestIDContextKey).(string); ok {
		return v
	}
	return ""
}

// WithTenantID creates a new context that includes the value tenantID.
// Use TenantIDFrom() to get the value back out.
func WithTenantID(ctx context.Context, tenantID string) context.Context {
	return context.WithValue(ctx, tenantIDContextKey, tenantID)
}

// TenantIDFrom returns the tenantID value from context. If that value
// has not been set in the context then the empty string is returned.
func TenantIDFrom(ctx context.Context) string {
	if v, ok := ctx.Value(tenantIDContextKey).(string); ok {
		return v
	}
	return ""
}
