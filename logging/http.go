package logging

import (
	"errors"
	"fmt"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/splunk/ssc-observation/tracing"
)

// httpAccessHandler logs http access logs
type httpAccessHandler struct {
	next http.Handler
}

// NewHTTPAccessHandler constructs a new middleware instance for emitting
// http access logs.
func NewHTTPAccessHandler(next http.Handler) http.Handler {
	return &httpAccessHandler{next: next}
}

// ServeHTTP implements http.Handler interface
func (h *httpAccessHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	rw := newHTTPResponseWriter(w)
	start := time.Now()
	h.next.ServeHTTP(rw, r)
	duration := time.Since(start)
	durationStringMS := fmt.Sprintf("%0.3f", duration.Seconds()*1000)

	ctx := r.Context()
	operation := tracing.OperationIDFrom(ctx)
	log := From(ctx)
	log.Info("Handled request",
		"method", r.Method,
		"operation", operation,
		"code", rw.StatusCode(),
		"responseBytes", rw.ResponseBytes(),
		"durationMS", durationStringMS,
		"path", r.URL.Path,
		"rawQuery", r.URL.RawQuery)
}

type requestLoggerHandler struct {
	handler      http.Handler // the next handler in the chain
	parentLogger *Logger
}

// NewRequestLoggerHandler creates a middleware instance that adds the request logger to
// the http request context. The request logger will include fields for tenant id and
// request id if those values are found on context (see the tracing package).
// If parentLogger is nil then the global logger
// will be used as the parent logger.
func NewRequestLoggerHandler(parentLogger *Logger, handler http.Handler) http.Handler {
	return &requestLoggerHandler{
		handler:      handler,
		parentLogger: parentLogger,
	}
}

// ServeHTTP implements the http.Handler interface.
func (h *requestLoggerHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if h.parentLogger != nil {
		ctx = NewContext(ctx, h.parentLogger)
	}
	if requestID := tracing.RequestIDFrom(ctx); len(requestID) > 0 {
		ctx = NewRequestContext(ctx, requestID)
	}
	if tenantID := tracing.TenantIDFrom(ctx); len(tenantID) > 0 {
		ctx = NewTenantContext(ctx, tenantID)
	}
	r = r.WithContext(ctx)
	h.handler.ServeHTTP(w, r)
}

// A private type used for communication between panicRequestHandler and panicHandler
type requestPanicKey int

type panicHandler struct {
	next http.Handler
}

// NewPanicHandler creates a middlware instance that handles panics and
// logs the error and callstack using the global logger. It also writes
// out an http response header and json body for an internal server error (status code 500).
// NewPanicRequestHandler can be used in addition to this handler to get panic logs with
// request context information (request id and tenant id).
func NewPanicHandler(next http.Handler) http.Handler {
	return &panicHandler{next: next}
}

func (p *panicHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	defer func() {
		if rerr := recover(); rerr != nil {
			e := panicError(rerr)
			// This global panic handler comes before the request logging handler so it
			// must use the global logger
			log := Global()
			if e != nil && log != nil {
				log.Error(e, "Handled panic", CallstackKey, debug.Stack())
			}
			w.Header().Add("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			_, e = w.Write([]byte(`{"code": 500, "message": "Internal server error"}`))
			if e != nil {
				log.Error(e, "Unable to write http response")
			}
		}
	}()

	p.next.ServeHTTP(w, r)
}

type panicRequestHandler struct {
	next http.Handler
}

// NewPanicRequestHandler creates a middleware instance that handles panics and
// logs the error and callstack using the request logger from http context. It should
// be registered after NewRequestLoggerHandler. This handler requires that
// the global panic handler registered with NewPanicHandler be set at the root
// as this handler does not write out an error response.
func NewPanicRequestHandler(next http.Handler) http.Handler {
	return &panicRequestHandler{next: next}
}

func (p *panicRequestHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	defer func() {
		if rerr := recover(); rerr != nil {
			e := panicError(rerr)
			log := From(r.Context())
			if log != nil {
				log.Error(e, "Handled panic", CallstackKey, string(debug.Stack()))
			}
			// Now panic again with a special token fall out to the global
			// panic handler
			// TODO: do we want to handle the case where we don't find a logger in ctx?
			//     : for that we would want to flow the callstack with the current panic
			panic(requestPanicKey(1))
		}
	}()

	p.next.ServeHTTP(w, r)
}

// panicError converts the recover arg rerr to an error.
// If the error was previously logged by the panic request handler then
// nil is returned.
func panicError(rerr interface{}) error {
	switch rerr := rerr.(type) {
	case string:
		return errors.New(rerr)
	case error:
		return rerr
	case requestPanicKey:
		// already logged in panicRequestHandler
		return nil
	default:
		return fmt.Errorf("Unknown panic '%v' of type %T", rerr, rerr)
	}
}
