package handlers

import (
	"net/http"
	"strings"

	"cd.splunkdev.com/libraries/go-observation/tracing"
)

type operationHandler struct {
	next http.Handler
}

// NewOperationHandler creates a middleware instance that gets the operation id from
// url path.
func NewOperationHandler(next http.Handler) http.Handler {
	return &operationHandler{
		next: next,
	}
}

func (h *operationHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	// Assumes url path is /{tenantId}/{operationName}
	// so 3 parts given the leading "/"
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) > 2 {
		ctx = tracing.WithOperationID(ctx, parts[2])
		r = r.WithContext(ctx)
	}

	h.next.ServeHTTP(w, r)
}
