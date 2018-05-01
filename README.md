# logging [ ![Codeship Status for splunk/sharedlogging](https://app.codeship.com/projects/abacb120-1375-0136-2114-428a351088a3/status?branch=master)](https://app.codeship.com/projects/283203)
A standard logging package for golang SSC services to do context aware tracing in the [SSC Logging Standard](https://confluence.splunk.com/display/PROD/ERD%3A+Shared+Logging) format.

## Setup
Add this line to import (note: current repository is logging and will be changed to ssc-observation as it will contain logging and metrics)
```
import (
	"github.com/splunk/ssc-observation/logging"
)
```

## Features
* Implements [Splunk SSC Logging Standards](https://confluence.splunk.com/display/PROD/ERD%3A+Shared+Logging) structured leveled logging
* Built over [zap](https://godoc.org/go.uber.org/zap) to provide low allocation, high performance logging.
* Context request tracing with provided http.Handler handlers

Forthcoming features not yet implemented:
* Level set/get that is shared across a chain of loggers (parent-child), and the ability to isolate a logger's level setting from others (like atomic levels in zap)
* Load capped sampling: add config support for zap sampling to put a cap on CPU and I/O load
* Logger sampling: add support to emit a set sampling of traces (for example, 1% of http 200 requests)
* Distributed tracing: integrate [opentracing-go](https://github.com/opentracing/opentracing-go)
* Remote logging administration: set logging levels on registered loggers. Related to this would be support for logging 'channels'.
* A separate dev tool for humanizing logs
* Add fluent API
* More middleware HTTP handlers and middleware features:
  * Support for X-DEBUG-TRACE http header to enable debug tracing for that request
  * Tenant context in request tracing
  * HTTP request/response tracing (errors, sampled non-errors)

## Quick Start
### Basic Usage
This is how you instantiate a new logger for your service:
```go
// In service main create the service logger
log := logging.New("service1")
log.Info("Service starting")

// Optionally set it to be the global logger
logging.SetGlobalLogger(log)

// Elsewhere in the service...
// Access the global logger with logging.Global()
log = logging.Global()
log.Info("message1")
log.SetLevel(logging.DebugLevel)
if log.Enabled(logging.DebugLevel) {
	// ...do something expensive here...
	log.Debug("message2")
}

// Call Flush before service exit
defer log.Flush()
```
This will trace out the following json line, note the inclusion of standard fields:
```json
{"level":"INFO","time":"2018-04-22T20:42:08.043Z","file":"examples/main.go:61","message":"Starting service","service":"service1","hostname":"df721610cf14"}
```
Five logging levels are available.
```go
log := logging.Global()
log.Debug("This is a debug log entry")
log.Info("This is an info log entry")
log.Warn("This is a warning log entry")

// Error and Fatal take an err argument. This traces the error as {"error": err.Error()} and encourages inclusion of a useful message with the error message.
err := errors.New("Invalid request")
log.Error(err, "This is a error log entry")
log.Fatal(err, "This is a fatal log entry")
```
The trace functions all include a 'message' parameter and a variadic fields parameter. The fields parameter is an alternating list of keys and values. For example,
```go
log.Info("Hello World!", "status", status, "duration", elapsed)
// or formatted vertically
log.Info("Hello World!",
	"status", status,
	"duration", elapsed)	
)
```
which will produce the line:
```json
{"level":"INFO","time":"2018-04-19T15:27:50.185Z","file":"service1/main.go:14","message":"Hello World!","service":"service1","hostname":"df721610cf14","status":200,"duration":"3.3ms"}
```

### Adding Logger Fields
Child loggers can be created with additional fields to be included in each trace output while still including the fields of the parent logger.
```go
log := logging.Global()
log.Info("Logging with standard fields")
var value1, value2 string
log = log.With("custom1", value1, "custom2", value2)
log.Info("Logging with custom field")
```
This will produce the lines:
```json
{"level":"INFO","time":"2018-04-19T15:25:56.567Z","file":"service1/main.go:13","message":"Logging with standard fields","service":"service1","hostname":"df721610cf14"}
{"level":"INFO","time":"2018-04-19T15:25:56.568Z","file":"service1/main.go:15","message":"Logging with custom field","service":"service1","hostname":"df721610cf14","custom1":"value1","custom2":"value2"}
```

### Request Logging
Most log tracing will be done in the context of an API request or similarly unique context. The log package has several features to facilitate correlated request traces.

The logging.NewRequestHandler function can be added into your services http handler processing pipeline. This handler will add a request-scoped logger to the http request context for each incoming request.
```go
// In your service middleware wire in the logging request handler...

// The middleware handler for operation1 extracts the context and passes it to the
// strongly-typed operation1() func.
operation1HandlerFunc := func(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	param1 := r.URL.Query().Get("param1")
	operation1(ctx, param1)
}

// Adapt the handler func to an http.Handler
var operation1Handler http.Handler
operation1Handler = http.HandlerFunc(operation1HandlerFunc)

// Wrap operation1Handler with the request logging handler that will set up
// request context tracing.
operation1Handler = logging.NewRequestHandler(operation1Handler, logging.Global())
http.Handle("/operation1", operation1Handler)
```
And then use logging.From(ctx) to get the request logger. The 'ctx context.Context' parameter should be passed throughout the request call path.
```go
// Strongly-typed implementation for service1.operation1.
func operation1(ctx context.Context, param1 string) {
	// Get the request logger from ctx, it was added there
	// by the logging.RequestHandler
	log := logging.From(ctx)

	log.Info("Executing operation1", "param1", param1)

	// Example error message, note the special handling for err
	err := fmt.Errorf("Bad value for param1")
	log.Error(err, "Bad request", "param1", param1)
}
```
The request tracing above will generate the following output, note the requestId standard field.
```json
{"level":"INFO","time":"2018-04-22T20:42:08.047Z","file":"examples/main.go:103","message":"Executing operation1","service":"service1","requestId":"5add5610eb744568c6000001","param1":"value1"}
{"level":"ERROR","time":"2018-04-22T20:42:08.047Z","file":"examples/main.go:107","message":"Bad request","service":"service1","requestId":"5add5610eb744568c6000001","param1":"value1","error":"Bad value for param1"}
```
Context request tracing can still be used when outside the scope of an http request.
Simply use logging.NewRequestContext() directly.
```go
func ExampleNonHttpRequest() {
	requestId := "" // pass "" to let the logger create one
        // NewRequestContext creates a new context with a request logger in it.
        // If the supplied context as no logger then the global logger is used as the parent logger.
	ctx := logging.NewRequestContext(context.Background(), requestId)
	logging.From(ctx).Info("New batch started")
}
```
A complete example for request tracing can be found in the [examples/main.go](https://github.com/splunk/logging/blob/master/examples/main.go).

### Proposal for Fluent-style API
Feedback has indicated a desire to also have a fluent-style API. This style API provides stronger-typing, stronger organization and a good way to add support for standard keys. These benefits come at the expense of extra verbosity but can be the preferred style when there are many fields to log. The use of a buffer pool (similar to what fmt and zerolog do) keeps the allocation count low.

There has also been feedback that adding this will result in the logging API having essentially two APIs. For the moment this API is on hold in the 'future features' bucket.

This API consists of:
* Infow(), Debugw() etc... methods that take no parameters and return an *Event. The w suffix stands for 'with fields'. Alternate suggestions are welcome. 'f' for 'fluent' is probably not a good option given Sprintf convention.
* Event type that has Str(key string, value string), Int(key string, value int), etc... methods for each field type supported. Can also have methods like Url(value string) where the key is a pre-defined standard key.
* To emit the trace the Flush() method must be called. A linting tool could be written to catch this.

```go
// This example demonstrates a fluent-api and contrasts it to the flat style
func ExampleFluent() {
	err := fmt.Errorf("An error")
	name, url := "name1", "http://github.com"
	value := 10

	log := logging.New("service1")

	//
	// The fluent style provides typing and structure but requires a call to .Flush() at the end.
	// A linting tool could be written to detect missing calls to Flush().
	// Vertically formatting can be used for long runs
	// Note how golang errors are handled in the fluent and flat styles
	//
	log.Infow("A fluent example").Str("name", name).Int("value", value).Url(url).Flush()

	log.Infow("A fluent example").
		Str("name", name).
		Int("value", value).
		Url(url).
		Flush()

	log.Errorw(err, "A fluent example").Str("name", name).Int("value", value).Url(url).Flush()

	//
	// Flat style shown here for comparison
	// Standard keys like "url" are a bit more tedious in the flat style
	//
	log.Info("A flat example", "name", name, "value", value)
	log.Info("A flat example", "name", name, "value", value, logging.UrlKey, url)
	log.Error(err, "A flat example", "name", name, "value", value, logging.UrlKey, url)
}
```

## Migrating from Logrus
In Logrus, we might do something like:
```go
log.WithFields(log.Fields{"param1": param1, "method": method}).Info("Request")
```
with this package, the equivalent would be in the flat and fluent styles:
```go
log.Info("Request", "param1", param1, "method", method)
log.Info("Request").Str("param1", param1).Str("method", method).Flush()
```

## Real World Examples:
Below is an example of what the logging output from the KVStore service.
```json
{"time":"2018-04-04T13:24:36.14Z", "level":"DEBUG", "message":"Running sql query", "pid":8117, "hostname":"e8f220aee09e", "service":"kvstore", "args":"[]interface {}(nil)", "file":"kvstore/db.go:182", "query":"SELECT schemas.schema_name, tables.table_name, indexes.indexname, indexes.indexdef\n\t\tFROM\n\t\t\tinformation_schema.schemata schemas\n\t\t\tLEFT OUTER JOIN\n\t\t\tinformation_schema.tables tables\n\t\t\tON (tables.table_schema = schemas.schema_name)\n\t\t\tLEFT OUTER JOIN\n\t\t\tpg_indexes indexes\n\t\t\tON (indexes.tablename = tables.table_name)\n\t\t\t\tAND indexes.indexname != concat(substring(indexes.tablename from 0 for (58)), '_pkey')\n\t\tWHERE schemas.schema_owner = 'testTenant' AND schemas.schema_name='testNamespace'"}
{"time":"2018-04-04T13:24:36.14Z", "level":"DEBUG", "message":"Query metrics", "pid":8117, "hostname":"e8f220aee09e", "service":"kvstore", "Post-query processing time":"22.215µs", "Query time":"4.268033ms", "file":"kvstore/db.go:149"}
{"time":"2018-04-04T13:24:36.14Z", "level":"DEBUG", "message": "Handled request", "hostname":"e8f220aee09e", "pid":8117, "service":"kvstore", "code":200, "durationMS":4.445762, "file":"handlers/handlers.go:83", "method":"GET", "path":"/testTenant/kvstore/v1/testNamespace"}
```

## FAQ

* __Any ideas for a shorter pacakge name?__ One idea for a shorter package name is 'lg'. It is idiomatic in go to use short package names for very common packages. Share your feedback if you have ideas on this. It was decided to keep 'log' available as the conventional name for a logger intance. 
* __Why not have the import be "github.com/splunk/logging"?__ Most likely we will end up with "github.com/splunk/ssc-logging/logging". We will maintain separate repos even for shared packages (simplifies breaking change management) and we need a repo name that is ssc specific.
* __What is the perf impact of tracing file and line?__ The expectation is that this is cheap enough but will confirm with some benchmarking.
* __What else will be in this repository?__ Expect the repository to contain logging and metrics APIs but nothing else. It is not the plan to create a single shared library repository.
* __How can I replace the request logger with customizations when using the request handler?__ In your code you can use ```logger = logger.With(...)``` to create a custom logger and ```ctx = logging.NewContext(ctx, logger)``` to put the logger in the context so logging.From(ctx) can be used in subsequent functions.
* __Where does the log output go?__ Currently the logging api is not opinionated (beyond defaulting to stdout) as to the destination of the logging output. Need to engage with the k8s folks to see how these are supported: multiple containers in a pod, getting older logs, k8s toolchain support.
* __What about other languages?__ This effort is focused on a golang API. All services should follow the standard logging format. Ideally services using other languages can work together to define a shared library for their language.
* __Why aren't their formatting methods like Infof()?__ There are no formatting methods, like log.Infof("Bad param %s", v), to encourage use of structured logging fields. In the rare case you need string formatting use fmt, guarding with an if log.Enabled(level) {} as necessary.


## Troubleshooting
1. Issue: I wrapped this library's log functions and now `file` does not show the correct file. `"file":"myservice/myloggerwrapper.go:81"`
Solution: When you initialize your logger, you can specify a callstack skip. Example: `logger := logging.New("kvstore").SetCallstackSkip(-1)`
This works on child loggers too.

## License
Copyright 2018, Splunk. All Rights Reserved.
