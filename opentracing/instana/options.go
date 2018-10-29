package instana

import (
	instana "github.com/instana/golang-sensor"
)

type Options struct {
	Opts       instana.Options
	TracerOpts instana.TracerOptions
}

// Option is used to override defaults when creating a new Tracer
type Option func(*Options)

// WithServiceName sets the serivce name for this tracer.
func WithServiceName(serviceName string) Option {
	return func(c *Options) {
		if c != nil {
			c.Opts.Service = serviceName
		}
	}
}

// WithAgentEndpoint sets the agent endpoint for instana agent.
func WithAgentEndpoint(host string, port int) Option {
	return func(c *Options) {
		if c != nil {
			c.Opts.AgentHost = host
			c.Opts.AgentPort = port
		}
	}
}

// WithMaxBufferedSpans sets the maximum number of spans that will be buffered
// before sending them to a collector.
func WithMaxBufferedSpans(maxBufferedSpans int) Option {
	return func(c *Options) {
		if c != nil {
			c.Opts.MaxBufferedSpans = maxBufferedSpans
		}
	}
}

// WithForceTransmissionStartingAt set the number of span logs can be held in buffer
// before sending the logs out. Basically a scheme to empty the log buffer earlier.
func WithForceTransmissionStartingAt(size int) Option {
	return func(c *Options) {
		if c != nil {
			c.Opts.ForceTransmissionStartingAt = size
		}
	}
}

// WithLogLevel set the logging level for span logging.
func WithLogLevel(level int) Option {
	return func(c *Options) {
		if c != nil {
			c.Opts.LogLevel = level
		}
	}
}

// WithMaxLogsPerSpan sets limits the number of logs in a single span.
func WithMaxLogsPerSpan(maxLogsPerSpan int) Option {
	return func(c *Options) {
		if c != nil {
			c.TracerOpts.MaxLogsPerSpan = maxLogsPerSpan
		}
	}
}

// WithDropAllLogs sets to turn log events on all Spans into no-ops.
func WithDropAllLogs(dropAllLogs bool) Option {
	return func(c *Options) {
		if c != nil {
			c.TracerOpts.DropAllLogs = dropAllLogs
		}
	}
}

// WithDebugAssertSingleGoroutine (debugging purposes only) sets tags to flag when Spans to
// passed across goroutine boundaries.
func WithDebugAssertSingleGoroutine(assertSingleGoroutine bool) Option {
	return func(c *Options) {
		if c != nil {
			c.TracerOpts.DebugAssertSingleGoroutine = assertSingleGoroutine
		}
	}
}

// WithAssertUseAfterFinish, strickly for aiding debugging, exacerbate issues emanating from use of Spans
// after calling Finish by running additional assertions.
func WithDebugAssertUseAfterFinish(assertUseAfterFinish bool) Option {
	return func(c *Options) {
		if c != nil {
			c.TracerOpts.DebugAssertUseAfterFinish = assertUseAfterFinish
		}
	}
}

// WithEnableSpanPool sspan buffer pooling for slight performance gain.
func WithEnableSpanPool(enableSpanPool bool) Option {
	return func(c *Options) {
		if c != nil {
			c.TracerOpts.EnableSpanPool = enableSpanPool
		}
	}
}
