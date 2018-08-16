package opentracing_test

import (
	"context"

	"github.com/splunk/ssc-observation/logging"
	ssctracing "github.com/splunk/ssc-observation/tracing/opentracing"
)

func Example_NewTracer() {
	logger := logging.New("Test_NewTracer")

	// Create, set tracer and bind tracer to service name
	tracer, closer := ssctracing.NewTracer("myservice", logger)
	defer closer.Close()
	ssctracing.SetGlobalTracer(tracer)
}

func Example_StartSpan() {
	logger := logging.New("Test_StartSpan")
	// Create, set tracer and bind tracer to service name
	// Not doing anything with returned tracer
	_, closer := ssctracing.NewTracer("myservice", logger)
	defer closer.Close()
	span := ssctracing.StartSpan("operationName")
	defer func() {
		if span != nil {
			span.Finish()
		}
	}()
}

func Example_StartSpanFromContext() {
	logger := logging.New("Test_StartSpanFromContext")

	// Create, set tracer and bind tracer to service name
	tracer, closer := ssctracing.NewTracer("myservice", logger)
	defer closer.Close()
	span := tracer.StartSpan("operationAName")
	defer func() {
		if span != nil {
			span.Finish()
		}
	}()

	// context.Background is just context and not span context, so create a new span.
	anotherSpan, anotherSpanContext := ssctracing.StartSpanFromContext(context.Background(), "operationBName")
	defer func() {
		if anotherSpan != nil {
			anotherSpan.Finish()
		}
	}()

	// anotherSpanContext is a span context, so below will creae a child span of anotherSpan
	childSpan, _ := ssctracing.StartSpanFromContext(anotherSpanContext, "operationCName")
	defer func() {
		if childSpan != nil {
			childSpan.Finish()
		}
	}()
}
