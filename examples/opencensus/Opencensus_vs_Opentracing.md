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

# Reference
Opencensus concepts
(https://opencensus.io/core-concepts)

NullExporter
(examples/opencenus/exporter/nullexporter.go)

Opencensus Repo
(https://github.com/census-instrumentation/opencensus-go)


