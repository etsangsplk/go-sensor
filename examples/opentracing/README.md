# Summary

This is a demo application that consists of several microservices and illustrates
the use of the tracing API. It can be run standalone, but requires Jaeger backend
to view the traces.

## Example
There are 3 microservices involved: service1, service2 and service3. All services are on localhost and listen to 9091, 9092, and 9093 respectively.

Service1 will also spawn a server and also acts as client to call on itself on operationA, OperationA will call service2 and service3 to finish the request. service2 will call  a local function to inside its handler.

The final tracing span should show the top span when client make the call, as the request travels each microservice, a new span should be created when the subsequent service handles the request. The local function call should also be another span.

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
curl 'http://localhost:9091/operationA?param1=hi'

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
![alt text](./tracinggui.png?raw=true)

Tracing UI
![alt text](./tracingspans.png?raw=true)