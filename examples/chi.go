package main

import (
	"fmt"
	"net"
	"net/http"
	"sync"

	"github.com/go-chi/chi"

	"github.com/splunk/ssc-observation/logging"
	"github.com/splunk/ssc-observation/metrics"
	"github.com/splunk/ssc-observation/tracing"
)

// ExampleChiServiceRequestLogger is the same example as in main.go but
// implemented using go-chi/chi
//
// The output from this example is:
//   {"level":"INFO","time":"2018-07-14T18:03:01.509Z","location":"examples/chi.go:39","message":"Starting Chi service","service":"service1","hostname":"abcdef"}
//   {"level":"INFO","time":"2018-07-14T18:03:01.509Z","location":"examples/chi.go:27","message":"Running client for service built with Chi","service":"http Chi client1","hostname":"abcdef"}
//   {"level":"INFO","time":"2018-07-14T18:03:01.511Z","location":"examples/main.go:129","message":"Executing operation1","service":"service1","hostname":"abcdef","param1":"value1","webframework":"go-chi/chi"}
//   {"level":"ERROR","time":"2018-07-14T18:03:01.511Z","location":"examples/main.go:133","message":"Bad request","service":"service1","hostname":"abcdef","param1":"value1","error":"Bad value for param1"}
//   {"level":"INFO","time":"2018-07-14T18:03:01.511Z","location":"examples/chi.go:33","message":"Response successful","service":"http Chi client1","hostname":"abcdef","statusCode":200}
func ExampleChiServiceRequestLogger() {
	var wg sync.WaitGroup
	wg.Add(1)
	const localDomain = "localhost:8082"
	// Run the service asynchronously and wait for it to start listening
	go serviceChiMain(localDomain, &wg)
	wg.Wait()

	// Run the client
	log := logging.New("http Chi client1")
	log.Info("Running client for service built with Chi")
	resp, err := http.Get(fmt.Sprintf("http://%v/operation1?param1=value1&webframework=%v", localDomain, "go-chi/chi"))
	if err != nil {
		log.Error(err, "Failed request")
		return
	}
	log.Info("Response successful", "statusCode", resp.StatusCode)
}

func serviceChiMain(hostPort string, wg *sync.WaitGroup) {
	// Use the global logger when not in a request context
	log := logging.Global()
	log.Info("Starting Chi service")

	listener, err := net.Listen("tcp", hostPort)
	if err != nil {
		log.Fatal(err, "The service failed to listen")
	}
	wg.Done()

	// Route http requests
	r := chi.NewRouter()
	// Or if you are using NewMux()
	// r := chi.NewMux()
	if r == nil {
		log.Fatal(fmt.Errorf("failed to initialize chi router"), "The service failed to initialize")
	}

	// The middleware handler for operation1 extracts the context and passes it to the
	// strongly-typed operation1HandlerFunc.

	operation1HandlerFunc := &ChiOperation1EndpointHandler{}

	// Setup Chi router with set of http handler.

	metrics.RegisterHTTPMetrics("chiexample")
	// Wrap operation1Handler with the request logging handler that will set up
	// request context logging.

	// You can use
	// r.Use(logging.NewRequestLoggerHandlerAdaptor(logging.Global()))
	// Or if you prefer to chain it yourself more explicitly
	r.Use(func(next http.Handler) http.Handler {
		return logging.NewRequestLoggerHandler(logging.Global(),
			operation1HandlerFunc)
	})
	r.Use(logging.NewPanicHandler)
	r.Use(metrics.NewPrometheusHandler)
	r.Use(tracing.NewRequestContextHandler)
	r.Use(logging.NewRequestLoggerHandlerAdaptor(log))
	r.Use(logging.NewPanicRequestHandler)
	r.Use(metrics.NewHTTPAccessHandler)
	r.Use(logging.NewHTTPAccessHandler)

	r.Handle("/operation1", operation1HandlerFunc)

	// Server listen for traffic and serve.
	err = http.Serve(listener, r)
	if err != nil {
		log.Error(err, "Exiting service served by Chi")
	}
}

// Sample http handler for Chi framework
type ChiOperation1EndpointHandler struct {
	http.Handler
}

func (e *ChiOperation1EndpointHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	param1 := r.URL.Query().Get("param1")
	webframework := r.URL.Query().Get("webframework")
	operation1(ctx, param1, webframework)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(fmt.Sprintf("hi %s", param1)))
}
