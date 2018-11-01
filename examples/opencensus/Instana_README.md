# Summary

This is a demo application that consists of several Microservices and illustrates
the use of the tracing API. It can be run standalone, but requires Jaeger backend
to view the traces.

## Example
There are 3 Microservices involved: service1, service2 and service3. All services are on localhost and listen to 9091, 9092, and 9093 respectively.

service1: api-gateway service where all external http request will hit.
service2: customer-catalog service. It will query a fake database to fulfill its operation.
service3: fulfillment service. It is a fake order fulfillment service called by api-gateway after customer-catalog.

The fake database will experience some arbitrary delay.
Fulfillment service will result an internal server error.

The final tracing span should show that how a request goes through each service and because some crucial step has failed,
the whole span should marked as failed.
When a request is made to a peer service a new span is created. If a an operation calls some location function that is crucial
in fufilling the operation (e.g. a database call), a new span is created. If the local function is not useful to fulfillment of
the operation, no new span is created (e.g. print to a screen)

A span is tagged with notable events that happened in the span.

## Running

Please read Instana README (Reference section below)

### Instana host
Please contact Kubernetes team.

# Environment Variables

Note if INSTANA_AGENT_HOST and INSTANA_AGENT_PORT are not set, it will use default INSTANA settings and send to their public
endpoint. For default settings, you don't need to set an environmen variables.
```
export INSTANA_AGENT_HOST=localhost
export INSTANA_AGENT_PORT=42699
```
are the default values

## Datastore
Either have a local instance of postgres running or verify that you can connect to AWS aurora.

## Instana Agent
You can either setup your own Instana Agent so your instrumented application
can forward the events to, or you need to talk to Kubernetes team for details.

(https://docs.instana.io/quick_start/agent_setup/other/)
(https://docs.instana.io/quick_start/agent_configuration/)

I use the following to spin one up locally (localhost) that use HTTP as transport, the collector is listening at localhost:8181, not using secure transport (using http).

```
sudo docker run  --volume /var/run/docker.sock:/var/run/docker.sock   --volume /dev:/dev   --volume /sys:/sys   --volume /var/log:/var/log  --privileged  --pid=host   --ipc=host  --env="INSTANA_AGENT_KEY=ASK_YOUR_LEAD"  --env="INSTANA_AGENT_ENDPOINT=ASK_YOUR_LEAD"  --env="INSTANA_AGENT_ENDPOINT_PORT=ASK_YOUR_LEAD" -p 443:443  -p 42699:42699 instana/agent
```

### Instana Agent

Ask kubernetes team. They should have some account setup.

### Run Microservices in Docker
The 3 microservices can be run in docker containers via docker-compose. Install Docker and Docker Compose if you don't already have it: https://docs.docker.com/compose/install. You can check if it is already installed by running `docker-compose`.
Running in Docker is required to connect to the instana agent.

First build the docker image:
```bash
make opentracing-example-docker
```

Next, run the containers:
```bash
make opentracing-docker-run
```

Finally, in another terminal window make a request to service1:
```
curl --header "X-Request-ID:12345" 'http://localhost:9091/tenant1/operationA?param1=hi'
```

This should output "Internal Server Error". The error is expected since the example is demonstrating an operational failure.


### Run Microservices On Localhost
There should 3 Microservices running. You can run in 3 terminals.

Note: when running locally edit the line in ./examples/opentracing/service1/service.go to set service2Host and service3Host to localhost.

### Run Microservices
There should 3 Microservices running. You can run in 3 terminals.

If you just want to test out the services, you can make sure each of the service can be started properly.

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

Instana README
![alt text](../../opentracing/instana/README.md?raw=true)

Tracing Span
![alt text](./tracingui.png?raw=true)

Tracing UI
![alt text](./tracingspans.png?raw=true)