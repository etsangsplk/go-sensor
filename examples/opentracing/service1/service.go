package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"sync"

	"github.com/cloudfoundry/multierror"
	opentracing "github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"

	"github.com/splunk/ssc-observation/logging"
	ssctracing "github.com/splunk/ssc-observation/tracing/opentracing"
)

const serviceName = "service1"

func main() {
	// Routine initialization of logger and tracer
	// We just need 1 tracer per service initialized with
	// the service name. This is important because
	// when we need to to query service dependencies graph,
	// service name for tracer initialization is what will be looked at.
	logger := logging.New(serviceName)
	logging.SetGlobalLogger(logger)

	// Create, set tracer and bind tracer to service name
	tracer, closer := ssctracing.NewTracer(serviceName, ssctracing.NewLogger(logger))
	defer closer.Close()
	ssctracing.SetGlobalTracer(tracer)

	var wg sync.WaitGroup
	wg.Add(1)
	go Service(net.JoinHostPort("localhost", "9091"), &wg)
	wg.Wait()

	// Run the Client
	span := tracer.StartSpan("testclient")
	span.SetOperationName("testend2end")
	defer span.Finish()

	log := logging.New("client1")
	log.Info("Running client")

	topSpanContext := opentracing.ContextWithSpan(context.Background(), span)
	httpClient := ssctracing.NewHTTPClient(topSpanContext, tracer)

	resp, err := httpClient.Get("http://localhost:9091/operationA?param1=value1")
	if err != nil {
		log.Error(err, "Failed request")
		return
	}
	log.Info("Response successful", "statusCode", resp.StatusCode)
}

// Simulated microservice A, serving requests.
func Service(hostPort string, wg *sync.WaitGroup) {
	logger := logging.Global()
	logger.Info(fmt.Sprintf("Starting service %s", serviceName))

	// Start listening for incoming requests and unblock client
	listener, err := net.Listen("tcp", hostPort)
	if err != nil {
		logger.Fatal(err, fmt.Sprintf("Service %s failed to listen", serviceName))
	}
	wg.Done()

	// Configure Route http requests
	// Service A operationA calls serviceB then serviceC which errors out at the end
	http.Handle("/operationA", ssctracing.NewHTTPOpentracingHandler(http.HandlerFunc(operationAHandler)))

	err = http.Serve(listener, nil)
	if err != nil {
		logger.Error(err, fmt.Sprintf("Exiting service %s", serviceName))
	}
	logger.Info("ready for handling requests")
}

func operationAHandler(w http.ResponseWriter, r *http.Request) {
	errors := multierror.MultiError{}

	param1 := r.URL.Query().Get("param1")
	// Get the request logger from ctx

	ctx := r.Context()
	logger := logging.From(ctx)
	logger.Info("Executing operation", "operation", "A", "param1", param1)

	// Get the tracer for this service
	tracer := ssctracing.Global()
	client := ssctracing.NewHTTPClient(ctx, tracer)
	// The Http Handler should have created a new span and we just need to add to it.
	// Add event to the current span
	span := opentracing.SpanFromContext(ctx)
	defer func() {
		if span != nil {
			span.Finish()
		}
	}()

	resp, err1 := client.Get(string("http://" + net.JoinHostPort("localhost", "9092") + "/operationB?param1=value1"))
	if err1 != nil {
		errors.Add(err1)
	}

	span.LogKV("event", "call service B", "type", "external service")
	if resp != nil {
		logger.Info("service", "B", "response", resp.StatusCode)
		ext.HTTPStatusCode.Set(span, uint16(resp.StatusCode))
	}
	logger.Error(err1, "error from calling service B")

	resp, err2 := client.Post(string("http://"+net.JoinHostPort("localhost", "9093")+"/operationC?param1=value1"), "application/x-www-form-urlencoded", nil)
	if err2 != nil {
		errors.Add(err1)
	}
	span.LogKV("event", "call service C", "type", "internal service")
	if resp != nil {
		logger.Info("service", "C", "response", resp.StatusCode)
		ext.HTTPStatusCode.Set(span, uint16(resp.StatusCode))
	}

	logger.Error(err2, "error from calling service C")

	// we have error
	if errors.Length() > 0 {
		ext.Error.Set(span, true)
	}
}
