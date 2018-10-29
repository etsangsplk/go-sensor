package instana

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"cd.splunkdev.com/libraries/go-observation/logging"
	"cd.splunkdev.com/libraries/go-observation/opentracing"
)

func TestNewTracerInitialization(t *testing.T) {
	outC, w := StartLogCapturing()
	logger := logging.NewWithOutput("test new tracer initialization", w)
	logging.SetGlobalLogger(logger)

	g := SaveGlobalTracer()
	defer RestoreGlobalTracer(g)

	env := StashEnv()
	defer PopEnv(env)

	os.Setenv(string(EnvInstanaAgentHost), "saas-us-west-2.instana.io")
	os.Setenv(string(EnvInstanaAgentPort), "443")

	tracer := NewTracer("test init ok")
	opentracing.SetGlobalTracer(tracer)

	s := StopLogCapturing(outC, w)
	assert.NotNil(t, tracer)
	// OMG. Instana provides no validation on agent host/port validation
	// It is not part of Tracer either but within "sensor" which has no
	// public api to validate.
	assert.NotContains(t, s[0], "\"error\":\"Configuration error\"")
}

func TestNewTracerInitializationMissingRequiredEnv(t *testing.T) {
	outC, w := StartLogCapturing()
	logger := logging.NewWithOutput("test new tracer initialization", w)
	logging.SetGlobalLogger(logger)
	g := SaveGlobalTracer()
	defer RestoreGlobalTracer(g)

	env := StashEnv()
	defer PopEnv(env)

	os.Setenv(string(EnvInstanaAgentHost), "127.0.0.1:")

	tracer := NewTracer("test missing required env")
	opentracing.SetGlobalTracer(tracer)

	s := StopLogCapturing(outC, w)
	assert.NotNil(t, tracer)
	// OMG. Instana provides no validation on agent host/port validation
	// It is not part of Tracer either but within "sensor" which has no
	// public api to validate.
	assert.Contains(t, s[0], "\"error\":\"Configuration error\"")
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
