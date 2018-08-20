package main

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"net/http"
	"sync"
	"time"

	tag "github.com/opentracing/opentracing-go/ext"

	"cd.splunkdev.com/libraries/go-observation/examples/opentracing/handlers"
	"cd.splunkdev.com/libraries/go-observation/logging"
	"cd.splunkdev.com/libraries/go-observation/tracing"
	// TODO we need a better name than opentracing --> confusing with the standard one.
	opentracing "cd.splunkdev.com/libraries/go-observation/opentracing"
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

	// Create, set tracer and bind tracer to service name
	tracer := lightstepx.NewTracer(serviceName)
	defer lightstepx.Close(context.Background())

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
	http.Handle("/operationB", logging.NewRequestLoggerHandler(logging.Global(),
		tracing.NewRequestContextHandler(
			handlers.NewOperationHandler(
				opentracing.NewHTTPOpenTracingHandler(
					http.HandlerFunc(operationBHandler))))))

	logger.Info("ready for handling requests")
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
	log.Info("Executing operation", "operation", "B")

	tenantID := tracing.TenantIDFrom(ctx)
	ret, err := queryDatabase(ctx, tenantID)
	_, _ = w.Write([]byte(ret))
	// Note: subscriber notification is not contributing to operation B, so no new span
	// is created.
	err1 := notifySubscriber(ctx, ret)
	log.Error(err1, "subscriber notification error")

	if err != nil {
		http.Error(w, err1.Error(), http.StatusInternalServerError)
	}
}

// queryDatabase queries a fake database for some data that is crucial for completion of operation.
// Assume that we also want to know the database information for the span.
func queryDatabase(ctx context.Context, tenantID string) (string, error) {
	var err error
	logger := logging.Global()

	// A new span for queryDatabase function assuming that it is significant to operation B.
	// Span is done when this function is over. Note that it includes
	// calling a fake DB plus the sleeping function for this example.
	span, _ := opentracing.StartSpanFromContext(ctx, "queryDatabase")
	defer func() {
		if span != nil {
			span.Finish()
		}
	}()

	logger.Info("excuted queryCustomerDatabase")
	spanLogger := opentracing.NewSpanLoggerWithSpan(logger, span)
	// We are a client calling the database server so set so.
	tag.SpanKindRPCClient.Set(span)
	tag.PeerService.Set(span, "mysql")
	span.SetTag("sql.query", fmt.Sprintf("SELECT tenant from Customer where tenantID=%v", tenantID))

	if tenantID == "" {
		err = fmt.Errorf("tenantID is empty")
	}
	// This operation sleep some random time and should show in reporter
	Sleep(time.Duration(1), time.Duration(2))

	spanLogger.Info("high database response latency observed", "event", "delay", "type", "planned")

	return "someresult", err
}

// notifySubscriber sends the result to subscriber. This function does not
// contribute anything to the completion of the calling operation functionality.
// Hence there is no new span being created. Assume that we still want
// to know that when this notification results in error.
// we are going to "Log" this event to the Span. This event logging is completely from
// ssc logging to a file.
func notifySubscriber(ctx context.Context, result string) error {
	logger := logging.Global()
	span := opentracing.SpanFromContext(ctx)
	spanLogger := opentracing.NewSpanLoggerWithSpan(logger, span)
	err := func() error {
		logger.Info("notifying subscriber", "result", "notify_with_result")
		return fmt.Errorf("server connected disconnected too many times")
	}()
	spanLogger.Error(err, "subscriber notification system error")
	logger.Info("executed notifySubscriber")
	return err
}

func Sleep(mean time.Duration, stdDev time.Duration) {
	delay := time.Duration(math.Max(1, rand.NormFloat64()*float64(stdDev)+float64(mean)))
	time.Sleep(delay)
}
