package logging

import (
	"errors"
	"io/ioutil"
	"strings"
	"testing"
)

// Provided solely as a relative value to compare against
func BenchmarkStringsRepeatBaseline(b *testing.B) {
	b.ResetTimer()
	b.ReportAllocs()
	for n := 0; n < b.N; n++ {
		strings.Repeat("LongMessage", 4096)
	}
	return
}

func BenchmarkShortMessage(b *testing.B) {
	logger := NewWithOutput("log with short message", ioutil.Discard)
	b.ResetTimer()
	b.ReportAllocs()
	for n := 0; n < b.N; n++ {
		logger.Info("event", "ShortMessage")
	}
	return
}

func BenchmarkLongMessage(b *testing.B) {
	logger := NewWithOutput("log with long message", ioutil.Discard)
	b.ResetTimer()
	b.ReportAllocs()
	for n := 0; n < b.N; n++ {
		logger.Info("event", strings.Repeat("LongMessage", 4096))
	}
	return
}

func BenchmarkShortErrorMessage(b *testing.B) {
	logger := NewWithOutput("log with short message", ioutil.Discard)
	b.ResetTimer()
	b.ReportAllocs()
	for n := 0; n < b.N; n++ {
		logger.Error(errors.New("ShortMessage"), "")
	}
	return
}

func BenchmarkLongErrorMessage(b *testing.B) {
	logger := NewWithOutput("log with long message", ioutil.Discard)
	b.ResetTimer()
	b.ReportAllocs()
	for n := 0; n < b.N; n++ {
		logger.Error(errors.New(strings.Repeat("LongMessage", 4096)), "")
	}
	return
}
