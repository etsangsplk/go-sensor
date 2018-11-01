# Summary
Neither Lightstep nor Instana supports Opencensus natively, we need to write a Exporter for them for sending data to their backend. Specically, stats and trace exporter. There are only 2 methods to implement for such exporter (see null exporter), but the broader question is that if there is any need for datastructure translation:

**  Trace
**  Span
**  Span Tag
**  Span Baggage (not seeing this in opencensus)
**  Span Log 

Span Log in Opentracing maps to Opencensus trace Annotation, message payload
https://github.com/census-instrumentation/opencensus-go/blob/264a2a48d94c062252389fffbc308ba555e35166/trace/export.go#L71:2
But does it do

Issue1 HTTP Request
openencensus api does not say anything about the http request, they only provide interface methods.
```
    SpanContextFromRequest(req *http.Request) (sc trace.SpanContext, ok bool)
    SpanContextToRequest(sc trace.SpanContext, req *http.Request)
```
location. trace/propagation/propagation.go
opentracing also just touches http request. But the spec alo defines a set of standard tag key names.
so far not seeing that from opencensus.
Same goes with Database/Queueing...

Since tag key is just string, we can import opentracing tag and use their exposed constants but why ...

Issue2 SpanTag settings
There are lack of convenience functions for Span Tag settings that help avoid mistakes
Note that the following is jus for reference, http is actually taken care of by plugin:
https://github.com/census-instrumentation/opencensus-go/tree/master/plugin/ochttp
For example, no database nor pubsub.
But note that we may not need to write out own(or just enhance) their roudntripper or inject meta data into http request.
We still need to take care of injecting out own meta data like tenantID and requestID.

``` 
    opentracing
    tag.HTTPMethod.Set(span, r.Method)
    tag.HTTPUrl.Set(span, r.URL.String())
```

```
    opencensus
    span.SetStatus(trace.Status{trace.Code: trace.StatusCodeUnknown, trace.Message: err.Error()})

```

Setting Span Status, is a bit harder, requires more understanding. 
Check out their status code:
https://github.com/census-instrumentation/opencensus-go/blob/264a2a48d94c062252389fffbc308ba555e35166/trace/status_codes.go
https://github.com/googleapis/googleapis/blob/master/google/rpc/code.proto

But luckily
https://github.com/census-instrumentation/opencensus-go/tree/aa2b39d1618ef56ba156f27cfcdae9042f68f0bc/plugin

# Reference
Opencensus concepts
(https://opencensus.io/core-concepts)

Opencensus plugin (http and grpc for now)
(https://github.com/census-instrumentation/opencensus-go/tree/master/plugin)

NullExporter
(examples/opencenus/exporter/nullexporter.go)

Opencensus Repo
(https://github.com/census-instrumentation/opencensus-go)


