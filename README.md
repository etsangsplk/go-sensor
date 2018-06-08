# ssc-observation [ ![Codeship Status for splunk/ssc-observation](https://app.codeship.com/projects/f6131db0-3764-0136-f72e-36b905590d28/status?branch=master)](https://app.codeship.com/projects/289654)

The ssc-observation repository provides packages for service instrumentation of logs and metrics.

The logging package contains a full API for structured logging instrumentation plus http middleware handlers. See the documentation section below.

The metrics package provides only http middleware for metrics instrumentation. Services should use the Prometheus client API for metrics instrumentation. See the documentation section below.

The tracing package provides http middleware and context APIs for enriching the http request context with common instrumentation values like tenant id, request id and operation id. Since services use a variety of request routing approaches there is no standard middleware for setting the operation id context. See the example below for how to do this with open-api (swagger) based services

# Documentation and Support
[Logging README](https://github.com/splunk/ssc-observation/tree/master/logging)

[Metrics README](https://github.com/splunk/ssc-observation/tree/master/metrics)

[Metrics Architecture](https://docs.google.com/document/d/11AlcILE3S_7XE5t3hgUAYSJCsosbFbzGQW2VALcz-hU/edit?usp=sharing)

Inspecting the code itself is also a great resource and most of the public APIs are well documented.

Join the ssc-observation slack channel to ask questions and hear announcements on improvements and changes.

## An Important Note About Private Repositories
Codeship does not have a solution for resolving imports to private repositories. So the recommended approach for using the ssc-observation repository is to checkin your entire vendor directory. For KV Store this meant:
1) Removing 'dep ensure' steps from Codeship. Even with all the packages checked in it will still try to resolve them for verification (apparently).
2) Adding a 'make dep' target to Makefile to run 'dep ensure'
3) Removing vendor from .gitignore
4) Git adding the files under /vendor and submitting

Other engineers will have to manually delete the vendor directory before they can pull the repo updates that include the vendor directory contents as part of the repo.

## Middleware Handler Configuration
HTTP middleware Handlers in tracing, logging and metrics work together to provide standardized instrumentation on your services http code path. The following example from the KV Store Service demonstrates how to compose together the various handlers from the tracing, logging and metrics packages.

```go
// The middleware configuration is for the handler executors. These do not apply to the swagger.json document.
// The middleware executes after routing but before authentication, binding and validation
func setupMiddlewares(handler http.Handler) http.Handler {
	return handlers.NewOperationHandler( // add route.Operation.ID
		logging.NewRequestLoggerHandler(logging.Global(), // create the request logger and add it to context
			logging.NewPanicRequestHandler( // request panic handler logs using request logger
				logging.NewHTTPAccessHandler( // emit http access logs
					metrics.NewHTTPAccessHandler( // observe http metrics
						handlers.NewAuthHandler(handler))))))
}

// The middleware configuration happens before anything, this middleware also applies to serving the swagger.json document.
// So this is a good place to plug in a panic handling middleware, logging and metrics
func setupGlobalMiddleware(handler http.Handler) http.Handler {
	return logging.NewPanicHandler( // root panic handler, logs using global logger
		metrics.NewPrometheusHandler( // publish the metrics endpoint
			tracing.NewRequestContextHandler( // add requestID and tenantID to context
				handlers.NewRateLimitHandler(handler))))
}
```

## Operation ID for Services using Open-API Generated Services
An example middleware handler for use by services generated with the golang open-api (swagger) tool. This handler will extract the operation id from the matched route and set it on the context. This can be used later on the request path for logging fields and metrics labels. This context value is recognized by request handlers in the metrics and logging packages. Note, the middleware.MatchedRouteFrom() API is only available in more recent versions of go-openapi.

```go
import 	"github.com/go-openapi/runtime/middleware"

type operationHandler struct {
	next http.Handler
}

// NewOperationHandler creates a middlware instance that gets the operation id from
// the open-api spec that the request is being routed to. The operation id is then
// added to the http request context and can be retrieved using the tracing package.
func NewOperationHandler(next http.Handler) http.Handler {
	return &operationHandler{
		next: next,
	}
}

func (h *operationHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if route := middleware.MatchedRouteFrom(r); route != nil {
		ctx = tracing.WithOperationID(ctx, route.Operation.ID)
		r = r.WithContext(ctx)
	}
	h.next.ServeHTTP(w, r)
}
```

## Metrics Endpoint Discovery
In addition to instrumenting your service with metrics you also need to make it discoverable by the Prometheus Server running in the kubernetes environment. Read the (Metrics Endpoint Discovery section)[https://github.com/splunk/ssc-observation/metrics#metrics-endpoint-discovery] for more details.

metrics#metrics-endpoint-discovery
## License
Copyright 2018, Splunk. All Rights Reserved.
