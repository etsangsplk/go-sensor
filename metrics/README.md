# Package metrics

SSC services are primarily instrumented with metrics using the [Prometheus client APIs](https://godoc.org/github.com/prometheus/client_golang/prometheus). The ssc-observation/metrics package provides additional common components such as http middleware. Instrumenting a service with metrics enables monitoring, alerting, troubleshooting and service capacity planning scenarios.

At this time, metrics should not be used for usage meters for billing. The recommended approach for usage meters is still under investigation.

## Support
For help, join the ssc-observation slack channel or contact ychristensen@splunk.com.

## Architecture
A more in depth architecture document that covers both metrics and alerts is available at [here](https://docs.google.com/document/d/11AlcILE3S_7XE5t3hgUAYSJCsosbFbzGQW2VALcz-hU/edit).

## Concepts
Metrics instrumentation enables observability of a service by externalizing measurements taken inside the service into a time series of numeric data.

* __Multi-dimensional data model__: A [model](https://prometheus.io/docs/concepts/data_model/) where a single metric like "http_requests_total" can be labelled with multiple dimensions such as http method, operation ID and status code. Queries can use aggregations to drill out, and filters to drill in or a combination (e.g., all POST operations with status code 5XX, or just listCollections operation with status code 429 (a throttling error)).
* Support for a rich set of [metric types](https://prometheus.io/docs/concepts/metric_types/)
  * __Counter__: A counter is a cumulative metric that represents a value that only ever goes up. Counters support automatic rate calculations such as http requests per second.
  * __Gauge__: A gauge is a metric that represents a value that can arbitrarily go up and down.
  * __Histogram__: A histogram is a metric that represents the distribution of a set of observations over a defined set of buckets. Histograms can be aggregated and are calculated in the prometheus server.
  * __Summary__: A summary is a metric that represents the distribution of a set of observations over a defined set of phi-quantiles over a sliding time window. Summaries can not be aggregated and are calculated in the service itself (not in the prometheus server).
* __ssc-observation__: Common library repository with APIs and middleware for use by SSC services for service instrumentation. Contains the packages metrics, tracing and logging. The tracing package provides functionality that is related to tracing, such as extracting the tenant ID and making it available, creating a request ID, and so on. The metrics and logging libraries make use of some of the functionality provided by the tracing package.
* __Metrics Middleware__: Common library code that provides consistent metrics on http requests and serves up the metrics endpoint through simple configuration.
* __Metrics Endpoint__: Prometheus uses a pull-based model and each service must publish a metrics endpoint (provided via common library middleware). 

Read the [Prometheus Overview](https://prometheus.io/docs/introduction/overview/) for more details on the full prometheus system and its features.

Read the [Histograms and Summaries](https://prometheus.io/docs/practices/histograms/) page for a more in depth discussion of those metrics.

Counter, Gauge, and Histogram will be the most common metric types used. Summary has unique capabilities but can’t be aggregated (for example if you want to combine data across different replicas).


## Basic Usage
Let’s start with a basic example of instrumenting a service with a single metric. In this case the metric is a measure of the number of open database connections dimensioned by database server host and database name. Since this value can increase and decrease over time the gauge is the chosen metric type. As a side note, database metrics like this will be defined in a common database library, but each service will have its own custom metrics as well.

The first step is to define the metric and register it. When defining a metric you typically define the labels that will be used to 'dimension' the data into different time series streams. In this case we have two labels for 'host' and 'database' so that we can monitor the number of open connections to each unique host and database. If we wanted to later get a count of connections across all hosts or databases we could simply use an aggregation function at query time.
```go
import 	"github.com/prometheus/client_golang/prometheus"

// NewGaugeVec defines a 'vector' of metrics. There are multiple because of the "host" label below. Each unique valeu for "host" will define a unique time series stream.
var dbConnections = prometheus.NewGaugeVec(
    prometheus.GaugeOpts{
        Name: "db_connections_open",
        Help: "A count of open database connections",
    },
    []string{"host", "database"},
)

func init() {
     // Each metric defined must be registered
     prometheus.MustRegister(dbConnections)
}
```

With the metric defined we can now instrument our runtime code to observe the metric. In this case the simplified getDb() function is the chosen place to observe the metric. This is a good place because this function is called by each request to get the proper connection pool (sql.DB) for the target host.

```go
func getDB(host string, database string) *sql.DB {
     db := ensureConnected(host, database)

     // Observe the metric for open connections
     // First, get the gauge instance for the given host value. If this is the first time host has been seen then a new instance will be created.
     // Next, set the current value. This is just a local memory operation.     
     dbConnections.WithLabelValues(host, database)
          .Set(float64(db.Stats().OpenConnections))
     
     return db
}
```

In addition to defining the metric and observing the values each service must serve a /metrics endpoint and make their service discoverable by the Prometheus Server. Promethues uses a pull based model to scrape metrics from each service on a configured interval. Those topics are covered later.


# Features of the Metrics APIs
As mentioned above, instrumenting an SSC service involves using both this package and the Prometheus client APIs. The metrics package provides largely supporting capability.

## Prometheus API Features
* API for defining new metrics and their dimensions (labels)
* API to register defined metrics for exposition to the Prometheus Server
* API to observe data values

## SSC Metrics API Features
* Pre-defined middleware handler that services up the metrics endpoint
* Metrics configuration and operational support
* Possibly, a facility for pushing service metrics to non-Prometheus targets such as splunk.
* Possibly, customizations of the local prometheus registry that all metrics must be registered with

Note that other shared libraries such as the IAC client library will define and publish their own metrics.

# Understanding Metrics Dimensions
Metrics dimensions are a powerful capability and provided necessary to drill-in and out on metrics data without changing service code. When defining a metric with labels you actually define a metrics vector like CounterVec. A metrics vector is a collector of a bundle of metrics. The bundle exists because each unique set of metric name and key-value pairs observed at runtime will define a new time series. For example, for the labels “method” and “statusCode” on the metric “http_requests_total” you might get three time series:
* {"http_requests_total", "method", "POST", "statusCode", "200"}
* {"http_requests_total", "method", "POST", "statusCode", "500"}
* {"http_requests_total", "method", "GET", "statusCode", "200"}

A new time series will be created when new values "GET" and "429" are later observed.
* {"http_requests_total", "method", "GET", "statusCode", "429"}

From the perspective of the service instrumentation this all happens behind the scenes.

One must be careful to understand the cardinality of the dimensions used. Something like user id will certainly have too many unique values. Bounded values like http status codes or operation ids are fine. TenantID is still TBD but will likely be allowed initially until we better understand the implications.

# Defining Custom Metrics
Defining a metric involves choosing the metric type, defining the metric name, and the labels (if any) that will be used to dimension the observations.

Defining histograms and services requires additional configuration. For histograms you must define the buckets that the observations will be collected in. The default [prometheus.DefBuckets](https://godoc.org/github.com/prometheus/client_golang/prometheus#pkg-variables) provides a good starting point for network service request latencies.  Similarly summaries have more complex configuration as well, see [SummaryOpts](https://godoc.org/github.com/prometheus/client_golang/prometheus#SummaryOpts).

Once defined each metric must be registered once with the prometheus registry.

By convention, the custom service metrics are defined in a service package in your service repository called 'metrics'. The service should call metrics.Register() from the service main.

```go
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	DbRequests = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "db_requests_bytes",
			Help: "Count of bytes transferred to and from the database, partitioned by command and direction",
		},
		[]string{"command", "direction"},
	)
	DbRequestsDurationsHistogram = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "db_durations_histogram_seconds",
			Help:    "Database latency distributions",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"command", "code"},
	)
)

func Register() {
	prometheus.MustRegister(DbRequests, DbRequestsDurationsHistogram)
}
```

# Observing Custom Metrics
Once defined and registered the service code must be instrumented to observe the metric values to the Prometheus server. Recall that Prometheus is pull based so from the service perspective observing a metric value is a local memory operation. The value observed will be folded into previous observations since the last scrape. For example, if a gauge is observed multiple times between scrapes only the latest value will be ingested into the Prometheus server. 

Building on the database metrics defined above one would observe runtime values with code like the following. A couple of notes:
* WithLabelValues() is used to get the specific time series for the given set of label values.
* All label values must be converted to strings.

Here is the code to observe the latency histogram and db request counter metrics.
```go
        codeString := strconv.FormatInt(code, 10)
	metrics.DbRequestsDurationsHistogram.WithLabelValues(command, codeString).Observe(elapsedSeconds)
	metrics.DbRequests.WithLabelValues(command, codeString).Add(1)
```
# Looking at what the Prometheus Server Scrapes
If you were running your service locally a simple curl to the metrics endpoint will get you a text representation of the current metric observations (there is also a protobuf protocol that the prometheus server uses). For example ```curl http://localhost:8066/service/metrics```.

Here is a small snippet of what that looks like including some additional examples of go runtime metrics that are provided by the prometheus client API.
```
# HELP db_requests_total How many database requests processed, partitioned by command
# TYPE db_requests_total counter
db_requests_total{code="0",command="delete"} 11
db_requests_total{code="29",command="delete"} 2
db_requests_total{code="0",command="get"} 48
db_requests_total{code="0",command="insert"} 52

# HELP db_durations_histogram_seconds Database latency distributions
# TYPE db_durations_histogram_seconds histogram
db_durations_histogram_seconds_bucket{code="0",command="insert",le="0.005"} 34
db_durations_histogram_seconds_bucket{code="0",command="insert",le="0.01"} 50
db_durations_histogram_seconds_bucket{code="0",command="insert",le="0.025"} 52
db_durations_histogram_seconds_bucket{code="0",command="insert",le="0.05"} 52
<...snip...>
db_durations_histogram_seconds_bucket{code="0",command="insert",le="10"} 52
db_durations_histogram_seconds_bucket{code="0",command="insert",le="+Inf"} 52

# HELP go_gc_duration_seconds A summary of the GC invocation durations.
# TYPE go_gc_duration_seconds summary
go_gc_duration_seconds{quantile="0"} 3.8911e-05
go_gc_duration_seconds{quantile="0.25"} 6.0017e-05
go_gc_duration_seconds{quantile="0.5"} 6.6778e-05
go_gc_duration_seconds{quantile="0.75"} 8.2406e-05
go_gc_duration_seconds{quantile="1"} 0.00020086
go_gc_duration_seconds_sum 0.004903172
go_gc_duration_seconds_count 64

# HELP go_memstats_gc_sys_bytes Number of bytes used for garbage collection system metadata.
# TYPE go_memstats_gc_sys_bytes gauge
go_memstats_gc_sys_bytes 708608
```

# Emitting Standard HTTP Metrics
The SSC metrics package defines middleware that can be used to observe http metrics in a standard way. Http request rates, latency distributions, and active http request counts are all supported. These metrics are all dimensioned on http method, operation name, and response status code. In a swagger-generated service the operation name is derived from the API operation.ID, in non-swagger services the operation name is derived from context.

Note, as shown below, only authorized, routed will have metrics observed. (open issue: do we want to observe unauthorized or unroutable requests separately (say to avoid skewing latency distributions) or if its ok to group all the 4XX type requests (user error) together. Either way, all request should be observed in some metric.)

This example demonstrates how to enable standard http metrics using the metrics package middleware and metrics definitions.
```go
import (
      "github.com/splunk/kvstore-service/kvstore/metrics"
      sscmetrics "github.com/splunk/ssc-observation/metrics" // TODO: revisit this naming conflict
)

func configureAPI(api *operations.KVStoreAPI) http.Handler {
        // Register http metrics
        sscmetrics.Register()
        // Register custom service metrics
	metrics.Register()
}

func setupMiddlewares(handler http.Handler) http.Handler {
        // Add the metrics middleware to the http pipeline
	return sscmetrics.NewHttpMetricsHandler(handler)
}
```
# Registering the Metrics Endpoint
Prometheus uses a pull based model to periodically scrape metrics from each monitored service. Each service must expose a metrics endpoint which for SSC will be '/service/metrics'. The metrics library provides middleware for serving up this endpoint and can be registered as follows.

```go
// The middleware configuration happens before anything, this middleware also applies to serving the swagger.json document.
// So this is a good place to plug in a panic handling middleware, logging and metrics
func setupGlobalMiddleware(handler http.Handler) http.Handler {
	return handlers.NewPanicHandler(
		handlers.NewPrometheusHandler(
			logging.NewRequestHandler(logging.Global(),
				handlers.NewRateLimitHandler(handler))))
}
```

__NOTE__: The metrics library is not yet available (as of 6/4/2018) but should be very soon.

# Prometheus Labels
Prometheus will add additional labels to the time series stream for the metric job, group and the host. The metric group is defined in the prometheus server configuration and will typically have values like "production", "staging", or "development". The job defines a scraping configuration which includes the location of the endpoints to scrape. And finally the host is simply the hostname of the target server scraped.

For example, when viewed in the prometheus web console you will see all of the labels (those defined by the service and those added by prometheus):
```
db_requests_total{code="0",command="none",group="production",instance="localhost:8066",job="prometheus_production1"}
```

# Metrics Endpoint Discovery
The Prometheus Server running in the kubernetes cluster can automatically discover which pods have a metrics endpoint. This is done by annotating the service's kubernetes metadata with the port and path. Note that the port is the pod port, not the service port.

For example see the .withAnnotationsmixin() call below:
```
local service = kService
  // select on the same labels as defined in the container
  .new(name, podLabel)
  .withPorts(servicePort)
  .withType(params.type)
  // add SSC metadata to service resource
  + utils.metadata(kService.mixin, metadata)
  + kService.mixin.metadata.withAnnotationsMixin({"prometheus.io/port": "8066", "prometheus.io/path": "/service/metrics"});
```

In YAML this looks like:
```
 Metadata:
  Annotations:
    prometheus.io/port: 8066
    prometheus.io/path: /service/metrics
```



