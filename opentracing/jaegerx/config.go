package jaegerx

import (
	"errors"
	"os"
	"strconv"
	"time"

	"cd.splunkdev.com/libraries/go-observation/logging"
)

// Configurations

// Environment Variables
type EnvKey string

// Environment variable keys
// Let jaeger config_env take care of all the configuration work.
const (
	EnvJaegerDisabled  EnvKey = "JAEGER_DISABLED"
	EnvJaegerAgentHost EnvKey = "JAEGER_AGENT_HOST"
)

// Enabled returns true if JAEGER_ENABLED is set true and
// JAEGER_AGENT_HOST is set.
func Enabled() bool {
	enabled := getenvOptionalBool(EnvJaegerDisabled, true)
	n := len(getenvOptionalString(EnvJaegerAgentHost, ""))
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
