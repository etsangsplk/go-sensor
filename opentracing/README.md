# Table of Contents

Opentracing is a vendor-neutral open standard for distributed tracing. If we trace our method calls via OpenTracing APIs, we can swap out our tracing vendors.

Distributed tracing attempts to help provide developers with more information about the behaviour of complex distributed systems. When an individual request traverses dozens or more systems with numerous components on each system, looking at logs emitted by each system in isolation assume that engineers have expert knowledge of the system end-to-end, system is in sync with what's in engineers mind, the dependencies are relatively easy to understand even it is a graph of dependecies.

## Terminology and core concepts

See opentracing spec for mor information.

* _Tracer: Tracer interface creates Spans and understands how to Inject (serialize) and Extract (deserialize) them across process boundaries. Opentracing does not provide concrete implementation of Tracer, which is up to vendor.

* _Span: Spans are logical units of work in a distributed system, and by definition they all have a name, a start time, and a duration. In a trace, Spans are associated with the distributed system component that generated them. See opentracing spec for mor information.

* _Relationship between Spans: Relationships are the connections between Spans. A Span may reference zero or more other Spans that are causally related. These connections between Spans help describe the semantics of the running system, as well as the critical path for latency-sensitive (distributed) transactions. Relationship are wired together by SpanContext.

* _SpanContext The SpanContext is more of a "concept" than a useful piece of functionality at the generic OpenTracing layer. Most OpenTracing users only interact with SpanContext via references when starting new Spans, or when injecting/extracting a trace to/from some transport protocol.

* _Span_Tag  A _tag_ is a key-value pair that provides certain metadata about the span instance. A _log_ is similar to a regular log statement, it contains a timestamp and some data, but it is associated with span from which it was logged. OpenTracing project documents certain "standard tags" that have prescribed semantic meanings.

* _Span_Log The OpenTracing Specification also recommends all log statements to contain an `event` field that describes the overall event being logged, with other attributes of the event provided as additional fields [opentracing semantic conventions]. OpenTracing project documents certain "standard log keys" which have prescribed semantic meanings. Opentracing does not dictate what log levels a tracer should support.

* _Span_Tag vs _Span_Log
OpenTracing API does not dictate how we do it; the general principle is that information that applies to the span as a whole should be recorded as a tag, while events that have timestamps should be recorded as logs

# Register a Tracer
Usually only one concrete Tracer will be associated with one microservice. A concrete tracer takes care of all the io and serialization management of tracing spans plus other functionality like sampling of of sending the traces collected.

```
import (
    "cd.splunkdev.com/libraries/go-observation/tracing"
)

const serviceName = "api-gateway"

func main() {

    // Create, set tracer and bind tracer to a service name
    // Usually you only need 1 tracer for per microservice.
    tracer, closer := ssctracing.NewTracer(serviceName, logger)
    // Closing a tracer for resource management is important when done.
    defer closer.Close()
    // Setting this tracer globally so that it will be available
    // to rest of microservice
    ssctracing.SetGlobalTracer(tracer)
    ... 
}

```

# Creating and finishing a Span
Use a registered tracer to create a Span with an operation name. An operation name is meant to
represent a _class of spans_, rather than a unique instance. This is important because when a user try to search for an 
operation by name in backend, it will be very bad user experience if the operation name is too specific. Another reason for choosing more general operation names is to allow the tracing systems to do aggregations.

```
    // Get the registered tracer for this microservice
    tracer := ssctracing.Global()
    // Start a span with an operation name
    span := tracer.StartSpan("user-registration")
    .... do something else
    // Must finish span before validation.
    span.Finish()

```

# Creating and finishing a child Span
To show a casual relationship between spans, we can create a child span and referring it to the parent span through SpanContext wrapped inside context.Context. Note that SpanContext is not context.Context. if we cannot derive a Span from context, a new Span is created.
Note the HTTP middleware automatically create a childSpan from incoming http request's request Context.

```
import (
    ssctracing "cd.splunkdev.com/libraries/go-observation/opentracing"
)

....
    ctx := context.Background()
    parentSpan := tracer.StartSpan("parent span")

    // create a child span from parent span context
    parentCtx := ssctracing.ContextWithSpan(ctx, parentSpan)
    childSpan, err := ssctracing.StartSpanFromContext(parentCtx, "child span")
    // Omit error handling for brevity

    // Must do resource clean up.
    defer parentSpan.Finish()
    defer childSpan.Finish()

```

# Tagging a Span
The tags are meant to describe attributes of the span that apply to the whole duration of the span. 

```
import (
    ssctracing "cd.splunkdev.com/libraries/go-observation/opentracing"
)
...
    
    tracer := ssctracing.Global()
    span := tracer.StartSpan("user-registration")
    .... do something else
    span.SetTag("organization","splunk")

```

# Logging events to a span
If you have some event that there is a clear timestamp associated with it within a span, it is a good practice to 
log such events in key-value pairs that can be automatically processed by log aggregation systems.

```
    // Get the registered tracer for this microservice
    tracer := ssctracing.Global()
    // Start a span with an operation name
    span := tracer.StartSpan("user-registration")
    span.LogKV("event", "start query customer db", "user", "value1")
    .... do database operations
    span.LogKV("event", "finish query customer db", "user", "value1")
    // Must finish span before validation.
    span.Finish()

```


# HTTP Middleware

The package provides middleware for automatically create spans on server for incoming request and tagging stardard Span tags to current span.

See the [ssc-observation README](https://github.com/splunk/ssc-observation) for details on how to configure the middleware pipeline.

You must use the following code to register the middleware:

```go
import (
      kvmetrics "github.com/splunk/kvstore-service/kvstore/metrics"
      ssctracing "github.com/splunk/ssc-observation/opentracing"
)
... 
// Simulated microservice A, serving requests.
func Service(hostPort string, wg *sync.WaitGroup) {
    // Configure Route http requests
    http.Handle("/tenant1/operationA",
            ssctracing.NewHTTPOpentracingHandler(
                http.HandlerFunc(operationAHandler))
   .... enacted ... 
}
```


# Running benchmark
From top folder,

```
go test -bench=.  ./...

```


Resouces:
[Dapper]: https://research.google.com/archive/papers/dapper-2010-1.pdf
[opentracing large systems]: http://opentracing.io/documentation/pages/instrumentation/instrumenting-large-systems.html
[opentracing spec]: http://opentracing.io/documentation/pages/spec
[opentracing semantic conventions]: https://github.com/opentracing/specification/blob/master/semantic_conventions.md
[opentracing-go
godocs](https://godoc.org/github.com/opentracing/opentracing-go).
