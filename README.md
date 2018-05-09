# Package logging [ ![Codeship Status for splunk/sharedlogging](https://app.codeship.com/projects/abacb120-1375-0136-2114-428a351088a3/status?branch=master)](https://app.codeship.com/projects/283203)
The logging package provides a standard for golang SSC services to instrument their services according to the [SSC Logging Standard](https://confluence.splunk.com/display/PROD/ERD%3A+Shared+Logging) format. Features include structured leveled logging, request loggers, component loggers, and http access tracing. This logging package wraps a more complicated logging package (zap) and exposes just the APIs needed to instrument your service according to the SSC standard.

## Setup
Add this line to import:
```
import (
	"github.com/splunk/ssc-observation/logging"
)
```
### An Important Note About Private Repositories
Codeship does not have a solution for resolving imports to private repositories. So the recommended approach for using the ssc-observation repository is to checkin your entire vendor directory. For KV Store this meant:
1) Removing 'dep ensure' steps from Codeship. Even with all the packages checked in it will still try to resolve them for verification (apparently).
2) Adding a 'make dep' target to Makefile to run 'dep ensure'
3) Removing vendor from .gitignore
4) Git adding the files under /vendor and submitting

Other engineers will have to manually delete the vendor directory before they can pull the repo updates that include the vendor directory contents as part of the repo.

