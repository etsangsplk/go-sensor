package instanax

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"cd.splunkdev.com/libraries/go-observation/logging"
	"cd.splunkdev.com/libraries/go-observation/opentracing"
	"cd.splunkdev.com/libraries/go-observation/opentracing/testutil"
)

func TestNewTracerInitialization(t *testing.T) {
	outC, w := testutil.StartLogCapturing()
	logger := logging.NewWithOutput("test new tracer initialization", w)
	logging.SetGlobalLogger(logger)

	g := testutil.GetGlobalTracer()
	defer testutil.RestoreGlobalTracer(g)

	env := testutil.StashEnv()
	defer testutil.PopEnv(env)

	os.Setenv(string(EnvInstanaAgentHost), "saas-us-west-2.instana.io")
	os.Setenv(string(EnvInstanaAgentPort), "443")

	tracer := NewTracer("test init ok")
	opentracing.SetGlobalTracer(tracer)

	s := testutil.StopLogCapturing(outC, w)
	assert.NotNil(t, tracer)
	// OMG. Instana provides no validation on agent host/port validation
	// It is not part of Tracer either but within "sensor" which has no
	// public api to validate.
	assert.NotContains(t, s[0], "\"error\":\"Configuration error\"")
}

func TestNewTracerInitializationMissingRequiredEnv(t *testing.T) {
	outC, w := testutil.StartLogCapturing()
	logger := logging.NewWithOutput("test new tracer initialization", w)
	logging.SetGlobalLogger(logger)
	g := testutil.GetGlobalTracer()
	defer testutil.RestoreGlobalTracer(g)

	env := testutil.StashEnv()
	defer testutil.PopEnv(env)

	os.Setenv(string(EnvInstanaAgentHost), "127.0.0.1:")

	tracer := NewTracer("test missing required env")
	opentracing.SetGlobalTracer(tracer)

	s := testutil.StopLogCapturing(outC, w)
	assert.NotNil(t, tracer)
	// OMG. Instana provides no validation on agent host/port validation
	// It is not part of Tracer either but within "sensor" which has no
	// public api to validate.
	assert.Contains(t, s[0], "\"error\":\"Configuration error\"")
}
