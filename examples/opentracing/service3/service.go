package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"

	ot "github.com/opentracing/opentracing-go"

	"cd.splunkdev.com/libraries/go-observation/examples/opentracing/handlers"
	"cd.splunkdev.com/libraries/go-observation/logging"
	opentracing "cd.splunkdev.com/libraries/go-observation/opentracing"
	"cd.splunkdev.com/libraries/go-observation/opentracing/instanax"
	"cd.splunkdev.com/libraries/go-observation/opentracing/lightstepx"
	"cd.splunkdev.com/libraries/go-observation/tracing"
)

const serviceName = "example-fulfillment"

func main() {
	// Routine initialization of logger and tracer
	// We just need 1 tracer per service initialized with
	// the service name. This is important because
	// when we need to to query service dependencies graph,
	// service name for tracer initialization is what will be looked at.
	logger := logging.New(serviceName)
	logging.SetGlobalLogger(logger)

	var tracer ot.Tracer
	// Create, set tracer and bind tracer to service name
	if lightstepx.Enabled() && instana.Enabled() {
		logger.Fatal(errors.New("cannot enable both Lighstep and Instana"), "use either Lightstep or Instana")
	}
	if lightstepx.Enabled() {
		tracer = lightstepx.NewTracer(serviceName)
		defer lightstepx.Close(context.Background())
	}
	if instana.Enabled() {
		tracer = instanax.NewTracer(serviceName)
		defer instanax.Close(context.Background())
	}

	opentracing.SetGlobalTracer(tracer)

	var wg sync.WaitGroup
	wg.Add(1)
	go Service(":9093", &wg)
	wg.Wait()

	logger.Info(fmt.Sprintf("Starting service %s", serviceName))

}

// Simulated microservice A, serving requests.
func Service(hostPort string, wg *sync.WaitGroup) {
	logger := logging.Global()
	logger.Info(fmt.Sprintf("Starting service %s", serviceName))

	// Configure Route http requests
	http.Handle("/operationC",
		logging.NewRequestLoggerHandler(logging.Global(),
			tracing.NewRequestContextHandler(
				handlers.NewOperationHandler(
					opentracing.NewHTTPOpenTracingHandler(
						http.HandlerFunc(operationCHandler))))))
	logger.Info("Listening...", "port", hostPort)
	err := http.ListenAndServe(hostPort, nil)
	wg.Done()
	if err != nil {
		logger.Error(err, fmt.Sprintf("Exiting service %s", serviceName))
	}
}

func operationCHandler(w http.ResponseWriter, r *http.Request) {
	// Get the request logger from ctx
	ctx := r.Context()
	log := logging.From(ctx)
	log.Info("Handling request", "operation", "operationC")

	// The Http Handler should have created a new span and we just need to add to it.
	// Add event to the current span
	childSpan := opentracing.SpanFromContext(ctx)
	// This operation will error out and should show in reporter.
	err := fmt.Errorf("faileed serviceC.operationC")
	if err != nil {
		// Add event to span
		childSpan.LogKV("message", "operationC error", "type", "server", "event", "error", "error", err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
