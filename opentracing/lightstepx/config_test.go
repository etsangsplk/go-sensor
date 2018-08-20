package lightstepx

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"cd.splunkdev.com/libraries/go-observation/logging"
)

func TestNewTracerInitialization(t *testing.T) {
	outC, w := StartLogCapturing()
	logger := logging.NewWithOutput("test new tracer initialization", w)
	logging.SetGlobalLogger(logger)

	env := StashEnv()
	defer PopEnv(env)

	os.Setenv(string(EnvCollectorEndpointHostPort), "127.0.0.1:8080")
	os.Setenv(string(EnvCollectorEndpointSendPlainText), "true")
	os.Setenv(string(EnvLightStepAPIHostPort), "127.0.0.5:8083")
	os.Setenv(string(EnvLightStepAPISendPlainText), "false")
	os.Setenv(string(EnvLightStepAccessToken), "2ece7a94e599cefd32050ed38d337f9f")

	tracer := NewTracer("test init")
	StopLogCapturing(outC, w)
	assert.NotNil(t, tracer)
}

func TestNewTracerInitializationAccessTokenError(t *testing.T) {
	outC, w := StartLogCapturing()
	logger := logging.NewWithOutput("test new tracer initialization access token error", w)
	logging.SetGlobalLogger(logger)

	env := StashEnv()
	defer PopEnv(env)
	os.Setenv(string(EnvCollectorEndpointHostPort), "127.0.0.1:8080")
	os.Setenv(string(EnvCollectorEndpointSendPlainText), "true")
	os.Setenv(string(EnvLightStepAPIHostPort), "127.0.0.1:8080")
	os.Setenv(string(EnvLightStepAPISendPlainText), "false")

	tracer := NewTracer("test init with access token error")
	s := StopLogCapturing(outC, w)
	assert.NotNil(t, tracer) // global Noop tracer
	assert.Contains(t, s[0], `"envVariable":"LIGHTSTEP_ACCESSTOKEN"`)
	assert.Contains(t, s[0], `"error":"Configuration error"`)
}

func TestNewTracerInitializationCollectorEndpointError(t *testing.T) {
	outC, w := StartLogCapturing()
	logger := logging.NewWithOutput("test new tracer initialization collector endpoint error", w)
	logging.SetGlobalLogger(logger)

	env := StashEnv()
	defer PopEnv(env)
	os.Setenv(string(EnvLightStepAccessToken), "test")
	os.Setenv(string(EnvCollectorEndpointSendPlainText), "true")
	os.Setenv(string(EnvLightStepAPIHostPort), "127.0.0.1:8080")
	os.Setenv(string(EnvLightStepAPISendPlainText), "false")
	tracer := NewTracer("test")
	s := StopLogCapturing(outC, w)

	assert.NotNil(t, tracer) // global Noop tracer
	assert.Contains(t, s[0], `"envVariable":"TRACER_COLLECTOR_HOST_PORT"`)
	assert.Contains(t, s[0], `"message":"using lightstep public satellite"`)
}

func TestGetSettingFromEnvironment(t *testing.T) {
	env := StashEnv()
	defer PopEnv(env)
	os.Setenv(string(EnvLightStepAPIHostPort), "UseHTTP")
	val := getLightStepTransportProtocol()

	assert.Equal(t, "UseGRPC", val)
}

// StashEnv stashes the current environment variables and returns an array of
// all environment values as key=val strings.
func StashEnv() []string {
	// Rip https://github.com/aws/aws-sdk-go/awstesting/util.go
	env := os.Environ()
	os.Clearenv()
	return env
}

// PopEnv takes the list of the environment values and injects them into the
// process's environment variable data. Clears any existing environment values
// that may already exist.
func PopEnv(env []string) {
	// Rip https://github.com/aws/aws-sdk-go/awstesting/util.go
	os.Clearenv()
	for _, e := range env {
		p := strings.SplitN(e, "=", 2)
		k, v := p[0], ""
		if len(p) > 1 {
			v = p[1]
		}
		os.Setenv(k, v)
	}
}
