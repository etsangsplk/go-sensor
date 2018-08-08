## Walkthrough

### Example Microservice App

```
go run ./examples/opentracing/service1/service.go &
go run ./examples/opentracing/service2/service.go &
go run ./examples/opentracing/service3/service.go &
```

```
$ curl 'http://localhost:9091/operation1?param1=hi'



```