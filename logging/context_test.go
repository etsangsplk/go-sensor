package logging

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestContext(t *testing.T) {
	outC, w := StartLogCapturing()

	logger := NewWithOutput("testContextLogger", w)
	logger.Info("message0")
	ctx := NewContext(context.Background(), logger, "field1", "value1")
	From(ctx).Info("message1")
	ctx = NewComponentContext(ctx, "component1", "field2", "value2")
	From(ctx).Info("message2")

	// Test with empty logger
	nilCtx := NewContext(ctx, nil)
	nilLogger := From(nilCtx)
	assert.Equal(t, Global(), nilLogger)

	s := StopLogCapturing(outC, w)
	assert.Contains(t, s[0], "message0")
	assert.Contains(t, s[1], "message1")
	assert.Contains(t, s[1], `"field1":"value1"`)
	assert.Contains(t, s[2], `"field2":"value2"`)
	assert.Contains(t, s[2], `"component":"component1"`)
}

