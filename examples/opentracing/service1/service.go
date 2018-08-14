package main

import (
	//"context"
	"fmt"
	"net"
	"net/http"
	"sync"

	"github.com/cloudfoundry/multierror"
	"github.com/opentracing/opentracing-go/ext"

	"github.com/splunk/ssc-observation/logging"
	"github.com/splunk/ssc-observation/tracing"
	ssctracing "github.com/splunk/ssc-observation/tracing/opentracing"
)

const serviceName = "api-gateway"

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
	go Service(":9091", &wg)
	wg.Wait()
}

// Simulated microservice A, serving requests.
func Service(hostPort string, wg *sync.WaitGroup) {
	logger := logging.Global()
	logger.Info(fmt.Sprintf("Starting service %s", serviceName))

	// Configure Route http requests
	// Service A operationA calls serviceB then serviceC which errors out at the end
	http.Handle("/tenant1/operationA", logging.NewRequestLoggerHandler(logging.Global(),
		tracing.NewRequestContextHandler(
			ssctracing.NewHTTPOpentracingHandler(http.HandlerFunc(operationAHandler)))))

	logger.Info("ready for handling requests")
	err := http.ListenAndServe(hostPort, nil)
	wg.Done()

	if err != nil {
		logger.Error(err, fmt.Sprintf("Exiting service %s", serviceName))
	}

}

func operationAHandler(w http.ResponseWriter, r *http.Request) {
	errors := multierror.MultiError{}

	param1 := r.URL.Query().Get("param1")
	// Get the request logger from ctx

	ctx := r.Context()
	logger := logging.From(ctx)
	logger.Info("Executing operation", "operation", "A", "param1", param1)

	// Get the tracer for this service
	client := ssctracing.NewHTTPClient(ctx)
	// The Http Handler should have created a new span and we just need to add to it.
	// Add event to the current span
	span := ssctracing.SpanFromContext(ctx)

	resp, err1 := client.Get(string("http://" + net.JoinHostPort("localhost", "9092") + "/operationB?param1=value1"))
	if err1 != nil {
		errors.Add(err1)
	}

	span.LogKV("event", "call service B", "type", "external service")
	if resp != nil {
		logger.Info("response code from B", "response code", resp.StatusCode)
		ext.HTTPStatusCode.Set(span, uint16(resp.StatusCode))
	}

	resp, err2 := client.Post(string("http://"+net.JoinHostPort("localhost", "9093")+"/operationC?param1=value1"), "application/x-www-form-urlencoded", nil)
	if err2 != nil {
		errors.Add(err1)
	}
	span.LogKV("event", "call service C", "type", "internal service")
	if resp != nil {
		logger.Info("response code from C", "response code", resp.StatusCode)
		ext.HTTPStatusCode.Set(span, uint16(resp.StatusCode))
	}

	// we have error from any of the calls.
	if errors.Length() > 0 {
		ext.Error.Set(span, true)
		ext.HTTPStatusCode.Set(span, uint16(http.StatusInternalServerError))
		http.Error(w, errors.Error(), http.StatusInternalServerError)
	}
	// else Ok.
	ext.HTTPStatusCode.Set(span, uint16(http.StatusOK))
}
