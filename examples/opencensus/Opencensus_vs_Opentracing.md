# Summary
Neither Lightstep nor Instana supports Opencensus natively, we need to write a Exporter for them for sending data to their backend. Specically, stats and trace exporter. There are only 2 methods to implement for such exporter (see null exporter), but the broader question is that if there is any need for datastructure translation:

**  Trace
**  Span
**  Span Tag
**  Span Baggage (not seeing this in opencensus)
**  Span Log 

Example NullExporter:
```
import (
    "go.opencensus.io/stats/view"
    "go.opencensus.io/trace"
)

// NullExporter is a stats and trace exporter that does nothing
type NullExporter struct{}

// stats.view.Exporter interface
func (e *NullExporter) ExportView(d *view.Data) {
    // noop
}

// trace.Exporter interface
func (e *NullExporter) ExportSpan(d *trace.SpanData) {
    // noop
}
```
# Reference

Opencensus Repo
(https://github.com/census-instrumentation/opencensus-go)


