package opentracing

import (
	"io/ioutil"
	"strings"
	"testing"

	"cd.splunkdev.com/libraries/go-observation/logging"
)

// Provided solely as a relative value to compare against
func BenchmarkStringsRepeatBaseline(b *testing.B) {
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		strings.Repeat("LongMessage", 4096)
	}
	return
}

func BenchmarkShortMessage(b *testing.B) {
	logger := logging.NewWithOutput("testlogger", ioutil.Discard)
	tracer, closer := NewTracer("test new tracer", logger)
	defer closer.Close()
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		span := tracer.StartSpan("operation")
		span.LogKV("event", "ShortMessage")
		span.Finish()
	}
	return
}

func BenchmarkLongMessage(b *testing.B) {
	logger := logging.NewWithOutput("testlogger", ioutil.Discard)
	tracer, closer := NewTracer("test new tracer", logger)
	defer closer.Close()
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		span := tracer.StartSpan("operation")
		span.LogKV("event", strings.Repeat("LongMessage", 4096))
		span.Finish()
	}
	return
}
