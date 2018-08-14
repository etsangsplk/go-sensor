package tracing

import (
	"net/http"
	"strings"

	"gopkg.in/mgo.v2/bson"
)

type requestContextHandler struct {
	next http.Handler // the next handler in the chain
}

// NewRequestContextHandler creates a tracing request handler that extracts the tracing relevant values
// request ID and tenant ID from the http request. The values are added to the request
// context for use in logging and metrics. The functions
// RequestIDFrom and TenantIDFrom can be used to extract these values back out of context.
// NewRequestContextHandler assumes that the first part of every endpoint path is
// the tenant id.
func NewRequestContextHandler(next http.Handler) http.Handler {
	return &requestContextHandler{
		next: next,
	}
}

// ServeHTTP implements the http.Handler interface.
func (h *requestContextHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// X-REQUEST-ID is the request ID header supplied by the gateway
	requestID := r.Header.Get(XRequestID)
	if len(requestID) == 0 {
		requestID = bson.NewObjectId().Hex()
	}
	tenantID := tenantIDFromPath(r.URL.Path)

	ctx := r.Context()
	ctx = WithRequestID(ctx, requestID)
	if len(tenantID) > 0 {
		ctx = WithTenantID(ctx, tenantID)
	}

	r = r.WithContext(ctx)

	h.next.ServeHTTP(w, r)
}

func tenantIDFromPath(path string) string {
	if len(path) <= 1 || path[0] != '/' {
		return ""
	}
	path = path[1:]
	i := strings.IndexByte(path, '/')
	if i == -1 {
		return ""
	}
	return path[:i]
}