## Features
* Implements [Splunk SSC Logging Standards](https://confluence.splunk.com/display/PROD/ERD%3A+Shared+Logging) structured leveled logging
* Built over [zap](https://godoc.org/go.uber.org/zap) to provide low allocation, high performance logging
* Context request tracing with provided middleware handlers
* Child loggers including 'component loggers' (channel loggers)

Forthcoming features not yet implemented:
* Level set/get that is shared across a chain of loggers (parent-child), and the ability to isolate a logger's level setting from others (like atomic levels in zap)
* Load capped sampling: add config support for zap sampling to put a cap on CPU and I/O load
* Logger sampling: add support to emit a set sampling of traces (for example, 1% of http 200 requests)
* Distributed tracing: integrate [opentracing-go](https://github.com/opentracing/opentracing-go)
* Logging administration: remotely set logging levels on registered loggers
* A separate dev tool for humanizing logs
* More middleware HTTP handlers and middleware features:
  * Support for X-DEBUG-TRACE http header to enable debug tracing for that request
  * Tenant context in request tracing
  * HTTP access tracing (errors, sampled non-errors)

## Basic Usage
This is how you instantiate a new logger for your service:
```go
// In service main create the service logger
log := logging.New("service1")
log.Info("Service starting")

// Set it to be the global logger so that adding a log statement doesn't
// require flowing it through all intermediate functions.
logging.SetGlobalLogger(log)

// Elsewhere in the service...
// Access the global logger with logging.Global()
log = logging.Global()
log.Info("message1")

// Five logging levels are available. Error and Fatal take an err argument which adds {"error": err.Error()}.
// Requiring both error and message encourages inclusion of a useful contextual message.
log.Debug("Debug message")
log.Info("Info message")
log.Warn("Warn message")
err := errors.New("Invalid request")
log.Error(err, "Error message")
log.Fatal(err, "Fatal message")

// Since this is structured logging it is easy to include sets of key-value pairs in a variadic list.
// More on this in later sections
log.Info("Hello World!", "status", status, "duration", elapsed)

// Expensive operations can be guarded with log.DebugEnabled() and log.Enabled(level)
if log.DebugEnabled() {
	// ...do something expensive here...
	log.Debug("message2")
}

// Call Flush before service exit
defer log.Flush()
```
Here is an example output, note the inclusion of standard fields:
```json
{"level":"INFO","time":"2018-04-22T20:42:08.043Z","file":"examples/main.go:61","message":"Starting service","service":"service1","hostname":"df721610cf14"}
```

## Structured Logging
Structured logging means including specific key-value pairs instead of a formatted string. In fact, to encourage structured tracing message no formatting
methods are included (e.g., no log.Infof("Foo: %s", name)). The trace functions all include a 'message' parameter and a variadic fields parameter. The fields
parameter is an alternating list of keys and values. As noted above the Fatal() and Error() methods take an err and a message string. For example,
```go
// A structured message with "status" and "duration" fields
log.Info("Hello World!", "status", status, "duration", elapsed)
// Or formatted vertically
log.Info("Hello World!",
	"status", status,
	"duration", elapsed)

// To facilitate log consumption there are standard logging keys for common key names.
// See logging/logger.go for the full list.
log.Info("S3 bucket created", logging.UrlKey, url)
// Or for example if you want to trace an error message at the Info level
log.Info("Request failed, retrying", logging.ErrorKey, err, "retryCount", count)
```
An example output of structured logging:
```json
{"level":"INFO","time":"2018-04-19T15:27:50.185Z","file":"service1/main.go:14","message":"Hello World!","service":"service1","hostname":"df721610cf14","status":200,"duration":"3.3ms"}
```

## Adding Logger Fields to Child Loggers
Child loggers can be created with additional fields to be included in each trace output. The child logger is a clone of the parent logger and will include the fields of the parent logger. Since log.With() creates a clone it should only be used when you need a logger for multiple log traces. It is not a fluent-alternative to using the variadic fields of Info, Error, etc...
```go
log := logging.Global()
log.Info("Logging with standard fields")
var value1, value2 string
childLogger := log.With("custom1", value1, "custom2", value2)
childLogger.Info("Logging with custom field")
```
This will produce the lines:
```json
{"level":"INFO","time":"2018-04-19T15:25:56.567Z","file":"service1/main.go:13","message":"Logging with standard fields","service":"service1","hostname":"df721610cf14"}
{"level":"INFO","time":"2018-04-19T15:25:56.568Z","file":"service1/main.go:15","message":"Logging with custom field","service":"service1","hostname":"df721610cf14","custom1":"value1","custom2":"value2"}
```

## Golang Context
The logging package integrates with golang's context package to flow loggers through the processing path. Don't pass a logger instance, instead add 'ctx context.Context' as the first parameter and pass the logger through ctx. In addition to making the logger available the context can provide other useful features like cancellation, deadlines and request-id flowing.  Adding context support into an existing service can be rather invasive but once done its a good thing.

Most commonly you will get a context from an http request or when creating a component logger (examples of that below). In the less common case that you need to create a context with a specific logger you can do it using logging.NewContext, as follows:
```go

func foo() {
	ctx := logging.NewContext(context.Background(), logging.New("logger1"))
	bar(ctx)

        // If you pass a context with no logger in it then the global logger will be used
        var value string
        bar(context.Background(), value)
}

// As a convention, ctx should always be the first parameter
func bar(ctx context.Context, value string) {
        // Extract the logger from the context
	log := logging.From(ctx)
        log.Info("Called bar", "value", value)
}
```

If you're not familiar with golang context here are a few blog posts that can provide some bacgkround [Go Concurrency Patterns by the Go team](https://blog.golang.org/context), [How to use context.Context](https://blog.gopheracademy.com/advent-2016/context-logging/), and [Context-logging](https://blog.gopheracademy.com/advent-2016/context-logging/).

## Request Logging
Request logging is a critical aspect of service instrumentation. With request logging a unique request id flows through the call path of every request (and in the future across service boundaries). A unique request logger is cloned from a parent logger and will trace the request id as {"requestId": requestId} on every log trace. This enables correlation of all request related traces even in the context of highly concurrent request processing. Golang features a core library package context for flowing things like request ids and loggers through a call path and to coordinate features like deadlines and cancellation. The logging package uses context.Context for just this purpose. In other words you don't pass request loggers, you pass 'ctx context.Context'.

Http request logging is probably the most common scenario but not the only scenario. For example you could consider pulling a batch of data from kubernetes and processing that through a pipeline as a single request with a unique request id, especially valuable if concurrent requests can flow through the pipeline(s).


The logging package has several features to facilitate request logging. For http cases there is an http.HttpHandler middleware that can be added to your service pipeline to automatically enhance the incoming http request context with a request logger. The logging.From(ctx) api lets you extract the logger in that context. The NewRequestContext() API lets you directly create a context for scenarios where you are not using an http handler. See code examples below.

In addition to wiring in code to create the request logger, you must also modify the request call path to have 'ctx context.Context' as the first parameter. You may find it useful in places to use context.TODO() as a temporary context as you iteratively transform your code. Finally, context.Background() provides the root context for places where no other context exists.

### Non-HTTP Request Logging
While not as common, request logging in the non-http case is easy. Simply use logging.NewRequestContext() directly. Note that the parent logger used to clone the request logger from will be taken from the provided context using logging.From(ctx). If no logger is found in ctx then the global logger is used.
```go
func ExampleNonHttpRequest() {
	requestId := "" // pass "" to let the logger create one
        // NewRequestContext creates a new context with a request logger in it.
        // If the supplied context as no logger then the global logger is used as the parent logger.
	ctx := logging.NewRequestContext(context.Background(), requestId)
	logging.From(ctx).Info("New batch started")
}
```

### HTTP Request Logging with Swagger
Integrating request logging with Swagger generated code is easy. An http handler is added to the pipeline and then ctx is pulled from the http request.

First mix-in a call to logging.NewRequestHandler() in setupGlobalMiddleware() which is found in a file something like configure_kvstore.go. Note that in this case logging.Global() is used to set the parent logger to the request logger. Any custom logger will do.
```go
// The middleware configuration happens before anything, this middleware also applies to serving the swagger.json document.
// So this is a good place to plug in a panic handling middleware, logging and metrics
func setupGlobalMiddleware(handler http.Handler) http.Handler {
	return handlers.NewPanicHandler(
		logging.NewRequestHandler(logging.Global(),
			handlers.NewHttpAccessLoggingHandler(
				handlers.NewRateLimitHandler(handler))))
}
```

Next extract the context (ctx) from the http request in each strongly-typed swagger handler. And then pass that ctx parameter to the functions implementing that API. In any function you want to use the logger just extract it from ctx using 'log := logging.From(ctx)'
```go
func CreateCollectionHandler(params operations.CreateCollectionParams) middleware.Responder {
	ctx := params.HTTPRequest.Context()

        // Extract the logger from ctx wherever you need it
        log := logging.From(ctx)
        log.Info("CreateCollection called")

        // Pass ctx as the first parameter all along the request path
	if e := kvstore.CreateCollection(ctx, params.Tenant, params.Namespace, *params.Collection.Name); e != nil {
		return errors.Serve(&operations.CreateCollectionDefault{}, e)
	}
	return operations.NewCreateCollectionCreated()
}
```

### Http Request Logging without Swagger
If you are not using Swagger then you will have your own calls to http.Handle() to set up the request routing. This section shows in detail how to accomplish request logging in such a service.

As with the Swagger case, the logging.NewRequestHandler() API is used to add a request logging handler into your services http handler processing pipeline. This handler will create a new logger for each http request that will trace the request id. This logger is added to the http request context.
```go
// In your service middleware wire in the logging request handler...

// Adapt the handler func to an http.Handler
var handler http.Handler
handler = http.HandlerFunc(operation1HandlerFunc)

// Wrap operation1Handler with the request logging handler that will set up
// request context tracing. Use the global logger as the parent for each request logger.
handler = logging.NewRequestHandler(logging.Global(), handler)
http.Handle("/operation1", handler)
```

In the middleware handler for each operation pull out the ctx that the NewRequestHandler() handler added to the http request
```go
// The middleware handler for operation1 extracts the context and passes it to the
// strongly-typed operation1() func.
func operation1Handler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	param1 := r.URL.Query().Get("param1")
	operation1(ctx, param1)
}
```

And then use logging.From(ctx) to get the request logger. The 'ctx context.Context' parameter should be passed throughout the request call path.
```go
// Strongly-typed implementation for service1.operation1.
func operation1(ctx context.Context, param1 string) {
	// Get the request logger from ctx, it was added there
	// by the logging.RequestHandler
	log := logging.From(ctx)

	log.Info("Executing operation1", "param1", param1)

        // Flow context to any routines that take a context including standard db routines
        rows, err := db.QueryContext(ctx, query, args...)
        if err != nil {
   	   log.Error(err, "Query error")
           return
        }

        // pass ctx to internal functions too
        transmogrifier(ctx, rows)
}

// Internal functions that need logging should take ctx as a first argument
func transmogrifier(ctx context.Context, rows) {
       log := logging.From(ctx)
       log.Info("Transmogrifying")
}
```

The request tracing above will generate the following output, note the requestId standard field.
```json
{"level":"INFO","time":"2018-04-22T20:42:08.047Z","file":"examples/main.go:103","message":"Executing operation1","service":"service1","requestId":"5add5610eb744568c6000001","param1":"value1"}
{"level":"ERROR","time":"2018-04-22T20:42:08.047Z","file":"examples/main.go:107","message":"Bad request","service":"service1","requestId":"5add5610eb744568c6000001","param1":"value1","error":"Bad value for param1"}
```

A complete example for request tracing can be found in the [examples/main.go](https://github.com/splunk/logging/blob/master/examples/main.go).

## Component Logging
A component logger is simply a logger used in a specific parts of the program. It traces out {"component": componentName}. There will be features in the future for registering these named loggers so they can be remotely administrated. Component loggers can be useful in non-request paths, for example a goroutine that does background reaping of stale data.

A component logger and the context containing it can be created using logging.NewComponentContext(ctx, componentName). The passed in ctx is used to derive the new context with the new component logger. The passed in ctx is also used to get the parent logger via logging.From(ctx). If no logger is found then logging.Global() is used.
```go
func reaper(done <-chan struct{}, wg *sync.WaitGroup) {
	defer wg.Done()
	ticker := time.NewTicker(TempCollectionCheckInterval)
	defer ticker.Stop()
	ctx := logging.NewComponentContext(context.Background(), "lookupReaper")
	for {
		select {
		case <-ticker.C:
			reapTempCollections(ctx)
		case <-done:
			return
		}
	}

}

func reapTempCollections(ctx context.Context) {
	log := logging.From(ctx)
        // code elided...you get the picture by now
}
```

## Unit Tests
Your unit tests will need to be updated to flow in ctx to all the runtime functions that now require it. logging.NewTestContext() can help create that context. Furthermore early-adopters will note that there is no default global logger (for now). You can use TestMain() to set the global logger.

```go
func TestMain(m *testing.M) {
	logging.SetGlobalLogger(logging.New("unit-test"))
}

func TestCreateCollection(t *testing.T) {
	ctx := logging.NewTestContext(t.Name())
        // ...
	if e := CreateCollection(ctx, testTenant, "app1", "table1"); e != nil {
		t.Fatal(e)
	}
}
```

## Migrating from Logrus
In Logrus, we might do something like:
```go
log.WithFields(log.Fields{"param1": param1, "method": method}).Info("Request")
```
with this package, the equivalent would be:
```go
log.Info("Request", "param1", param1, "method", method)
```

## Real World Examples:
Below is an example of what the logging output from the KVStore service.
```json
{"time":"2018-04-04T13:24:36.14Z", "level":"DEBUG", "message":"Running sql query", "pid":8117, "hostname":"e8f220aee09e", "service":"kvstore", "args":"[]interface {}(nil)", "file":"kvstore/db.go:182", "query":"SELECT schemas.schema_name, tables.table_name, indexes.indexname, indexes.indexdef\n\t\tFROM\n\t\t\tinformation_schema.schemata schemas\n\t\t\tLEFT OUTER JOIN\n\t\t\tinformation_schema.tables tables\n\t\t\tON (tables.table_schema = schemas.schema_name)\n\t\t\tLEFT OUTER JOIN\n\t\t\tpg_indexes indexes\n\t\t\tON (indexes.tablename = tables.table_name)\n\t\t\t\tAND indexes.indexname != concat(substring(indexes.tablename from 0 for (58)), '_pkey')\n\t\tWHERE schemas.schema_owner = 'testTenant' AND schemas.schema_name='testNamespace'"}
{"time":"2018-04-04T13:24:36.14Z", "level":"DEBUG", "message":"Query metrics", "pid":8117, "hostname":"e8f220aee09e", "service":"kvstore", "Post-query processing time":"22.215µs", "Query time":"4.268033ms", "file":"kvstore/db.go:149"}
{"time":"2018-04-04T13:24:36.14Z", "level":"DEBUG", "message": "Handled request", "hostname":"e8f220aee09e", "pid":8117, "service":"kvstore", "code":200, "durationMS":4.445762, "file":"handlers/handlers.go:83", "method":"GET", "path":"/testTenant/kvstore/v1/testNamespace"}
```

## FAQ
* __What is the perf impact of tracing file and line?__ The expectation is that this is cheap enough but will confirm with some benchmarking.
* __What else will be in this repository?__ Expect the repository to contain logging and metrics APIs but nothing else. It is not the plan to create a single shared library repository.
* __How can I replace the request logger with customizations when using the request handler?__ In your code you can use ```logger = logger.With(...)``` to create a custom logger and ```ctx = logging.NewContext(ctx, logger)``` to put the logger in the context so logging.From(ctx) can be used in subsequent functions.
* __Where does the log output go?__ Currently the logging api is not opinionated (beyond defaulting to stdout) as to the destination of the logging output. Need to engage with the k8s folks to see how these are supported: multiple containers in a pod, getting older logs, k8s toolchain support.
* __What about other languages?__ This effort is focused on a golang. All services should follow the standard logging format. Ideally services using other languages can work together to define a shared library for their language.

## Troubleshooting
1. Issue: I wrapped this library's log functions and now `file` does not show the correct file. `"file":"myservice/myloggerwrapper.go:81"`
Solution: When you initialize your logger, you can specify a callstack skip. Example: `logger := logging.New("kvstore").SetCallstackSkip(-1)`
This works on child loggers too.

## License
Copyright 2018, Splunk. All Rights Reserved.
