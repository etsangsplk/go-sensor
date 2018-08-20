# Table of Contents


Opentracing is a vendor-neutral open standard for distributed tracing. If we trace our method calls via OpenTracing APIs, we can swap out our tracing vendors.

Distributed tracing attempts to help provide developers with more information about the behaviour of complex distributed systems. When an individual request traverses dozens or more systems with numerous components on each system, looking at logs emitted by each system in isolation assume that engineers have expert knowledge of the system end-to-end, system is in sync with what's in engineers mind, the dependencies are relatively easy to understand even it is a graph of dependecies.

Opentracing 
## Terminology and core concepts

See opentracing spec for mor information.

* _Tracer: Tracer interface creates Spans and understands how to Inject (serialize) and Extract (deserialize) them across process boundaries. Opentracing does not provide concrete implementation of Tracer, which is up to vendor.

* _Span: Spans are logical units of work in a distributed system, and by definition they all have a name, a start time, and a duration. In a trace, Spans are associated with the distributed system component that generated them. See opentracing spec for mor information.

* _Relationship between Spans: Relationships are the connections between Spans. A Span may reference zero or more other Spans that are causally related. These connections between Spans help describe the semantics of the running system, as well as the critical path for latency-sensitive (distributed) transactions. Relationship are wired together by SpanContext.

* _SpanContext The SpanContext is more of a "concept" than a useful piece of functionality at the generic OpenTracing layer. Most OpenTracing users only interact with SpanContext via references when starting new Spans, or when injecting/extracting a trace to/from some transport protocol.

* _Span_Tag OpenTracing project documents certain "standard tags" that have prescribed semantic meanings.

* _Span_Log OpenTracing project documents certain "standard log keys" which have prescribed semantic meanings.

OpenTracing API does not dictate how we do it; the general principle is that information that applies to the span as a whole should be recorded as a tag, while events that have timestamps should be recorded as logs

Resouces:
[Dapper]: https://research.google.com/archive/papers/dapper-2010-1.pdf
[opentracing large systems]: http://opentracing.io/documentation/pages/instrumentation/instrumenting-large-systems.html
[opentracing spec]: http://opentracing.io/documentation/pages/spec
[opentracing semantic conventions]: https://github.com/opentracing/specification/blob/master/semantic_conventions.md
[opentracing-go
godocs](https://godoc.org/github.com/opentracing/opentracing-go).
