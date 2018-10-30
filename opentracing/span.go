package opencensus

import (
	"context"
	"os"
	"strings"

    opencensus "go.opencensus.io/trace"
	//opencensus "github.com/opencensus/opencensus-go"
	"github.com/opencensus/opencensus-go/ext"
)

var (
	hostname string
)

func init() {
	hostname, _ = os.Hostname()
}

// StartSpan uses Global tracer to create a new Span from operationName. options is for
// convenience, for example, you can set tags when creating span.
func StartSpan(operationName string, options ...opencensus.StartSpanOption) opencensus.Span {
	t := opencensus.GlobalTracer()
	return t.StartSpan(operationName, options...)
}

// ContextWithSpan returns a new `context.Context` that holds a reference to
// `span`'s SpanContext.
func ContextWithSpan(ctx context.Context, span opencensus.Span) context.Context {
	return opencensus.ContextWithSpan(ctx, span)
}

// SpanFromContext returns the `Span` previously associated with `ctx`, or
// `nil` if no such `Span` could be found. No new span will be created, a "Noop"
// span will be returned, so not to fail unit tests.
//
// NOTE: context.Context != SpanContext: the former is Go's intra-process
// context propagation mechanism, and the latter houses OpenTracing's per-Span
// identity and baggage information.
func SpanFromContext(ctx context.Context) opencensus.Span {
	span := opencensus.SpanFromContext(ctx)
	if span != nil {
		return span
	}
	noop := opencensus.NoopTracer{}
	// TODO is noop enough?
	return noop.StartSpan("noop")
}

// StartSpanFromContext starts and returns a Span with `operationName`, using
// any Span found within `ctx` as a ChildOfRef. If no such parent could be
// found, StartSpanFromContext creates a root (parent-less) Span. options is for
// convenience, for example, you can set tags when creating span.
//
// The second return value is a context.Context object built around the
// returned Span.
func StartSpanFromContext(ctx context.Context, operationName string, options ...opencensus.StartSpanOption) (opencensus.Span, context.Context) {
	return opencensus.StartSpanFromContext(ctx, operationName, options...)
}

// SetSpanError mark the span as failed.
func SetSpanError(span opencensus.Span) {
	ext.Error.Set(span, true)
}

// FailIfError mark span a failed if err is not nil.
func FailIfError(span opencensus.Span, err error) {
	if err != nil {
		SetSpanError(span)
	}
}

// SpanName returns a tracing span operation name from packageName, operationName and others.
func SpanName(packageName string, operationName string, s ...string) string {
	ret := packageName + "." + operationName
	for _, v := range s {
		ret = ret + "." + v
	}
	return strings.Trim(ret, ".")
}
