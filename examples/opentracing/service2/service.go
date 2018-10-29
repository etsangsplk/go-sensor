package main

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"net/http"
	"sync"
	"time"

	ot "github.com/opentracing/opentracing-go"
	tag "github.com/opentracing/opentracing-go/ext"

	"cd.splunkdev.com/libraries/go-observation/examples/opentracing/handlers"
	"cd.splunkdev.com/libraries/go-observation/logging"
	"cd.splunkdev.com/libraries/go-observation/tracing"
	// TODO we need a better name than opentracing --> confusing with the standard one.
	opentracing "cd.splunkdev.com/libraries/go-observation/opentracing"
	"cd.splunkdev.com/libraries/go-observation/opentracing/instanax"
	"cd.splunkdev.com/libraries/go-observation/opentracing/jaegerx"
	"cd.splunkdev.com/libraries/go-observation/opentracing/lightstepx"
)

const serviceName = "example-customer-catalog"

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
		tracer, closer, err := jaegerx.NewTracer(serviceName)
		logger.Fatal(err, "fail to initialize jaeger")
		defer jaegerx.Close(closer, context.Background())
	}

	opentracing.SetGlobalTracer(tracer)

	var wg sync.WaitGroup
	wg.Add(1)
	go Service(":9092", &wg)
	wg.Wait()
}

func Service(hostPort string, wg *sync.WaitGroup) {
	logger := logging.Global()
	logger.Info(fmt.Sprintf("Starting service %s", serviceName))
	// Configure Route http requests
	http.Handle("/tenant1/operationB",
		logging.NewRequestLoggerHandler(logging.Global(),
			tracing.NewRequestContextHandler(
				handlers.NewOperationHandler(
					opentracing.NewHTTPOpenTracingHandler(
						http.HandlerFunc(operationBHandler))))))

	logger.Info("Listening...", "port", hostPort)
	err := http.ListenAndServe(hostPort, nil)
	wg.Done()
	if err != nil {
		logger.Error(err, fmt.Sprintf("Exiting service %s", serviceName))
	}
}

func operationBHandler(w http.ResponseWriter, r *http.Request) {
	// Get the request logger from ctx
	ctx := r.Context()
	log := logging.From(ctx)
	tenantID := tracing.TenantIDFrom(ctx)
	log.Info("Handling request", "operation", "operationB", "tenant", tenantID)
	ret, err := queryDatabase(ctx, tenantID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Error(err, "queryDatabase error")
		return
	}
	_, _ = w.Write([]byte(ret))

	// Note: subscriber notification is not contributing to operation B, so no new span
	// is created.
	err = notifySubscriber(ctx, ret)
	if err != nil {
		log.Error(err, "Subscriber notification error")
	}
}

// queryDatabase queries a fake database for some data that is crucial for completion of operation.
// Assume that we also want to know the database information for the span.
func queryDatabase(ctx context.Context, tenantID string) (string, error) {
	var err error
	log := logging.From(ctx)

	// A new span for queryDatabase function assuming that it is significant to operation B.
	// Span is done when this function is over. Note that it includes
	// calling a fake DB plus the sleeping function for this example.
	span, _ := opentracing.StartSpanFromContext(ctx, "queryDatabase")
	defer span.Finish()

	log.Info("Excuted queryCustomerDatabase")
	tag.SpanKindRPCClient.Set(span)
	tag.PeerService.Set(span, "mysql")
	span.SetTag("sql.query", fmt.Sprintf("SELECT tenant from Customer where tenantID=%v", tenantID))

	if tenantID == "" {
		err = fmt.Errorf("tenantID is empty")
	}
	// This operation sleep some random time and should show in reporter
	Sleep(time.Duration(1), time.Duration(2))
	span.LogKV("message", "High database response latency observed", "event", "delay", "type", "planned")

	return "someresult", err
}

// notifySubscriber sends the result to subscriber. This function does not
// contribute anything to the completion of the calling operation functionality.
// Hence there is no new span being created. Assume that we still want
// to know that when this notification results in error.
// we are going to "Log" this event to the Span. This event logging is completely from
// ssc logging to a file.
func notifySubscriber(ctx context.Context, result string) error {
	log := logging.From(ctx)
	span := opentracing.SpanFromContext(ctx)
	err := fmt.Errorf("Server connected disconnected too many times (intentional failure to demonstrate an error)")
	span.LogKV("message", "Subscriber notification system error", "event", "error", "error", err.Error())
	log.Error(err, "Subscriber notified")
	return err
}

func Sleep(mean time.Duration, stdDev time.Duration) {
	delay := time.Duration(math.Max(1, rand.NormFloat64()*float64(stdDev)+float64(mean)))
	time.Sleep(delay)
}
