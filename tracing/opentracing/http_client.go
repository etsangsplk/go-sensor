package opentracing

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"strings"
	// opentracing "github.com/opentracing/opentracing-go"
	// "golang.org/x/net/context/ctxhttp"
)

type Client struct {
	httpClient     *http.Client
	requestContext context.Context
	tracer         Tracer
	traceRequest   RequestFunc // Request wrapper
}

// NewRequest returns a new request from upstream ctx context.
func NewRequest(ctx context.Context, method string, url string, body io.Reader) (*http.Request, error) {
    req, err := http.NewRequest(http.MethodPost, url, body)
    if err != nil {
        return nil, err
    }
    //req = outBoundHTTPRequest(req.WithContext(ctx))
    return req, err
}

// NewHTTPClient returns a new client instance from upstream ctx context.
// It wraps http.client and add opentracing meta data into outbound http.request.
func NewHTTPClient(ctx context.Context) *Client {
	c := &Client{
		requestContext: ctx,
		httpClient:     &http.Client{},
		tracer:         Global(), // assume that there is a tracer already registered.
	}
	c.traceRequest = OutboundHTTPRequest()
	return c
}

// Do adds opentracing http header instrumentation before making a http Do request.
func (c *Client) Do(req *http.Request) (*http.Response, error) {
	ctx := c.requestContext
	req = c.traceRequest(req.WithContext(ctx))
	return c.httpClient.Do(req)
}

// Get adds opentracing http header instrumentation before making a http Get request.
func (c *Client) Get(url string) (resp *http.Response, err error) {
	ctx := c.requestContext
	// Since we need to annotate the http header, we need to use NewRequest
	// Looks like golang also just use nil for last parameter.
	// https://github.com/golang/go/blob/1b9f66330b123b042e738eff5e47e869dd301a98/src/net/http/client.go#L369
	req, err := http.NewRequest(http.MethodGet, url, nil)
	req = c.traceRequest(req.WithContext(ctx))
	return c.httpClient.Do(req)
}

// Head adds opentracing http header instrumentation before making a http Head request.
func (c *Client) Head(url string) (resp *http.Response, err error) {
	ctx := c.requestContext
	req, err := http.NewRequest(http.MethodHead, url, nil)
	req = c.traceRequest(req.WithContext(ctx))
	return c.httpClient.Do(req)
}

// Post adds opentracing http header instrumentation before making a http Post request.
func (c *Client) Post(url string, contentType string, body io.Reader) (resp *http.Response, err error) {
	ctx := c.requestContext
	// https://github.com/golang/go/blob/1b9f66330b123b042e738eff5e47e869dd301a98/src/net/http/client.go#L736
	req, err := http.NewRequest(http.MethodPost, url, body)
	req = c.traceRequest(req.WithContext(ctx))
	return c.httpClient.Do(req)
}

// PostForm adds opentracing http header instrumentation before making a http PostForm request.
func (c *Client) PostForm(url string, data url.Values) (resp *http.Response, err error) {
	// https://github.com/golang/go/blob/1b9f66330b123b042e738eff5e47e869dd301a98/src/net/http/client.go#L774
	return c.Post(url, "application/x-www-form-urlencoded", strings.NewReader(data.Encode()))
}
