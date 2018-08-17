# go-observation

The go-observation repository provides golang packages for instrumenting services with logs and metrics.

The logging package contains a full API for structured logging plus http middleware handlers. See the documentation section below.

The metrics package provides only http middleware for metrics instrumentation. Services should use the Prometheus client API for metrics instrumentation. See the documentation section below.

The tracing package provides http middleware and context APIs for enriching the http request context with common instrumentation values like tenant id, request id and operation id. The context APIs can be used directly as an extensibility point to enrich http request context in custom ways for use by metrics and logging package handlers. Since services use a variety of request routing approaches there is no standard middleware for setting the operation id context. See the example below for how to do this with open-api (swagger) based services

# Documentation and Support

- [Alerts](./alerts/README.md)
- [Logging](./logging/README.md)
- [Metrics](./metrics/README.md)
- [Prometheus Client API](https://godoc.org/github.com/prometheus/client_golang/prometheus)
- [Metrics Architecture](https://docs.google.com/document/d/11AlcILE3S_7XE5t3hgUAYSJCsosbFbzGQW2VALcz-hU/edit?usp=sharing)

Inspecting the go-observation code itself is also a great resource and most of the public APIs are well documented.

Join the go-observation slack channel to ask questions and hear announcements on improvements and changes.

## An Important Note About Private Repositories
Codeship does not have a solution for resolving imports to private repositories. So the recommended approach for using the go-observation repository is to checkin your entire vendor directory. For KV Store this meant:
1) Removing 'dep ensure' steps from Codeship. Even with all the packages checked in it will still try to resolve them for verification (apparently).
2) Adding a 'make dep' target to Makefile to run 'dep ensure'
3) Removing vendor from .gitignore
4) Git adding the files under /vendor and submitting

Other engineers will have to manually delete the vendor directory before they can pull the repo updates that include the vendor directory contents as part of the repo.

## Middleware Handler Configuration
HTTP middleware handlers in tracing, logging and metrics packages compose together to provide standardized instrumentation on your service's http request path. These are based on the idiomatic http.Handler interface. The following example from the KV Store Service demonstrates how to compose the handler construction functions. There are cross-handler dependencies and so the order of construction and the chaining are important for proper behavior. The tracing package has handlers that add new values to the http request context which can then be used by logging and metrics middleware. This provides an extensibility point for services that want to add tenantID, requestID or operationID in a custom way (operationID is always custom).

| Middleware | Behavior and dependencies
|------------|--------------------------
| logging.NewPanicHandler          | Logs out panics using the global logger and returns a 500 error response and body. Should be at the base to catch all panics. No dependencies.
| metrics.NewPrometheusHandler     | Serves up the metrics endpoint /service/metrics to be scraped by the Prometheus server.
| tracing.NewRequestContextHandler | Adds requestID and tenantID to the http request context. See context APIs in the tracing package.
| handlers.NewOperationHandler     | A custom service handler to add operationID to the http request context. By convention the operationID should match what is in the service's open-api spec.  Each service will have to implement their own custom handler, see the next section for an example of how to do this.
| logging.NewRequestLoggerHandler  | Creates a request logger and adds it to the http request context. The request logger will trace requestID, operationID and tenantID if they are available in the http context (set by earlier handlers).
| logging.NewPanicRequestHandler   | Logs out request-scoped panics using the request logger. Will re-panic control to the global panic handler which must also be configured. Depends on NewRequestLoggerHandler and NewPanicHandler.
| logging.NewHTTPAccessHandler     | Logs out http access logs using the request logger. Optionally depends on NewRequestLoggerHandler and its dependencies or custom middleware to provide context.
| metrics.NewHTTPAccessHandler     | Observes http access metrics using request-scoped context. Optionally depends on operationID context. Must call the metrics.RegisterHTTPMetrics function during service initialization.

```go
// These 'global' handlers do not depend on routing operation id and should come before those listed in func setupMiddleware below
func setupGlobalMiddleware(handler http.Handler) http.Handler {
	return logging.NewPanicHandler( // root panic handler, logs using global logger
		metrics.NewPrometheusHandler( // publish the metrics endpoint
			tracing.NewRequestContextHandler( // add requestID and tenantID to context
				handlers.NewRateLimitHandler(handler)))) // a custom service handler
}

// These handlers optionally depend on operationID in context which is determined during routing. The operationID should be the value used in the service's open-api spec.
func setupMiddleware(handler http.Handler) http.Handler {
	return handlers.NewOperationHandler( // add route.Operation.ID
		logging.NewRequestLoggerHandler(logging.Global(), // create the request logger and add it to context
			logging.NewPanicRequestHandler( // request panic handler logs using request logger
				logging.NewHTTPAccessHandler( // emit http access logs
					metrics.NewHTTPAccessHandler( // observe http metrics. metrics.RegisterHTTPMetrics(serviceName) must be called during service initialization
						handlers.NewAuthHandler(handler)))))) // a custom service handler
}
```

## Operation ID for Services using Open-API Generated Services
The open-api spec for your service contains an operation id for each endpoint. This id provides a human-friendly, low-cardinality way to aggregate http requests for metrics and provide context in logs. The tracing package provides an extensible way to enrich http request context with an operationID value. Because SSC services use a variety of routing packages or custom routing logic there is no standard handler for doing this.

As an example the middleware below shows how to do this. In this example the service is using open-api routing (swagger). Note, the middleware.MatchedRouteFrom() API is only available in more recent versions of go-openapi.

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
In addition to instrumenting your service with metrics you also need to make it discoverable by the Prometheus Server running in the kubernetes environment. Read the [Metrics Endpoint Discovery section](./metrics/README.md#metrics-endpoint-discovery) for more details.

## License
Copyright 2018, Splunk. All Rights Reserved.
