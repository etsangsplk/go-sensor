package lightstepx

import (
	"errors"
	"fmt"
	"net"
	"os"
	"strconv"
	"time"

	"cd.splunkdev.com/libraries/go-observation/logging"
)

var options = &Options{}

// Configurations

// Environment Variables
type EnvKey string

// Environment variable keys
const (
	// Feature flag to enabled tracer
	EnvEnabled EnvKey = "LIGHTSTEP_ENABLED"

	// Collector
	EnvURIScheme                      EnvKey = "TRACER_URI_SCHEME"
	EnvCollectorEndpointHostPort      EnvKey = "TRACER_COLLECTOR_HOST_PORT"
	EnvCollectorEndpointSendPlainText EnvKey = "TRACER_COLLECTOR_SEND_PLAINTEXT"

	EnvVerbose EnvKey = "TRACER_VERBOSE"
	// Lightstep
	EnvLightStepAccessToken EnvKey = "LIGHTSTEP_ACCESSTOKEN"

	//LightStep API
	EnvLightStepAPIHostPort      EnvKey = "LIGHTSTEP_API_HOST_PORT"
	EnvLightStepAPISendPlainText EnvKey = "LIGHTSTEP_API_SEND_PLAINTEXT"

	// LightStep reporting settings
	EnvLightStepMaxBufferedSpans   EnvKey = "LIGHTSTEP_MAXBUFFERED_SPANS"
	EnvLightStepMaxLogKeyLen       EnvKey = "LIGHTSTEP_MAX_LOGKEY_LEN"
	EnvLightStepMaxLogValueLen     EnvKey = "LIGHTSTEP_MAX_LOG_VALUE_LEN"
	EnvLightStepMaxLogsPerSpan     EnvKey = "LIGHTSTEP_MAX_LOGS_PER_SPAN"
	EnvLightStepReportingPeriod    EnvKey = "LIGHTSTEP_REPORTING_PERIOD"
	EnvLightStepMinReportingPeriod EnvKey = "LIGHTSTEP_MIN_REPORTING_PERIOD"
	EnvLightStepDropSpanLogs       EnvKey = "LIGHTSTEP_DROP_SPANLOGS"

	// Transport type
	EnvLightStepTransportProtocol EnvKey = "LIGHTSTEP_TRANSPORT_PROTOCOL"
)

// getLightStepTransportProtocol retrieves transportation medium using lightstep api from environment variable.
// Missing value will default using http.
func getLightStepTransportProtocol() string {
	val, ok := getenvTryRequired(EnvLightStepTransportProtocol)
	if !ok {
		return DefaultTransportProtocol
	}
	return val
}

// LoadConfig loads and returns Options
func LoadConfig() error {
	// retrieves lightstep api access token from environment variable. panic if missing.
	// also let LightStep do the validation for us when calling client initialization.
	accessToken := getenvRequired(EnvLightStepAccessToken)

	// retrieves what uri scheme for transport. Example: https or http
	uriScheme := getenvOptionalString(EnvURIScheme, DefaultURIScheme)

	// retrieves collector endpoint host port from environment variable.
	// Missing value will result panic.
	collectorHost, collectorPort, err := getHostPort(EnvCollectorEndpointHostPort)
	if err != nil {
		return err
	}

	// retrieves collector endpoint send plain text setting from environment variable.
	// defaults to true (no encryption).
	collectorSendPlainText := getenvOptionalBool(EnvCollectorEndpointSendPlainText, false)

	// retrieves the maximum number of spans in buffer before trigger a send.
	// 0 signals system to use built-in default.
	maxBufferedSpans := int(getenvOptionalInt64(EnvLightStepMaxBufferedSpans, 0))

	// sets the maximum allowable size (in characters) of an
	// OpenTracing logging key. Longer keys are truncated
	// 0 signals system to use built-in default.
	maxLogKeyLen := int(getenvOptionalInt64(EnvLightStepMaxBufferedSpans, 0))

	// sets the maximum allowable size (in characters) of an
	// OpenTracing logging value. Longer values are truncated. Only applies to
	// variable-length value types (strings, interface{}, etc).
	// 0 signals system to use built-in default.
	maxLogValueLen := int(getenvOptionalInt64(EnvLightStepMaxLogValueLen, 0))

	// sets the limit of the number of logs in a single span.
	// 0 signals system to use built-in default.
	maxLogsPerSpan := int(getenvOptionalInt64(EnvLightStepMaxLogsPerSpan, 0))

	// sets ReportingPeriod is the maximum duration of time between sending spans
	// to a collector.  If zero, the default will be used.
	// 0 signals system to use built-in default.
	reportingPeriod := getenvOptionalTimeDuration(EnvLightStepReportingPeriod, 0)

	// sets MinReportingPeriod is the minimum duration of time between sending spans
	// to a collector.  If zero, the default will be used. It is strongly
	// recommended to use the default.
	// 0 signals system to use built-in default.
	minReportingPeriod := getenvOptionalTimeDuration(EnvLightStepMinReportingPeriod, 0)

	// set turning log events on all Spans into no-ops.
	dropSpanLogs := getenvOptionalBool(EnvLightStepDropSpanLogs, false)
	// TransportProtocol
	transportType := getLightStepTransportProtocol()
	// Verbose logging for tracer
	verbose := getenvOptionalBool(EnvVerbose, false)

	WithVerbose(verbose)(options)
	WithAccessToken(accessToken)(options)
	WithCollectorHost(uriScheme, collectorHost, collectorPort, collectorSendPlainText)(options)
	WithMaxBufferedSpans(maxBufferedSpans)(options)
	WithMaxLogKeyLen(maxLogKeyLen)(options)
	WithMaxLogValueLen(maxLogValueLen)(options)
	WithMaxLogsPerSpan(maxLogsPerSpan)(options)
	WithReportingPeriod(reportingPeriod)(options)
	WithMinReportingPeriod(minReportingPeriod)(options)
	WithDropSpanLogs(dropSpanLogs)(options)
	WithLogRecorder(logging.Global())(options)
	WithTransportProtocol(transportType)(options)

	return nil
}

