package main

import (
	"fmt"
	"net/http"
	"sync"

	opentracing "github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"

	"github.com/splunk/ssc-observation/logging"
	ssctracing "github.com/splunk/ssc-observation/tracing/opentracing"
)

const serviceName = "service3"

func main() {
	// Routine initialization of logger and tracer
	// We just need 1 tracer per service initialized with
	// the service name. This is important because
	// when we need to to query service dependencies graph,
	// service name for tracer initialization is what will be looked at.
	logger := logging.New(serviceName)
	logging.SetGlobalLogger(logger)

	// Create, set tracer and bind tracer to service name
	tracer, closer := ssctracing.NewTracer(serviceName, logger)
	defer closer.Close()
	ssctracing.SetGlobalTracer(tracer)

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

	// Start listening for incoming requests and unblock client
	//listener, err := net.Listen("tcp", hostPort)
	//if err != nil {
	//	logger.Fatal(err, fmt.Sprintf("Service %s failed to listen", serviceName))
	//}

	// Configure Route http requests
	// Service A operationA calls serviceB then serviceC which errors out at the end
	http.Handle("/operationC", logging.NewRequestLoggerHandler(logging.Global(),
		ssctracing.NewHTTPOpentracingHandler(http.HandlerFunc(operationCHandler))))
	logger.Info("ready for handling requests")
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
	log.Info("Executing operation", "operation", "C")

	// The Http Handler should have created a new span and we just need to add to it.
	// Add event to the current span
	childSpan := opentracing.SpanFromContext(ctx)
	// This operation will error out and should show in reporter.
	err := func() error { return fmt.Errorf("failed operationC") }()
	if err != nil {
		ext.Error.Set(childSpan, true)
	}
	// Add event to span
	childSpan.LogKV("event", "error", "type", "server error", "error", err.Error())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}