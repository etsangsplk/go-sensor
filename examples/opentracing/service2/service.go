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

	"github.com/splunk/ssc-observation/logging"
	"github.com/splunk/ssc-observation/tracing"
	// TODO we need a better name than opentracing --> confusing with the standard one.
	ssctracing "github.com/splunk/ssc-observation/tracing/opentracing"
)

const serviceName = "customer-catalog"

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
	go Service(":9092", &wg)
	wg.Wait()
}

func Service(hostPort string, wg *sync.WaitGroup) {
	logger := logging.Global()
	logger.Info(fmt.Sprintf("Starting service %s", serviceName))
	// Configure Route http requests
	// Service A operationA calls serviceB then serviceC which errors out at the end
	http.Handle("/operationB", logging.NewRequestLoggerHandler(logging.Global(),
		ssctracing.NewHTTPOpentracingHandler(http.HandlerFunc(operationBHandler))))

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
	ret := queryDatabase(ctx, fmt.Sprintf("SELECT tenant from Customer where tenantID=%v", tenantID))
	notifySubscriber(ctx, ret)
	w.Write([]byte(ret))
	w.WriteHeader(http.StatusOK)
}

// queryDatabase queries a fake database for some data that is crucial for completion of operation.
// Assume that we also want to know the database information for the span.
func queryDatabase(ctx context.Context, statment string) string {
	logger := logging.Global()

	// A new span for local function, ignoring the returned context from this
	// operation for this example, since we are not propogating to another level
	// in tthis example. But if you do need to propagate, you need to return back and
	// wrap this as span context and into the request context.
	span, _ := ssctracing.StartSpanFromContext(ctx, "queryDatabase")
	defer func() {
		if span != nil {
			span.Finish()
		}
	}()

	logger.Info("excuted queryDatabase")
	// We are a client calling the database server so set so.
	tag.SpanKindRPCClient.Set(span)
	tag.PeerService.Set(span, "mysql")
	span.SetTag("sql.query", statment)
	// This operation sleep some random time and should show in reporter
	Sleep(time.Duration(1), time.Duration(2))
	span.LogKV("event", "delay", "type", "planned db delay")
	return "someresult"
}

// notifySubscriber sends the result to subscriber. This function does not
// contribute anything to the completion of the calling operation functionality.
// Hence there is no new span being created. Assume that we still want
// to know that when this notification results in error.
// we are going to "Log" this event to the Span. This event logging is completely from
// ssc logging to a file.
func notifySubscriber(ctx context.Context, result string) error {
	logger := logging.Global()
	span := ssctracing.SpanFromContext(ctx)
	ret := "notify_with_result"
	err := func() error {
		logger.Info("notifying subscriber", "result", ret)
		return fmt.Errorf("server connected disconnected too many times")
	}()
	span.LogKV("event", "error", "message", err.Error())
	logger.Info("excuted notifySubscriber")
	return err
}

func Sleep(mean time.Duration, stdDev time.Duration) {
	delay := time.Duration(math.Max(1, rand.NormFloat64()*float64(stdDev)+float64(mean)))
	time.Sleep(delay)
}
