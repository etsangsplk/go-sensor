package lightstepx

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"cd.splunkdev.com/libraries/go-observation/logging"
	"cd.splunkdev.com/libraries/go-observation/opentracing/testutil"
)

func TestNewTracerInitialization(t *testing.T) {
	outC, w := testutil.StartLogCapturing()
	logger := logging.NewWithOutput("test new tracer initialization", w)
	logging.SetGlobalLogger(logger)

	env := testutil.StashEnv()
	defer testutil.PopEnv(env)

	os.Setenv(string(EnvCollectorEndpointHostPort), "127.0.0.1:8080")
	os.Setenv(string(EnvCollectorEndpointSendPlainText), "true")
	os.Setenv(string(EnvLightStepAPIHostPort), "127.0.0.5:8083")
	os.Setenv(string(EnvLightStepAPISendPlainText), "false")
	os.Setenv(string(EnvLightStepAccessToken), "2ece7a94e599cefd32050ed38d337f9f")

	tracer := NewTracer("test init")
	testutil.StopLogCapturing(outC, w)
	assert.NotNil(t, tracer)
}

func TestNewTracerInitializationAccessTokenError(t *testing.T) {
	outC, w := testutil.StartLogCapturing()
	logger := logging.NewWithOutput("test new tracer initialization access token error", w)
	logging.SetGlobalLogger(logger)

	env := testutil.StashEnv()
	defer testutil.PopEnv(env)
	os.Setenv(string(EnvCollectorEndpointHostPort), "127.0.0.1:8080")
	os.Setenv(string(EnvCollectorEndpointSendPlainText), "true")
	os.Setenv(string(EnvLightStepAPIHostPort), "127.0.0.1:8080")
	os.Setenv(string(EnvLightStepAPISendPlainText), "false")

	tracer := NewTracer("test init with access token error")
	s := testutil.StopLogCapturing(outC, w)
	assert.NotNil(t, tracer) // global Noop tracer
	assert.Contains(t, s[0], `"envVariable":"LIGHTSTEP_ACCESSTOKEN"`)
	assert.Contains(t, s[0], `"error":"Configuration error"`)
}

func TestNewTracerInitializationCollectorEndpointError(t *testing.T) {
	outC, w := testutil.StartLogCapturing()
	logger := logging.NewWithOutput("test new tracer initialization collector endpoint error", w)
	logging.SetGlobalLogger(logger)

	env := testutil.StashEnv()
	defer testutil.PopEnv(env)
	os.Setenv(string(EnvLightStepAccessToken), "test")
	os.Setenv(string(EnvCollectorEndpointSendPlainText), "true")
	os.Setenv(string(EnvLightStepAPIHostPort), "127.0.0.1:8080")
	os.Setenv(string(EnvLightStepAPISendPlainText), "false")
	tracer := NewTracer("test")
	s := testutil.StopLogCapturing(outC, w)

	assert.NotNil(t, tracer) // global Noop tracer
	assert.Contains(t, s[0], `"envVariable":"TRACER_COLLECTOR_HOST_PORT"`)
	assert.Contains(t, s[0], `"message":"using lightstep public satellite"`)
}

func TestGetSettingFromEnvironment(t *testing.T) {
	env := testutil.StashEnv()
	defer testutil.PopEnv(env)
	os.Setenv(string(EnvLightStepAPIHostPort), "UseHTTP")
	val := getLightStepTransportProtocol()

	assert.Equal(t, "UseGRPC", val)
}
