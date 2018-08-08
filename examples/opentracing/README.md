# Summary

This is a demo application that consists of several microservices and illustrates
the use of the tracing API. It can be run standalone, but requires Jaeger backend
to view the traces. 

## Features


## Running

### Run Jaeger Backend

An all-in-one Jaeger backend is packaged as a Docker container with in-memory storage.

```bash
docker run -d --name jaeger -p6831:6831/udp -p16686:16686 jaegertracing/all-in-one:latest
```

Jaeger UI can be accessed at http://localhost:16686.

### Run Microservices 
There should 3 microservices running. You can run in 3 terminals or in background.
Service 1 depends on 2 and 3.

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
$ curl 'http://localhost:9091/operationA?param1=hi'

```

View trace at backend.
Then open http://127.0.0.1:8080
