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

func WithOperationID(ctx context.Context, operationID string) context.Context {
	return context.WithValue(ctx, operationIDContextKey, operationID)
}

func OperationIDFrom(ctx context.Context) string {
	if v, ok := ctx.Value(operationIDContextKey).(string); ok {
		return v
	}
	return ""
}

func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDContextKey, requestID)
}

func RequestIDFrom(ctx context.Context) string {
	if v, ok := ctx.Value(requestIDContextKey).(string); ok {
		return v
	}
	return ""
}

func WithTenantID(ctx context.Context, tenantID string) context.Context {
	return context.WithValue(ctx, tenantIDContextKey, tenantID)
}

func TenantIDFrom(ctx context.Context) string {
	if v, ok := ctx.Value(tenantIDContextKey).(string); ok {
		return v
	}
	return ""
}
