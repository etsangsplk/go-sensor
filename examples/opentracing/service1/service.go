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

	resp1, _ := client.Get(string("http://" + net.JoinHostPort("localhost", "9092") + "/operationB?param1=value1"))
	defer func() {
		if resp1 != nil {
			resp1.Body.Close()
		}
	}()

	// Lets assume a http response that is not StatusOK results an error
	err1 := isStatusNOK(resp1.StatusCode)
	if err1 != nil {
		errors.Add(err1)
	}

	span.LogKV("event", "call service B", "type", "external service")
	if resp1 != nil {
		logger.Info("response code from B", "response code", resp1.StatusCode)
	}

	resp2, _ := client.Post(string("http://"+net.JoinHostPort("localhost", "9093")+"/operationC?param1=value1"), "application/x-www-form-urlencoded", nil)
	defer func() {
		if resp2 != nil {
			resp2.Body.Close()
		}
	}()

	// Lets assume a http response that is not StatusOK results an error
	err2 := isStatusNOK(resp2.StatusCode)
	if err2 != nil {
		errors.Add(err2)
	}

	span.LogKV("event", "call service C", "type", "internal service")
	if resp2 != nil {
		logger.Info("response code from C", "response code", resp2.StatusCode)
	}

    httpClient := &http.Client{}
    newReq, _ := ssctracing.NewRequest(ctx, http.MethodPost, string("http://" + net.JoinHostPort("localhost", "9092") + "/operationB?param1=value1"), nil)
    resp3, _ := httpClient.Do(newReq)
    if resp3 != nil {
        logger.Info("response code from calling google", "response code", resp3.StatusCode)
    }
    err4 := isStatusNOK(resp3.StatusCode)
    if err4 != nil {
        errors.Add(err4)
    }

	// we have error from any of the calls.
	if errors.Length() > 0 {
		ext.Error.Set(span, true)
		w.WriteHeader(http.StatusInternalServerError)
		http.Error(w, errors.Error(), http.StatusInternalServerError)
	} else {
		w.WriteHeader(http.StatusOK)
	}
}

func isStatusNOK(statusCode int) error {
	if statusCode != http.StatusOK {
		return fmt.Errorf(http.StatusText(statusCode))
	}
	return nil
}
