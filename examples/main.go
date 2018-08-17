package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"sync"

	"github.com/splunk/ssc-observation/logging"
)

func main() {
	ExampleGlobalLogger()
	ExampleNonHttpRequest()
	ExampleServiceRequestLogger()
	ExampleChiServiceRequestLogger()
}

// The global logger is used for code paths that do not have a Context, for
// example at service startup and shutdown.
// This should be relatively rare.
//
// The output from this example is:
//  {"level":"INFO","time":"2018-04-22T20:43:31.013Z","file":"examples/main.go:29","message":"Logged from the global logger","service":"service1","mykey":"myvalue","hostname":"abc1000"}
//  {"level":"INFO","time":"2018-04-22T20:43:31.013Z","file":"examples/main.go:33","message":"message1","service":"service1","hostname":"abc1000"}
//  {"level":"DEBUG","time":"2018-04-22T20:43:31.013Z","file":"examples/main.go:34","message":"message2","service":"service1","hostname":"abc1000"}
func ExampleGlobalLogger() {
	// In service main set the the global logger
	log := logging.New("service1")
	log.Info("Service starting")

	// Optionally set it to be the global logger
	logging.SetGlobalLogger(log)

	// Access the global logger with log.Global()
	log = logging.Global()
	log.Info("message1")
	log.SetLevel(logging.DebugLevel)
	if log.Enabled(logging.DebugLevel) {
		// do something expensive here...
		log.Debug("message2")
	}

	// Call Flush before service exit
	defer log.Flush()
}

// The request logger is going to be the most common usage as it will include a unique requestId
// and other request specific information like tenantId
//
// The output from this example is:
//   {"level":"INFO","time":"2018-07-14T19:44:46.421Z","location":"examples/main.go:86","message":"Starting service","service":"service1","hostname":"abcdef"}
//   {"level":"INFO","time":"2018-07-14T19:44:46.423Z","location":"examples/main.go:70","message":"Running client","service":"client1","hostname":"abcdef"}
//   {"level":"INFO","time":"2018-07-14T19:44:46.424Z","location":"examples/main.go:130","message":"Executing operation1","service":"service1","hostname":"abcdef","param1":"value1","webframework":"http"}
//   {"level":"ERROR","time":"2018-07-14T19:44:46.424Z","location":"examples/main.go:134","message":"Bad request","service":"service1","hostname":"abcdef","param1":"value1","error":"Bad value for param1"}
//   {"level":"INFO","time":"2018-07-14T19:44:46.424Z","location":"examples/main.go:76","message":"Response successful","service":"client1","hostname":"abcdef","statusCode":200}
func ExampleServiceRequestLogger() {
	var wg sync.WaitGroup
	wg.Add(1)

	// Run the service asynchronously and wait for it to start listening
	go serviceMain("localhost:8081", &wg)
	wg.Wait()

	// Run the client
	log := logging.New("client1")
	log.Info("Running client")
	resp, err := http.Get("http://localhost:8081/operation1?param1=value1&webframework=http")
	if err != nil {
		log.Error(err, "Failed request")
		return
	}
	log.Info("Response successful", "statusCode", resp.StatusCode)
}

func serviceMain(hostPort string, wg *sync.WaitGroup) {
	// In the service main set the global logger
	logging.SetGlobalLogger(logging.New("service1"))

	// Use the global logger when not in a request context

	log := logging.Global()
	log.Info("Starting service")

	// Start listening for incoming requests and unblock client
	listener, err := net.Listen("tcp", hostPort)
	if err != nil {
		log.Fatal(err, "The service failed to listen")
	}
	wg.Done()

	// Route http requests

	// The middleware handler for operation1 extracts the context and passes it to the
	// strongly-typed operation1() func.
	operation1HandlerFunc := func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		param1 := r.URL.Query().Get("param1")
		param2 := r.URL.Query().Get("webframework")
		operation1(ctx, param1, param2)
	}

	// TODO: add a tenant handler and connect it in

	// Adapt the handler func to an http.Handler
	var operation1Handler http.Handler
	operation1Handler = http.HandlerFunc(operation1HandlerFunc)

	// Wrap operation1Handler with the request logging handler that will set up
	// request context logging.
	operation1Handler = logging.NewRequestLoggerHandler(logging.Global(), operation1Handler)
	http.Handle("/operation1", operation1Handler)

	err = http.Serve(listener, nil)

	if err != nil {
		log.Error(err, "Exiting service")
	}
}

// Strongly-typed implementation for service1.operation1.
func operation1(ctx context.Context, param1, webframework string) {
	// Get the request logger from ctx
	log := logging.From(ctx)

	log.Info("Executing operation1", "param1", param1, "webframework", webframework)

	// Example error message, note the special handling for err
	err := fmt.Errorf("Bad value for param1")
	log.Error(err, "Bad request", "param1", param1)
}

// Context request tracing can still be used when outside the scope of an http request.
// Simply use log.NewRequestContext() directly.
func ExampleNonHttpRequest() {
	requestId := "" // let the logger create one
	ctx := logging.NewRequestContext(context.Background(), requestId)
	logging.From(ctx).Info("New batch started")
}
