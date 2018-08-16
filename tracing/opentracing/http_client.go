package opentracing

import (
	"context"
	"io"
	"net/http"
)

// NewRequest returns a new request from upstream ctx context.
func NewRequest(ctx context.Context, method string, url string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}
	req = annotateOutboundRequest(req.WithContext(ctx))
	return req, err
}