// Enabled returns true if LIGHTSTEP_ENABLED is set true and
// the required LightStep environment variable is set.
func Enabled() bool {
	enabled := getenvOptionalBool(EnvEnabled, false)
	n := len(getenvOptionalString(EnvLightStepAccessToken, ""))
	return enabled && (n > 0)
}

func getHostPort(key EnvKey) (string, int, error) {
	log := logging.Global()

	value, ok := getenvTryRequired(key)
	if !ok {
		log.Warn("no value for", "envVariable", key, "message", "using lightstep public satellite")
		return "", 0, nil
	}

	host, port, err := net.SplitHostPort(value)
	if err != nil {
		log.Error(errors.New("invalid value"), "invalid value for key", "key", key, "value", value)
		return "", -1, fmt.Errorf("invalid value for key %s", key)
	}
	p, err := strconv.Atoi(port)
	if err != nil {
		log.Error(errors.New("invalid value"), "invalid value for key", "key", key, "value", value)
		return "", -1, fmt.Errorf("invalid value for port %s", key)
	}
	return host, p, nil
}

// Return the environment variable specified by key, or exit
// if the key is not defined
func getenvRequired(key EnvKey) string {
	if v, ok := getenvTryRequired(key); ok {
		return v
	}
	e := errors.New("Configuration error")
	log := logging.Global()
	log.Error(e, "Missing required environment variable",
		"envVariable", key)
	return "" // unreachable
}

func getenvTryRequired(key EnvKey) (string, bool) {
	if v, ok := os.LookupEnv(string(key)); ok {
		return v, true
	}
	return "", false
}

func getenvRequiredBool(key EnvKey) bool {
	result, err := strconv.ParseBool(getenvRequired(key))
	if err != nil {
		log := logging.Global()
		log.Error(err, "Environment variable can't be parsed into bool",
			"envVariable", key)
	}
	return result
}

func getenvOptionalBool(key EnvKey, defaultValue bool) bool {
	v, ok := getenvTryRequired(key)
	if !ok {
		return defaultValue
	}
	result, err := strconv.ParseBool(v)
	if err != nil {
		log := logging.Global()
		log.Error(err, "Environment variable can't be parsed into bool",
			"envVariable", key, "value", v)
	}
	return result
}

func getenvRequiredInt64(key EnvKey) int64 {
	result, err := strconv.ParseInt(getenvRequired(key), 10, 64)
	if err != nil {
		log := logging.Global()
		log.Error(err, "Environment variable can't be parsed into int64",
			"envVariable", key)
	}
	return result
}
func getenvOptionalString(key EnvKey, defaultValue string) string {
	v, ok := getenvTryRequired(key)
	if !ok {
		return defaultValue
	}
	return v
}

func getenvOptionalInt64(key EnvKey, defaultValue int64) int64 {
	v, ok := getenvTryRequired(key)
	if !ok {
		return defaultValue
	}
	result, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		log := logging.Global()
		log.Error(err, "Environment variable can't be parsed into int64",
			"envVariable", key)
	}
	return result
}

func getenvRequiredTimeDuration(key EnvKey) time.Duration {
	result, err := time.ParseDuration(getenvRequired(key))
	if err != nil {
		log := logging.Global()
		log.Error(err, "Environment variable can't be parsed into time duration",
			"envVariable", key)
	}
	return result
}

func getenvOptionalTimeDuration(key EnvKey, defaultValue time.Duration) time.Duration {
	v, ok := getenvTryRequired(key)
	if !ok {
		return defaultValue
	}
	result, err := time.ParseDuration(v)
	if err != nil {
		log := logging.Global()
		log.Error(err, "Environment variable can't be parsed into time.Duration",
			"envVariable", key)
	}
	return result
}
