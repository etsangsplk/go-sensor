package exporter

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

// For our vendor tracer (lightstep/jaeger) etc.
func (e *NullExporter) Flush() {}

func (e *NullExporter) Close() {}
