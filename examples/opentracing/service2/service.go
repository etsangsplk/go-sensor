package main

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"net/http"
	"sync"
	"time"

	opentracing "github.com/opentracing/opentracing-go"

	"github.com/splunk/ssc-observation/logging"
	// TODO we need a better name than opentracing --> confusing with the standard one.
	ssctracing "github.com/splunk/ssc-observation/tracing/opentracing"
)

const serviceName = "service2"

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

	// For some reason this operation wants to call a local function
	// local function will perform a logical unit of work that warrants a span.
	somelocaloperation(ctx)

	// The Http Handler should have created a new span and we just need to add to it.
	// Add event to the current span
	span := opentracing.SpanFromContext(ctx)
	defer span.Finish()

	// This operation sleep some random time and shoukd show in reporter
	Sleep(time.Duration(1), time.Duration(2))
	// Add event to span
	span.LogKV("event", "delay", "type", "planned deplay")
}

// This function is to show how to propagate the in-process context.
// The Go stardard library usually use `context.Context`, instead of custom type like Span.
//
func somelocaloperation(parentContext context.Context) string {
	logger := logging.Global()

	childSpan, _ := opentracing.StartSpanFromContext(parentContext, "somelocaloperation")
	defer childSpan.Finish()

	// The following has nothing to do with ssc logging, and size is not as neglible as
	// the logging library, so don't treat as such.
	// These "logs" are actually events related to the span, which this case childSpan.
	// They will be serialized and sent to the remote reporter.
	childSpan.LogKV("event", "something useful", "type", "localoperation")

	logger.Info("excuted somelocaloperation")
	return "someresult"
}

func Sleep(mean time.Duration, stdDev time.Duration) {
	delay := time.Duration(math.Max(1, rand.NormFloat64()*float64(stdDev)+float64(mean)))
	time.Sleep(delay)
}
