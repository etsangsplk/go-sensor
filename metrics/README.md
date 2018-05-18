# Package metrics

SSC services use this metrics package combined with the [Prometheus client APIs](https://godoc.org/github.com/prometheus/client_golang/prometheus) to externalize their metrics. This instrumentation can enable monitoring, alerting, troubleshooting and service capacity planning scenarios. These packages should not be used for scenarios requiring durable, reliable delivery such as metering for billing or contractual SLA (scenarios directly involving $$).

## Status
__WORK IN PROGRESS - PROPOSAL__

This document proposes an API for metrics instrumentation of SSC Services written in Golang. Send feedback to ychristensen@splunk.com

## Concepts
Metrics instrumentation enables observability of a service by externalizing key measurements taken inside the service into a time series of numeric data.
 
The core concepts are:
* __Metric__: A time series of numeric values to be observed. The different metrics types gauge, counter, histogram and summary each have unique capabilities.
* __Metric Dimensions__: Dimensions (labels) partition the metric data into different pivots. For example an http request may be partitioned by http method values (“GET”, “POST”, ...) and status code values (500, 200, 429, ...). 
* __Metrics Server__: The server that provides metrics ingest, storage and query. A metrics server exploits special characteristics of time series data to optimize all of these features.
* __Instrumented Service__: The service that is instrumented with both custom and standard metrics. For example, the KV Store Service.

Metric dimensions (also called labels) are a powerfully important capability. Dimensions enable the operations engineer analyze the broader and narrower slices of the metrics data. Using query aggregation functions you can analyze different dimensions of the data without having to declare new metrics in the code for each pivot. For example, http request rates for all http POST requests, or all GET requests that were throttled (status=429), or simply all http requests across all service replicas.

## Basic Usage
This example demonstrates 1) defining a metric with labels (dimensions) and 2) externalizing (observing) the metric values to the metrics server. It defines a simple gauge metric to monitor the number of open database connections. Gauge is the right metric type to choose since open connections is a value that can increase and decrease. 

The first step is to define the metric and register it. When defining a metric you typically define the labels that will be used to 'dimension' the data into different time series streams. In this case we have two labels for 'host' and 'database' so that we can monitor the number of open connections to each unique host and database. If we wanted to later get a count of connections across all hosts or databases we could simply use an aggregation function at query time.
```go
import 	"github.com/prometheus/client_golang/prometheus"

// NewGaugeVec defines a 'vector' of metrics. There are multiple because of the "host" label below. Each unique valeu for "host" will define a unique time series stream.
var dbConnections = prometheus.NewGaugeVec(
    prometheus.GaugeOpts{
        Name: "db_connections_active",
        Help: "A count of active database connections",
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

     // Observe metrics
     // First, get the gauge instance for the given host value. If this is the first time host has been seen then a new instance will be created. 
     gauge := dbConnections.WithLabelValues(host, database)
     // Next, set the current value. This is just a local memory operation.
     count := float64(db.Stats().OpenConnections)
     gauge.Set(count)
     
     return db
}
```

# Features of the Metrics APIs
As mentioned above, instrumenting an SSC service involves using both this package and the Prometheus client APIs. The metrics package provides largely supporting capability.

## Prometheus API Features
* [Multi-dimensional data model](https://prometheus.io/docs/concepts/data_model/) where a single metric like "http_requests_total" can be dimensioned with multiple labels such as http method and status code.
* Support for a rich set of [metric types](https://prometheus.io/docs/concepts/metric_types/)
  ** Counter: A counter is a cumulative metric that represents a value that only ever goes up. Counters support automatic rate calculations such as http requests per second.
  ** Gauge: A gauge is a metric that represents a value that can arbitrarily go up and down.
  ** Histogram: A histogram is a metric that represents the distribution of a set of observations over a defined set of buckets. Histograms can be aggregated and are calculated in the prometheus server.
  ** Summary: A summary is a metric that represents the distribution of a set of observations over a defined set of phi-quantiles over a sliding time window. Summaries can not be aggregated and are calculated in the service itself (not in the prometheus server).

Read the [Prometheus Overview](https://prometheus.io/docs/introduction/overview/) for more details on the full prometheus system and its features.

Read the [Histograms and Summaries](https://prometheus.io/docs/practices/histograms/) page for a more in depth discussion of those metrics.

Counter, Gauge, and Histogram will be the most common metric types used. Summary has unique capabilities but can’t be aggregated (for example if you want to combine data across different replicas).

## SSC Metrics API Features
* Pre-defined middleware handler for emitting http metrics.
* Metrics configuration and operational support
* Possibly, a facility for pushing service metrics to non-Prometheus targets such as splunk.
* Possibly, customizations of the local prometheus registry that all metrics must be registered with

Note that other shared libraries such as the IAC client library will define and publish their own metrics.

# Defining Custom Metrics
Defining a metric involves choosing the metric type, defining the metric name, and the labels (if any) that will be used to dimension the observations. If one or more labels are defined then you will declare a vector of metrics like CounterVec. A metrics vector is a collector of a bundle of metrics. The bundle exists because each unique set of metric name and key-value pairs observed at runtime will define a new time series. For example, for the labels “method” and “statusCode” on the metric “http_requests_total” you might get three time series:
* {"http_requests_total", "method", "POST", "statusCode", "200"}
* {"http_requests_total", "method", "POST", "statusCode", "500"}
* {"http_requests_total", "method", "GET", "statusCode", "200"}

A new time series will be created when new values "GET" and "429" are later observed.
* {"http_requests_total", "method", "GET", "statusCode", "429"}

From the perspective of the service instrumentation this all happens behind the scenes. 

Defining histograms and services requires additional configuration. For histograms you must define the buckets that the observations will be collected in. The default [prometheus.DefBuckets](https://godoc.org/github.com/prometheus/client_golang/prometheus#pkg-variables) provides a good starting point for network service request latencies.  Similarly summaries have more complex configuration as well, see [SummaryOpts](https://godoc.org/github.com/prometheus/client_golang/prometheus#SummaryOpts).

Once defined each metric must be registered once with the prometheus registry.

By convention, the custom service metrics are defined in a service package called 'metrics'. The service should call metrics.Register() from the service main.

```go
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	DbRequests = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "db_requests_total",
			Help: "How many database requests processed, partitioned by command and code",
		},
		[]string{"command", "code"},
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
	prometheus.MustRegister(DbRequests)
	prometheus.MustRegister(DbRequestsDurationsHistogram)
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

# Prometheus Labels
Prometheus will add additional labels to the time series stream for the metric job, group and the host. The metric group is defined in the prometheus server configuration and will typically have values like "production", "staging", or "development". The job defines a scraping configuration which includes the location of the endpoints to scrape. And finally the host is simply the hostname of the target server scraped.

For example, when viewed in the prometheus web console you will see all of the labels (those defined by the service and those added by prometheus):
```
db_requests_total{code="0",command="none",group="production",instance="localhost:8066",job="prometheus_production1"}
```

