package main

import (
	"context"
	"fmt"
	"io"
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
	http.Handle("/tenant1/operationA", logging.NewRequestLoggerHandler(logging.Global(),
		tracing.NewRequestContextHandler(
			ssctracing.NewHTTPOpentracingHandler(
				http.HandlerFunc(operationAHandler)))))

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

	// The Http Handler should have created a new span and we just need to add to it.
	// Add event to the current span
	span := ssctracing.SpanFromContext(ctx)

	httpClient := &http.Client{}
	// Each of the following client call will trigger the "remote" server to create a new span on their side. If remote server
	// is not responding, no new span is created.
	resp1, err := doCall(ctx, httpClient, http.MethodGet, string("http://"+net.JoinHostPort("localhost", "9092")+"/operationB?param1=value1"), nil)
	if resp1 != nil {
		logger.Info("response code from B", "response code", resp1.StatusCode)
	}
	if err != nil {
		errors.Add(err)
	}

	span.LogKV("event", "call service C", "type", "external service")
	resp2, err := doCall(ctx, httpClient, http.MethodPost, string("http://"+net.JoinHostPort("localhost", "9093")+"/operationC?param1=value1"), nil)
	if resp2 != nil {
		logger.Info("response code from C", "response code", resp2.StatusCode)
	}
	if err != nil {
		errors.Add(err)
	}

	span.LogKV("event", "call service B", "type", "internal service")
	resp3, err := doCall(ctx, httpClient, http.MethodPut, string("http://"+net.JoinHostPort("localhost", "9092")+"/operationB?param1=value1"), nil)
	if resp3 != nil {
		logger.Info("response code from calling service B", "response code", resp3.StatusCode)
	}
	if err != nil {
		errors.Add(err)
	}

	// we have error from any of the calls.
	if errors.Length() > 0 {
		ext.Error.Set(span, true)
		http.Error(w, errors.Error(), http.StatusInternalServerError)
		return
	}
	return
}

func doCall(ctx context.Context, httpClient *http.Client, method, url string, body io.Reader) (*http.Response, error) {
	req, _ := makeRequest(ctx, method, url, body)
	resp, _ := httpClient.Do(req)
	defer func() {
		if resp != nil {
			resp.Body.Close()
		}
	}()
	// Lets assume a http response that is not StatusOK results an error
	if resp != nil {
		err := isStatusNOK(resp.StatusCode)
		return resp, err
	}

	return resp, nil
}

func makeRequest(ctx context.Context, method, url string, body io.Reader) (*http.Request, error) {
	req, err := ssctracing.NewRequest(ctx, method, url, body)
	if method == http.MethodPost {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	// This propagate X-Request-ID to another microservice
	req.Header.Add(tracing.XRequestID, tracing.RequestIDFrom(ctx))
	return req, err
}

func isStatusNOK(statusCode int) error {
	if statusCode != http.StatusOK {
		return fmt.Errorf(http.StatusText(statusCode))
	}
	return nil
}
