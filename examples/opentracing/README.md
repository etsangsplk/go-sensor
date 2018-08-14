# Summary

This is a demo application that consists of several microservices and illustrates
the use of the tracing API. It can be run standalone, but requires Jaeger backend
to view the traces.

## Example
There are 3 microservices involved: service1, service2 and service3. All services are on localhost and listen to 9091, 9092, and 9093 respectively.

Service1: api-gateway service where all external http request will hit.
Service2: customer-catalog service. It will query a fake database to fulfill its operation.
Service3: fulfillment service. It is a fake order fulfillment service called by api-gateway after customer-catalog.

The fake database will experience some arbitrary delay.
Fulfillment service will result an internal server error. 

The final tracing span should show that how a request goes through each service and because some crucial step has failed,
the whole span should marked as failed.
When a request is made to a peer service a new span is created. If a an operation calls some location function that is crucial
in fufilling the operation (e.g. a database call), a new span is created. If the local function is not useful to fulfillment of 
the operation, no new span is created (e.g. print to a screen)

A span is tagged with notable events that happened in the span.

## Running

### Run Jaeger Backend

An all-in-one Jaeger backend is packaged as a Docker container with in-memory storage.

```bash
docker run -d --name jaeger -p6831:6831/udp -p16686:16686 jaegertracing/all-in-one:latest
```

Jaeger UI can be accessed at http://localhost:16686.

### Run Microservices
There should 3 microservices running. You can run in 3 terminals.

Make sure each of the service can be started properly.

```bash
go run ./examples/opentracing/service3/service.go
go run ./examples/opentracing/service2/service.go
go run ./examples/opentracing/service1/service.go
```

When a service is running you should see something like these:

```
{"level":"INFO","time":"2018-08-08T17:28:24.929Z","location":"opentracing/logger.go:31","message":"Initializing logging reporter\n","service":"service3","hostname":"xxx-xxx15"}
{"level":"INFO","time":"2018-08-08T17:28:24.929Z","location":"service3/service.go:41","message":"Starting service service3","service":"service3","hostname":"xxx-xxx15"}

```


Make a call to service1.

```
curl --header "X-Request-ID:12345" 'http://localhost:9091/tenant1/operationA?param1=hi'

```

Logs from Service 1, the service that you are hitting, should be something like this:

```
{"level":"INFO","time":"2018-08-08T21:47:36.508Z","location":"opentracing/logger.go:31","message":"Initializing logging reporter\n","service":"service1","hostname":"xxx"}
{"level":"INFO","time":"2018-08-08T21:47:36.509Z","location":"service1/service.go:43","message":"Starting service service1","service":"service1","hostname":"xxx"}
{"level":"INFO","time":"2018-08-08T21:47:36.509Z","location":"service1/service.go:50","message":"ready for handling requests","service":"service1","hostname":"xxx"}
{"level":"INFO","time":"2018-08-08T21:47:38.131Z","location":"service1/service.go:68","message":"Executing operation","service":"service1","hostname":"xxx","operation":"A","param1":"hi"}
{"level":"INFO","time":"2018-08-08T21:47:38.133Z","location":"service1/service.go:89","message":"response code from B","service":"service1","hostname":"xxx","response code":200}
{"level":"INFO","time":"2018-08-08T21:47:38.134Z","location":"service1/service.go:99","message":"response code from C","service":"service1","hostname":"xxx","response code":500}
{"level":"INFO","time":"2018-08-08T21:47:38.134Z","location":"opentracing/logger.go:31","message":"Reporting span 265081a2dc8ae62b:265081a2dc8ae62b:0:1","service":"service1","hostname":"xxx"}
{"level":"INFO","time":"2018-08-08T21:47:38.135Z","location":"opentracing/logger.go:31","message":"Reporting span 265081a2dc8ae62b:265081a2dc8ae62b:0:1","service":"service1","hostname":"xxx"}

```

View trace at backend.
Then open http://127.0.0.1:8080

Tracing Span
![alt text](./tracingui.png?raw=true)

Tracing UI
![alt text](./tracingspans.png?raw=true)