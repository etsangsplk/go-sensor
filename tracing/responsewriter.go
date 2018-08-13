package tracing

import (
	"net/http"
)

// Note, there are duplicate copies of this in /logging/responsewriter.go and /metrics/responsewriter.go
// Ultimately it should go in /tracing but is kept here private until the interface stabilizes.

// ResponseWriter implements the http.ResponseWriter interface.
// It is used to capture the response status code.
type httpResponseWriter struct {
	w             http.ResponseWriter
	responseBytes uint64
	statusCode    int
}

func NewHTTPResponseWriter(w http.ResponseWriter) *httpResponseWriter {
	return &httpResponseWriter{w: w}
}

// Implements http.ResponseWriter
func (r *httpResponseWriter) Write(b []byte) (int, error) {
	r.responseBytes += uint64(len(b))
	return r.w.Write(b)
}

// Implements http.ResponseWriter
func (r *httpResponseWriter) WriteHeader(statusCode int) {
	r.statusCode = statusCode
	r.w.WriteHeader(statusCode)
}

// Implements http.ResponseWriter
func (r *httpResponseWriter) Header() http.Header {
	return r.w.Header()
}

func (r *httpResponseWriter) ResponseBytes() uint64 {
	return r.responseBytes
}

func (r *httpResponseWriter) StatusCode() int {
	return r.statusCode
}
