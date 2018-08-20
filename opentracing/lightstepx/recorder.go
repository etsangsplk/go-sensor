package lightstepx

import (
	lightstep "github.com/lightstep/lightstep-tracer-go"

	"cd.splunkdev.com/libraries/go-observation/logging"
)

// LoggingRecorder is a Recorder implementation for lightstep Recorder. It logs
// span information using ssc logger. This will be useful to correlate
// log events in splunk backend and those in Lghtstep.
type LoggingRecorder struct {
	lightstep.SpanRecorder
	Logger *logging.Logger
}

// RecordSpan will record extracted span information as log entries.
func (r *LoggingRecorder) RecordSpan(span lightstep.RawSpan) {
	fieldBuilder := logging.FieldsBuilder{}
	fieldBuilder.AddFields("traceID", span.Context.TraceID,
		"spanID", span.Context.SpanID,
		"parentSpanID", span.ParentSpanID,
		"operation", span.Operation)
	fieldBuilder.AddFields("Start", span.Start,
		"duration", span.Duration)
	fieldBuilder.FlattenMapInterface(span.Tags)
	r.Logger.Info("span info", fieldBuilder.Fields()...)
}
