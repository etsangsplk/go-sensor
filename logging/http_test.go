package logging

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"cd.splunkdev.com/libraries/go-observation/tracing"
	"github.com/stretchr/testify/assert"
)

// Test for just the global panic handler
func TestPanicHandler(t *testing.T) {
	var e error
	var h http.Handler
	h = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("game over")
	})
	h = NewPanicHandler(h)
	r := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)

	assert.Equal(t, w.Code, 500, "Status code not correct")

	var result struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}
	if e = json.Unmarshal(w.Body.Bytes(), &result); e != nil {
		t.Fatal(e)
	}
	assert.Equal(t, result.Code, 500, "Result code not correct")
	assert.NotEmpty(t, result.Message, "Result message empty")
}

// Test for the panic request handler and the logger handler.
func TestPanicRequestHandler(t *testing.T) {
	var e error
	var h http.Handler
	h = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("game over")
	})

	outC, logWriter := StartLogCapturing()
	logger := NewWithOutput("TestRequestPanicHandler", logWriter)
	h = NewPanicHandler(
		tracing.NewRequestContextHandler(
			NewRequestLoggerHandler(logger,
				NewPanicRequestHandler(h))))
	r := httptest.NewRequest("GET", "/tenant1/foo", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	s := StopLogCapturing(outC, logWriter)

	assert.Equal(t, w.Code, 500, "Status code not correct")

	var result struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}
	if e = json.Unmarshal(w.Body.Bytes(), &result); e != nil {
		t.Fatal(e)
	}
	assert.Equal(t, result.Code, 500, "Result code not correct")
	assert.NotEmpty(t, result.Message, "Result message empty")

	assert.Contains(t, s[0], "requestId")
	assert.Contains(t, s[0], `"tenant":"tenant1"`)
}

func TestHTTPAccessHandler(t *testing.T) {
	var e error
	var h http.Handler
	h = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		_, e = w.Write([]byte(`"Success"`))
		if e != nil {
			t.Fatal(e)
		}
	})

	outC, logWriter := StartLogCapturing()
	logger := NewWithOutput("TestHttpAccessHandler", logWriter)
	h = NewPanicHandler(
		tracing.NewRequestContextHandler(
			NewRequestLoggerHandler(logger,
				NewHTTPAccessHandler(h))))
	r := httptest.NewRequest("GET", "/tenant1/foo?param1=value1", nil)
	r.Header.Add("Referer", "www.referertest.com")
	r.Header.Add("User-Agent", "test-user-agent")
	r.RemoteAddr = "1.2.3.4:1234"
	r.Header.Add("X-Forwarded-For", "119.14.55.14, 119.14.55.15, 119.14.55.16")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)

	r1 := httptest.NewRequest("GET", "/tenant1/foo?param1=value1", nil)
	r1.RemoteAddr = "1.2.3.4:1234"
	r1.Header.Add("X-Forwarded-For", "a.b.c.d.e") // xff silently ignore this
	h.ServeHTTP(w, r1)

	r2 := httptest.NewRequest("GET", "/tenant1/foo?param1=value1", nil)
	r2.Header.Add("Referer", "abcde") // Will not verify for proper format
	r2.RemoteAddr = "1.2.3.4:1234"
	r2.Header.Add("X-Forwarded-For", "") // no effect
	h.ServeHTTP(w, r2)

	s := StopLogCapturing(outC, logWriter)

	assert.Equal(t, w.Code, 200, "Status code not correct")

	assert.Contains(t, s[0], "requestId")
	assert.Contains(t, s[0], "durationMS")
	assert.Contains(t, s[0], `"path":"/tenant1/foo"`)
	assert.Contains(t, s[0], `"rawQuery":"param1=value1"`)
	assert.Contains(t, s[0], `"method":"GET"`)
	assert.Contains(t, s[0], `"tenant":"tenant1"`)
	assert.Contains(t, s[0], `"code":200`)
	assert.Contains(t, s[0], `"responseBytes":9`)
	assert.Contains(t, s[0], `"referer":"www.referertest.com"`)
	assert.Contains(t, s[0], `"realClientIP":"119.14.55.14:1234"`)
	assert.Contains(t, s[0], `"userAgent":"test-user-agent"`)

	assert.Contains(t, s[1], `"realClientIP":"1.2.3.4:1234"`)
	assert.Contains(t, s[1], `"referer":""`)
	assert.Contains(t, s[2], `"realClientIP":"1.2.3.4:1234"`)
	assert.Contains(t, s[2], `"referer":"abcde"`)
}
