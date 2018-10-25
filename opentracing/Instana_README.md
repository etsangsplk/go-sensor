# Table of Contents

Opentracing is a vendor-neutral open standard for distributed tracing. Opentracing  only concerns about the data model, what a Span should look like, but this means you have to choose 1 vendor. Its main goals are:

* to modernize current logging/tracing tool chain for distributed Microservices environment.

* bring the focus to request transaction level so to:

    * help engineers to understand how a system is behaving, without being a domain expert

    * show what the system is really doing

    * show areas of interest at Root Cause Analysis

    * show system behavior at steady state so people can make architectural improvements

    * make extra information about the whole transaction searchable, though not enforced by standard.
      User can usually search via Span Tags, Span Operation Name etc about a Span. Then investigate more by lookin at what events associated with the Span.

## Background

A typical setup for Application instrumentation is as the following block diagram.

Application Instrumentation.
![alt text](./Application.png?raw=true)

Tracing information is propagated along with request as some sort of Headers, hop across Microservices within a process or cross processes. Since Opentracing only deals with data model part, and we have to rely on a tracer (see terminology below). Our library consists of 2 levels:

* a vendor-neutral part

   * when appropriate wrap around Opentracing API to provide convenience

   * wrap around our logging library, so when we log events to a Span (see below), we also log to SKYNET

   * http utility, so to lessen to burden to instrument with a default set of Tags incoming and outgoing HTTP request both on the http client side and the standard http server middleware.

* a vendor specific part:

   * vendors sometimes provides their own convince function on top of standard Opentracing API, to maintain vendor neutral, we only the vendor API that agrees with standard Opentracing API

   * expose just enough API to configure vendor tracer

   * our extension to vendor API when it makes sense. For example, LightStep allows user to record and send Span information to other data sink, we leverage that to log span information via our logging library to SKYNET.


## Terminology and core concepts

See Opentracing spec for more information.

* _Tracer: Tracer interface creates Spans and understands how to Inject (serialize) and Extract (deserialize) them across process boundaries. Opentracing does not provide concrete implementation of Tracer, which is up to vendor.

* _Span: Spans are logical units of work in a distributed system, and by definition they all have a name, a start time, and a duration. In a trace, Spans are associated with the distributed system component that generated them. See Opentracing spec for more information.

* _Relationship between Spans: Relationships are the connections between Spans. A Span may reference zero or more other Spans that are causally related. These connections between Spans help describe the semantics of the running system, as well as the critical path for latency-sensitive (distributed) transactions. Relationship are wired together by SpanContext.

* _SpanContext The SpanContext is more of a "concept" than a useful piece of functionality at the generic OpenTracing layer. Most OpenTracing users only interact with SpanContext via references when starting new Spans, or when injecting/extracting a trace to/from some transport protocol.

* _Span_Tag  A _tag_ is a key-value pair that provides certain metadata about the span instance. A _log_ is similar to a regular log statement, it contains a timestamp and some data, but it is associated with span from which it was logged. OpenTracing project documents certain "standard tags" that have prescribed semantic meanings.

* _Span_Log The OpenTracing Specification also recommends all log statements to contain an `event` field that describes the overall event being logged, with other attributes of the event provided as additional fields [Opentracing semantic conventions]. OpenTracing project documents certain "standard log keys" which have prescribed semantic meanings. Opentracing does not dictate what log levels a tracer should support.

* _Span_Tag vs _Span_Log
OpenTracing API does not dictate how we do it; the general principle is that information that applies to the span as a whole should be recorded as a tag, while events that have timestamps should be recorded as logs

# Tracer Configuration Settings 
See README under instana

# Register a Tracer
Usually only one concrete Tracer will be associated with one Microservice. A concrete tracer takes care of all the io and serialization management of tracing spans plus other functionality like sampling of of sending the traces collected.

```
import (
    "cd.splunkdev.com/libraries/go-observation/tracing"
    "cd.splunkdev.com/libraries/go-observation/opentracing"
)

const serviceName = "api-gateway"

func main() {

    // Create, set tracer and bind tracer to a service name
    // Usually you only need 1 tracer for per microservice.
    tracer, closer := opentracing.NewTracer(serviceName, logger)
    // Closing a tracer for resource management is important when done.
    defer closer.Close()
    // Setting this tracer globally so that it will be available
    // to rest of microservice
    opentracing.SetGlobalTracer(tracer)
    ...
}

```

# Creating and finishing a Span
Use a registered tracer to create a Span with an operation name. An operation name is meant to
represent a _class of spans_, rather than a unique instance. This is important because when a user try to search for an
operation by name in backend, it will be very bad user experience if the operation name is too specific. Another reason for choosing more general operation names is to allow the tracing systems to do aggregations.

```
    // Get the registered tracer for this microservice
    tracer := opentracing.Global()
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
    cd.splunkdev.com/libraries/go-observation/opentracing"
)

....
    ctx := context.Background()
    parentSpan := tracer.StartSpan("parent span")

    // create a child span from parent span context
    parentCtx := opentracing.ContextWithSpan(ctx, parentSpan)
    childSpan, err := opentracing.StartSpanFromContext(parentCtx, "child span")
    // Omit error handling for brevity

    // Must do resource clean up.
    defer parentSpan.Finish()
    defer childSpan.Finish()

```

# Tagging a Span
The tags are meant to describe attributes of the span that apply to the whole duration of the span.

```
import (
    "cd.splunkdev.com/libraries/go-observation/opentracing"
)
...

    tracer := opentracing.Global()
    span := tracer.StartSpan("user-registration")
    .... do something else
    span.SetTag("organization","splunk")

```

