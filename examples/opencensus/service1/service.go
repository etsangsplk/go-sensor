package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"path"
	"sync"

	"go.opencensus.io/exporter/prometheus"
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
	"go.opencensus.io/trace"
	"go.opencensus.io/zpages"
	"go.opencensus.io/exporter/jaeger"
    jaegerConfig "github.com/uber/jaeger-client-go/config"
    
	"cd.splunkdev.com/libraries/go-observation/examples/opentracing/handlers"
	"cd.splunkdev.com/libraries/go-observation/logging"
	//	opentracing "cd.splunkdev.com/libraries/go-observation/opentracing"
	//	"cd.splunkdev.com/libraries/go-observation/opentracing/instanax"
	//	"cd.splunkdev.com/libraries/go-observation/opentracing/jaegerx"
	//	"cd.splunkdev.com/libraries/go-observation/opentracing/lightstepx"
	"cd.splunkdev.com/libraries/go-observation/tracing"
)

const serviceName = "example-api-gateway"

var (
	projectID    = os.Getenv("GOOGLE_PROJECT_ID")
	jaegerConfig = struct {
		host string
		port string
	}{
		host: os.Getenv("JAEGER_HOST"),
		port: os.Getenv("JAEGER_PORT")}
)

var (
	service2Host = "service2"
	service3Host = "service3"

	service1Port = "9091"
	service2Port = "9092"
	service3Port = "9093"
)

func main() {
	// Routine initialization of logger and tracer
	// We just need 1 tracer per service initialized with
	// the service name. This is important because
	// when we need to to query service dependencies graph,
	// service name for tracer initialization is what will be looked at.
	logger := logging.New(serviceName)
	logging.SetGlobalLogger(logger)

	// To run outside of docker-compose uncomment these lines
	// service2Host = "localhost"
	// service3Host = "localhost"

	tracer := opentracing.Global()
	// Create, set tracer and bind tracer to service name
	// TODO change to switch statements
	if lightstepx.Enabled() && instanax.Enabled() {
		logger.Fatal(errors.New("cannot enable both Lighstep and Instana"), "use either Lightstep or Instana")
	}
	if lightstepx.Enabled() {
		tracer = lightstepx.NewTracer(serviceName)
		defer lightstepx.Close(context.Background())
	}
	if instanax.Enabled() {
		tracer = instanax.NewTracer(serviceName)
		defer instanax.Close(context.Background())
	}
	if jaegerx.Enabled() {
		t, closer, err := jaegerx.NewTracer(serviceName)
		tracer = t // get around variable shadowing.
		logger.Fatal(err, "fail to initialize jaeger")
		defer jaegerx.Close(closer, context.Background())
	}

	exporter, err := initExporter(serviceName)
	if err != nil {
		logger.Fatal(errors.New("cannot initialize exporter"), "exporter_type", "jaeger")
	}
	defer exporter.Flush()

	// opentracing.SetGlobalTracer(tracer)

	var wg sync.WaitGroup
	wg.Add(1)
	go Service(":"+service1Port, &wg)
	wg.Wait()
}

// Simulated microservice A, serving requests.
func Service(hostPort string, wg *sync.WaitGroup) {
	logger := logging.Global()
	logger.Info(fmt.Sprintf("Starting service %s", serviceName))

	// Configure Route http requests
	http.Handle("/tenant1/operationA",
		logging.NewRequestLoggerHandler(logging.Global(),
			tracing.NewRequestContextHandler(
				handlers.NewOperationHandler(
					opentracing.NewHTTPOpenTracingHandler(
						http.HandlerFunc(operationAHandler))))))

	logger.Info("Listening...", "port", hostPort)
	err := http.ListenAndServe(hostPort, nil)
	wg.Done()

	if err != nil {
		logger.Error(err, fmt.Sprintf("Exiting service %s", serviceName))
	}

}

func operationAHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	ctx = tracing.WithOperationID(ctx, "operationA")
	log := logging.From(ctx)
	param1 := r.URL.Query().Get("param1")
	log.Info("Handling request", "operation", "operationA", "param1", param1)
	err := operationA(ctx, param1)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func operationA(ctx context.Context, param1 string) error {
	var err error
	log := logging.From(ctx)

	transport := opentracing.NewTransportWithRoundTripper(opentracing.NewHandlerList(), http.DefaultTransport)
	httpClient := &http.Client{Transport: transport}

	// Each of the following client call will trigger the "remote" server to create a new span on their side. If remote server
	// is not responding, no new span is created.
	err = service2OperationB(ctx, httpClient, "value1")
	if err != nil {
		log.Error(err, "Error from operationB")
		return err
	}

	err = service3OperationC(ctx, httpClient, "value1")
	if err != nil {
		log.Error(err, "Error from operationC")
		return err
	}
	return nil
}

func service2OperationB(ctx context.Context, httpClient *http.Client, param1 string) error {
	hostPort := net.JoinHostPort(service2Host, service2Port)
	urlPath := "/tenant1/operationB?param1=" + param1
	ctx = tracing.WithOperationID(ctx, "operationB")
	_, err := doCall(ctx, httpClient, http.MethodGet, "operationB", hostPort, urlPath, nil)
	return err
}

func service3OperationC(ctx context.Context, httpClient *http.Client, param1 string) error {
	hostPort := net.JoinHostPort(service3Host, service3Port)
	urlPath := "/operationC?param1=" + param1
	ctx = tracing.WithOperationID(ctx, "operationC")
	_, err := doCall(ctx, httpClient, http.MethodPost, "operationC", hostPort, urlPath, nil)
	return err
}

func doCall(ctx context.Context, httpClient *http.Client, method, operation, hostPort, urlPath string, body io.Reader) (*http.Response, error) {
	// TODO: this pattern of handling the span at the http I/O layer may not be viable since the
	//     : decision to interpret a response code as an error requires application layer logic
	span, ctx := opentracing.StartSpanFromContext(ctx, operation)
	defer span.Finish()

	url := "http://" + path.Join(hostPort, urlPath)
	span.LogKV("event", "HTTP client call", "type", "external service", "url", url, "operation", operation)
	req, _ := makeRequest(ctx, method, url, body)
	resp, err := httpClient.Do(req)
	if err != nil {
		opentracing.SetSpanError(span)
		return nil, err
	}

	defer resp.Body.Close()
	span.LogKV("event", "HTTP response", "statusCode", resp.StatusCode)

	// Lets assume a http response that is <400 is success
	if err = isStatusSuccess(resp); err != nil {
		opentracing.SetSpanError(span)
	}
	return resp, err
}

func makeRequest(ctx context.Context, method, url string, body io.Reader) (*http.Request, error) {
	req, err := newRequest(ctx, method, url, body)
	if method == http.MethodPost {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	// This propagate X-Request-ID to another microservice
	req.Header.Add(tracing.XRequestID, tracing.RequestIDFrom(ctx))
	return req, err
}

func isStatusSuccess(resp *http.Response) error {
	statusCode := resp.StatusCode
	if statusCode < 400 {
		return nil
	}

	return fmt.Errorf(http.StatusText(statusCode))
}

// newRequest returns a new request from upstream ctx context.
func newRequest(ctx context.Context, method string, url string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}
	// This propagate X-Request-ID to another microservice
	req.Header.Add(tracing.XRequestID, "abcde")
	req = opentracing.InjectHTTPRequestWithSpan(req.WithContext(ctx))
	return req, err
}

func initExporter(ctx context.Context, serviceName string) (trace.Exporter, error) {
	jaegerURL := fmt.Sprintf("http://%v:%v", jaegerConfig.host, jaegerConfig.port)

	exporter, err := jaeger.NewExporter(jaeger.Options{
		Endpoint:    jaegerURL,
		ServiceName: serviceName,
	})
	return exporter, err
}
