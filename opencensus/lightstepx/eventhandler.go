package lightstepx

import (
	"reflect"

	"cd.splunkdev.com/libraries/go-observation/logging"
	lightstep "github.com/lightstep/lightstep-tracer-go"
)

// logTracerEventHandler connects lightstep observability events to logging
func logTracerEventHandler(event lightstep.Event) {
	logger := logging.Global()
	switch event := event.(type) {
	case lightstep.EventStatusReport:
		if options.Verbose {
			logger.Debug("LightStep status report", "status", event.String())
		}
	case lightstep.EventConnectionError:
		logger.Warn("LightStep connection error", logging.ErrorKey, event.Err())
	case lightstep.EventStartError:
		logger.Warn("LightStep start error", logging.ErrorKey, event.Err())
	case lightstep.ErrorEvent:
		logger.Warn("LightStep error", logging.ErrorKey, event.Err())
	default:
		logger.Warn("LightStep unknown event", "event", event.String(), "type", reflect.TypeOf(event))
	}
}
