# Summary
Neither Lightstep nor Instana supports Opencensus natively, we need to write a Exporter for them for sending data to their backend. Specically, stats and trace exporter. There are only 2 methods to implement for such exporter (see null exporter), but the broader question is that if there is any need for datastructure translation:

**  Trace
**  Span
**  Span Tag
**  Span Baggage (not seeing this in opencensus)
**  Span Log 


# Reference
NullExporter
(examples/opencenus/exporter/nullexporter.go)

Opencensus Repo
(https://github.com/census-instrumentation/opencensus-go)


