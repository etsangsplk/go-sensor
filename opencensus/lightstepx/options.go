package lightstepx

import (
	"strings"
	"time"

	lightstep "github.com/lightstep/lightstep-tracer-go"
	opentracing "github.com/opentracing/opentracing-go"

	"cd.splunkdev.com/libraries/go-observation/logging"
)

type Options lightstep.Options

// Option is used to override defaults when creating a new Tracer.
type Option func(*Options)

// WithVerbose sets tracer to verbose logging.
func WithVerbose(verbose bool) Option {
	return func(c *Options) {
		if c != nil {
			c.Verbose = verbose
		}
	}
}

// WithAccessToken sets the LightStep project API key.
// AccessToken is the unique API key for your LightStep project.
func WithAccessToken(accessToken string) Option {
	return func(c *Options) {
		if c != nil {
			c.AccessToken = accessToken
		}
	}
}

// WithCollectorHost sets the host, port, and plaintext option to use
// for the collector.
func WithCollectorHost(scheme string, collectorHost string, collectorPort int, collectorSendPlainText bool) Option {
	return func(c *Options) {
		if c != nil {
			c.Collector = lightstep.Endpoint{
				Scheme:    scheme,
				Host:      collectorHost,
				Port:      collectorPort,
				Plaintext: collectorSendPlainText,
			}
		}
	}
}

// WithLightStepAPI sets the the host, port, and plaintext option to use
// for the LightStep web API.
func WithLightStepAPI(collectorHost string, collectorPort int, collectorSendPlainText bool) Option {
	return func(c *Options) {
		if c != nil {
			c.LightStepAPI = lightstep.Endpoint{
				Host:      collectorHost,
				Port:      collectorPort,
				Plaintext: collectorSendPlainText,
			}
		}
	}
}

// WithMaxBufferedSpans sets the maximum number of spans that will be buffered
// before sending them to a collector.
func WithMaxBufferedSpans(maxBufferedSpans int) Option {
	return func(c *Options) {
		if c != nil {
			c.MaxBufferedSpans = maxBufferedSpans
		}
	}
}

// WithMaxLogKeyLen set the maximum allowable size (in characters) of an
// OpenTracing logging key. Longer keys are truncated.
func WithMaxLogKeyLen(maxLogKeyLen int) Option {
	return func(c *Options) {
		if c != nil {
			c.MaxLogKeyLen = maxLogKeyLen
		}
	}
}

// WithMaxLogValueLen set the maximum allowable size (in characters) of an
// OpenTracing logging value. Longer values are truncated. Only applies to
// variable-length value types (strings, interface{}, etc).
func WithMaxLogValueLen(maxLogValueLen int) Option {
	return func(c *Options) {
		if c != nil {
			c.MaxLogValueLen = maxLogValueLen
		}
	}
}

// WithMaxLogsPerSpan sets limits the number of logs in a single span.
func WithMaxLogsPerSpan(maxLogsPerSpan int) Option {
	return func(c *Options) {
		if c != nil {
			c.MaxLogsPerSpan = maxLogsPerSpan
		}
	}
}

// WithReportingPeriod sets the maximum duration of time between sending spans
// to a collector.  If zero, the default will be used.
func WithReportingPeriod(reportingPeriod time.Duration) Option {
	return func(c *Options) {
		if c != nil {
			c.ReportingPeriod = reportingPeriod
		}
	}
}

// WithMinReportingPeriod sets the the minimum duration of time between sending spans
// to a collector.  If value is 0, the default will be used. It is strongly
// recommended to use the default.
func WithMinReportingPeriod(minReportingPeriod time.Duration) Option {
	return func(c *Options) {
		if c != nil {
			c.MinReportingPeriod = minReportingPeriod
		}
	}
}

// WithDropSpanLogs sets to turn log events on all Spans into no-ops.
func WithDropSpanLogs(dropSpanLogs bool) Option {
	return func(c *Options) {
		if c != nil {
			c.DropSpanLogs = dropSpanLogs
		}
	}
}

// WithLogRecorder sets the recorder to use logger.
func WithLogRecorder(l *logging.Logger) Option {
	logRecorder := &LoggingRecorder{Logger: l}
	return withCustomRecorder(logRecorder)
}

// WithTags sets tags when initializing tracer.
func WithTags(t opentracing.Tags) Option {
	return func(c *Options) {
		if c != nil {
			c.Tags = t
		}
	}
}

// withCustomRecorder sets the recorder to use logger.
func withCustomRecorder(r lightstep.SpanRecorder) Option {
	return func(c *Options) {
		if c != nil {
			c.Recorder = r
		}
	}
}

// WithConnection intercepts the outgoing connection from lightstep client.
// This funciton is only used in testing.
func WithConnection(fakeConnection lightstep.ConnectorFactory) Option {
	return func(c *Options) {
		if c != nil {
			c.ConnFactory = fakeConnection
		}
	}
}

// WithTransportProtocol sets a specific transport protocol. If multiple are set to true,
// the following order is used to select for the first option: thrift, http, grpc.
// If none are set to true, HTTP is defaulted to.
func WithTransportProtocol(transportType string) Option {
	return func(c *Options) {
		switch strings.ToLower(transportType) {
		case "usethrift":
			c.UseThrift = true
		case "usegrpc":
			c.UseGRPC = true
		default:
			c.UseHttp = true
		}
	}
}
