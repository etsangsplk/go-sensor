package instana

import (
	"errors"
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
	EnvEnabled EnvKey = "INSTANA_ENABLED"

	// Agent
	EnvInstanaAgentHost EnvKey = "INSTANA_AGENT_HOST" // go-sensor separate this inside their fsm.go
	EnvInstanaAgentPort EnvKey = "INSTANA_AGENT_PORT"

	// LightStep reporting settings
	EnvInstanaMaxBufferedSpans EnvKey = "INSTANA_MAXBUFFERED_SPANS"

	//EnvInstanaMaxLogKeyLen       EnvKey = "INSTANA_MAX_LOGKEY_LEN"
	//EnvInstanaMaxLogValueLen     EnvKey = "INSTANA_MAX_LOG_VALUE_LEN"
	EnvInstanaMaxLogsPerSpan     EnvKey = "INSTANA_MAX_LOGS_PER_SPAN"
	EnvInstanaReportingPeriod    EnvKey = "INSTANA_REPORTING_PERIOD"
	EnvInstanaMinReportingPeriod EnvKey = "INSTANA_MIN_REPORTING_PERIOD"

	EnvInstanaDropAllSpanLogs    EnvKey = "INSTANA_DROP_ALL_SPANLOGS"
	EnvInstanaTrimUnsampledSpans EnvKey = "INSTANA_TRIM_UNSAMPLED_LOGS"

	EnvInstanaForceTransmissionStartingAt EnvKey = "INSTANA_FORCED_TRANSMISSION_AT"
	EnvInstanaLogLevel                    EnvKey = "INSTANA_LOG_LEVEL"

	DebugAssertSingleGoroutine EnvKey = "INSTANA_ENABLE_DEBUG_ASSERT_SINGLE_GOROUTINE"

	DebugAssertUseAfterFinish EnvKey = "INSTANA_ENABLE_DEBUG_USE_AFTER_FINISH"

	EnableSpanPool EnvKey = "INSTANA_ENABLE_SPAN_POOL"
)

// LoadConfig loads and returns Options
func LoadConfig() error {
	agentHost := getenvRequired(EnvInstanaAgentHost)
	agentPort := int(getenvRequiredInt64(EnvInstanaAgentPort))

	// retrieves the maximum number of spans in buffer before trigger a send.
	// 0 signals system to use built-in default.
	maxBufferedSpans := int(getenvOptionalInt64(EnvInstanaMaxBufferedSpans, 0))

	// sets the limit of the number of logs in a single span.
	// 0 signals system to use built-in default.
	maxLogsPerSpan := int(getenvOptionalInt64(EnvInstanaMaxLogsPerSpan, 0))

	// force to spend log once number of span reaches this level.
	forceTransmissionStartingAt := int(getenvOptionalInt64(EnvInstanaForceTransmissionStartingAt, 0))

	// Minimal log level to send logs out Error = 0 .. Debug = 3
	logLevel := int(getenvOptionalInt64(EnvInstanaLogLevel, 2))

	// set turning log events on all Spans into no-ops.
	dropSpanLogs := getenvOptionalBool(EnvInstanaDropAllSpanLogs, false)

	WithAgentEndpoint(agentHost, agentPort)(options)
	WithMaxBufferedSpans(maxBufferedSpans)(options)
	WithForceTransmissionStartingAt(forceTransmissionStartingAt)(options)
	WithMaxLogsPerSpan(maxLogsPerSpan)(options)
	WithLogLevel(logLevel)(options)
	WithDropAllLogs(dropSpanLogs)(options)

	return nil
}

// Enabled returns true if INSTANA_ENABLED is set true and
// INSTANA_AGENT_HOST is set.
func Enabled() bool {
	enabled := getenvOptionalBool(EnvEnabled, false)
	n := len(getenvOptionalString(EnvInstanaAgentHost, ""))
	return enabled && (n > 0)
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