# Logging events to a span
If you have some event that there is a clear timestamp associated with it within a span, it is a good practice to
log such events in key-value pairs that can be automatically processed by log aggregation systems.

```
    // Get the registered tracer for this microservice
    tracer := opentracing.Global()
    // Start a span with an operation name
    span := tracer.StartSpan("user-registration")
    span.LogKV("event", "start query customer db", "user", "value1")
    .... do database operations
    span.LogKV("event", "finish query customer db", "user", "value1")
    // Must finish span before validation.
    span.Finish()

```

SpanLogger is a convenient struct that allows logging events to both tracer and to the logging library.
It only offers Info and Error since no all levels of logs make sense to sent o tracer backend.

```
    import (
        "cd.splunkdev.com/libraries/go-observation/opentracing"
    )
    ...

    logger := logging.NewWithOutput(serviceName, w)
    logging.SetGlobalLogger(logger)

    span := tracer.StartSpan("a span")
    spanLogger := opentracing.NewSpanLogger(logger, span)

    // The first one will sent to both tracer and logging library
    spanLogger.Info("message 3", "sql statement", "SELECT tenantinfo, subscription from tenant where tenantID=1234")
    // This one only sends to underlying logger form logging library.
    spanLogger.Base.Info("will not send to span", "logger type", "ordinary")

```

# HTTP Client Roundtripper

If for some reason, you cannot use http.Request object but still have access to http.Client (for example swagger gernerated client or
AWS go client(yes you can trace AWS client too)). You can use a roundtripper to inject tracing meta data into outgoing http requests.

```
    import (
        "cd.splunkdev.com/libraries/go-observation/opentracing"
    )
    ...

    span := tracer.StartSpan("topspan").(*mocktracer.MockSpan)
    topSpanContext := opentracing.ContextWithSpan(context.Background(), span)

    httpClient := &http.Client{
        Transport: &Transport{},
    }
    req, err := NewRequest(topSpanContext, "GET", "http://test.biz/tenant1/foo?spantype=old", nil)
    httpClient.Do(req)

    httpClient.Get("http://test.biz/tenant2/foo?spantype=new")

```

# HTTP Middleware

The package provides middleware for automatically create spans on server for incoming request and tagging standard Span tags to current span.

See the [ssc-observation README](cd.splunkdev.com/libraries/go-observation/opentracing) for details on how to configure the middleware pipeline.

You must use the following code to register the middleware:

```go
import (
      kvmetrics "github.com/splunk/kvstore-service/kvstore/metrics"
      "cd.splunkdev.com/libraries/go-observation/opentracing"
)
...
// Simulated microservice A, serving requests.
func Service(hostPort string, wg *sync.WaitGroup) {
    // Configure Route http requests
    http.Handle("/tenant1/operationA",
            openTracing.NewHttpOpentracingHandler(
                http.HandlerFunc(operationAHandler))
   .... enacted ...
}
```


# Running benchmark
From top folder,

```
go test -bench=.  ./opentracing/.

```

With cpu, memory profile and pdf reports:

```
go test -bench=. -cpuprofile=cpu.out -memprofile=mem.out ./opentracing/.

go tool pprof --pdf opentracing.test cpu.out > cpu1.pdf
go tool pprof --pdf opentracing.test mem.out > mem1.pdf

```


You can reference other options by: go test --help


# Observation

 *   Standardization of naming and what information should be part of Span. Recall that Span tags are searchable as Key/Value pairs.

 *   Put burden on engineers to understand what information is important to put on Span for various purposes (Steady Statey architecture improvement or RAC). To decide what is information is important for a Span under the context of that request function, domain expertise may be important here.

 *   For a system of Microservices, the initial impact won't be obvious until you have more and more Microservices also instrumented, the initial investment is high.

*    Engineers will tend to think/treat Distributed tracing as typical logging, which is not. First you don't want to send log everything to a Span (DEBUG log like put URL PATH to lower case, may be too trivial for the whole transaction)

 *   Lots of judgment call. This does not look like a one time thing (think about doing continuous improvement to your architecture, and tracing as part of your experiment data)

 *   Not all vendors are created the same. For LightStep their SDK is strong, integration is a breeze (2 days), despite they are still working on some UX like service map discovery etc. For Instana, I cannot draw the same conclusion. You can tell some javascript guy wrote the go-sdk, and have tried to be too smart (Automagically get default values, but no documented, options available for configuration but you found out they are hard coded in the end)

 *   What is your strategy with this instrumentation data. Now you at a minimal you know the time duration, a high level picture of the request "fulfillment staircase model" down to the cloud provider (AWS client). What should we do about this information? Ongoing data for Architectural improvement experimentation? More intelligent deployment scaling ? Failure pre-emtpive detection? Minimize ownership cost, maximize profit? Compliance violation detection (rogue request) ....


# Deployment

All participating Microservice need to:

 *  be manually instrumented and redeployed.

 *  change of their deployment k8s file to either get the Tracer configuration via config map or env to inject to container environment variables

 *  obtain the appropriate environment variables and if API KEY is required by vendor, need to ensure protection as such

Resources:

[Dapper](https://research.google.com/archive/papers/dapper-2010-1.pdf)

[opentracing large systems](http://opentracing.io/documentation/pages/instrumentation/instrumenting-large-systems.html)

[opentracing spec](http://opentracing.io/documentation/pages/spec)

[opentracing semantic conventions](https://github.com/opentracing/specification/blob/master/semantic_conventions.md)

[opentracing-go godocs](https://godoc.org/github.com/opentracing/opentracing-go).
