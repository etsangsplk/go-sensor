# Package metrics

Metrics instrumentation of SSC services is mostly done with the [Prometheus client APIs](https://godoc.org/github.com/prometheus/client_golang/prometheus). The ssc-observation/metrics package provides additional common components such as http middleware. Instrumenting a service with metrics enables monitoring, alerting, troubleshooting and service capacity planning scenarios.

At this time, metrics should not be used for usage meters for billing. The recommended approach for usage meters is still under investigation.

## Support
For help, join the ssc-observation slack channel or contact ychristensen@splunk.com.

## Resources
The [Prometheus client API](https://godoc.org/github.com/prometheus/client_golang/prometheus) reference documentation provides more complete coverage than this README.

A more in depth look can be found in the [SSC Metrics and Alerts architecture document](https://docs.google.com/document/d/11AlcILE3S_7XE5t3hgUAYSJCsosbFbzGQW2VALcz-hU/edit).

The [Prometheus documentation](https://prometheus.io/docs/introduction/overview/) also has extensive information about its architecture and best practices.

## Concepts
Metrics instrumentation enables observability of a service by externalizing measurements taken inside the service into a time series of numeric data.

* __Multi-dimensional data model__: A [model](https://prometheus.io/docs/concepts/data_model/) where a single metric like "http_requests_total" can be labelled with multiple dimensions such as http method, operation ID and status code. Queries can use aggregations to drill out, and filters to drill in or a combination (e.g., all POST operations with status code 5XX, or just listCollections operation with status code 429 (a throttling error)).
* Support for a rich set of [metric types](https://prometheus.io/docs/concepts/metric_types/)
  * __Counter__: A counter is a cumulative metric that represents a value that only ever goes up. Counters support automatic rate calculations such as http requests per second.
  * __Gauge__: A gauge is a metric that represents a value that can arbitrarily go up and down.
  * __Histogram__: A [histogram] is a metric that represents the distribution of a set of observations over a defined set of buckets. Histograms can be aggregated and are calculated in the prometheus server.
  * __Summary__: A summary is a metric that represents the distribution of a set of observations over a defined set of phi-quantiles over a sliding time window. Summaries can not be aggregated and are calculated in the service itself (not in the prometheus server). At this time we are recomending that services not use summaries. If you have a use case please post it to the ssc-observation channel in slack.
* __ssc-observation__: Common library repository with APIs and middleware for use by SSC services for service instrumentation. Contains the packages metrics, tracing and logging. The tracing package provides functionality that is related to tracing, such as extracting the tenant ID and making it available, creating a request ID, and so on. The metrics and logging libraries make use of some of the functionality provided by the tracing package.
* __Metrics Middleware__: Common library code that provides consistent metrics on http requests and serves up the metrics endpoint through simple configuration.
* __Metrics Endpoint__: Prometheus uses a pull-based model and each service must publish a metrics endpoint (provided via common library middleware).

Read the [Prometheus Overview](https://prometheus.io/docs/introduction/overview/) for more details on the full prometheus system and its features.

Read the [Histograms](https://prometheus.io/docs/practices/histograms/) page for a more in depth discussion of this metric.

Counter, Gauge, and Histogram will be the most common metric types used.

> **IMPORTANT:**
> Please see below for **specific instruction** regarding the use of [histogram] metrics.

## Non-Golang Services
Prometheus has client libraries for golang, java, scala and python. Non-golang services are recommended to use these libraries. For components instrumented with dropwizard we are working on enabling this.

Options for C++ support (possibly backporting a 3rd party package) are still under investigation.

A full list is available in the [Prometheus documentation](https://prometheus.io/docs/instrumenting/clientlibs/)

## Basic Usage
Let’s start with a basic example of instrumenting a service with a single metric. In this case the metric is a measure of the number of open database connections dimensioned by database server host and database name. Since this value can increase and decrease over time the gauge is the chosen metric type. As a side note, database metrics like this will be defined in a common database library, but each service will have its own custom metrics as well.

The first step is to define the metric and register it. When defining a metric you typically define the labels that will be used to 'dimension' the data into different time series streams. In this case we have two labels for 'host' and 'database' so that we can monitor the number of open connections to each unique host and database. If we wanted to later get a count of connections across all hosts or databases we could simply use an aggregation function at query time.
```go
import 	"github.com/prometheus/client_golang/prometheus"

// NewGaugeVec defines a 'vector' of metrics. There are multiple because of the dimensions "host" and "database". Each unique value of "host" and "database" pairs creates a new time series.
var dbConnections = prometheus.NewGaugeVec(
    prometheus.GaugeOpts{
        Namespace: "kvstore",
        Name: "db_connections_open",
        Help: "A count of open database connections",
    },
    []string{"host", "database"},
)

func init() {
     // Each defined metric must be registered
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

If your metric has no dimensions then use prometheus.NewGauge(), [see example](https://godoc.org/github.com/prometheus/client_golang/prometheus#hdr-A_Basic_Example).

In addition to defining the metric and observing the values each service must serve a /metrics endpoint and make their service discoverable by the Prometheus Server. Promethues uses a pull based model to scrape metrics from each service on a configured interval. Those topics are covered later.

## Correct use of histograms

The [Prometheus histogram] implementation is counter intuitive as
compared to a [typical histogram]. The intent of this portion of the
documentation is to understand how it's unique, and how to use it
correctly.

### Cumulative buckets

The bucket implementation is [cumulative], which means that the
buckets include the samples of the buckets below it. The [rationale]
for [cumulative] buckets is that it allows buckets to be deleted
without breaking queries. While this useful, it makes it difficult to
query for the observation counts within a particular bucket (because
it includes observations from all buckets below it). Obtaining
observation counts for each bucket requires subtraction between the
two.

### Quantile estimation error

The [histogram quantile] calculation **assumes a linear
interpolation** within buckets. The upstream developers have
documented the [errors of quantile estimation], but basically the
[histogram quantile] function is always approximate (based on the
bucket layout). For this reason please honor the following
recommendations:

1. Do not use Prometheus [histogram] metrics for statistical analysis
   if you need an accurate value.  If you need to perform statistical
   analysis log each observation and use Splunk to perform analysis.
1. For alerting purposes it's appropriate to use [histogram] metrics,
   but do not use the [histogram quantile] function. Instead create
   alerts directly against the bucket boundaries which will provide
   accurate measurements.

### Alerting on histogram bucket boundaries

If you're using a histogram for the purpose of alerting, you have two
goals:

1. Alert your team when `n` percentage of requests fall outside an
   acceptable bucket boundary.
1. Alert your team when bucket sizes are potentially incorrect.

Let's assume the following example inputs:

- The name of the histogram metric is: `k8s_demo_rest_api_histogram_seconds`
- The bucket boundaries include `1` and `10` the latter being the
  largest (below `+Inf`), in this case we're using [default buckets]
- The metric has `code`, `method`, and `operation` labels for grouping
- The expected latency is below `1s`
- You want to be alerted if `1%` of traffic falls above this threshold
  given a `5m` average
- You want to be alerted if `1%` of traffic is beyond the largest
  bucket

#### Latency alert

```
sum(rate(k8s_demo_rest_api_histogram_seconds_bucket{le="1"}[5m])
    / ignoring(le) rate(k8s_demo_rest_api_histogram_seconds_count[5m]))
by (code, method, operation) < 0.99
```

This alert will tell us if more than `1%` of the traffic is above `1s`.

#### Boundary alert

- If the largest configured bucket is `10s`:

```
sum(rate(k8s_demo_rest_api_histogram_seconds_bucket{le="10"}[5m])
    / ignoring(le) rate(k8s_demo_rest_api_histogram_seconds_count[5m]))
by (code, method, operation) < 0.995
```

This will tell us when `0.5%` or more traffic is larger than the largest
bucket. If you see this you need to either adjust the bucket boundary
or make your program faster :)

> NOTE:
> Because the calculation involves metrics with different labels, the
> `le` label must be `ignored` (the rest are identical)

# Features of the Metrics APIs
As mentioned above, instrumenting an SSC service involves using both this package and the Prometheus client APIs. The metrics package provides largely supporting capability.

## Prometheus API Features
* API for defining new metrics and their dimensions (labels)
* API to register defined metrics for exposition to the Prometheus Server
* API to observe data values

## SSC Metrics API Features
* Pre-defined middleware handlers.
* Possibly, a facility for pushing service metrics to non-Prometheus targets such as splunk. (future)
* Possibly, customizations of the local prometheus registry that all metrics must be registered with. (future)

The total set of metrics in the Prometheus server will include custom metrics from each SSC service, kubernetes metrics, and metrics pulled from AWS Cloudwatch.

# Understanding Metrics Dimensions and Cardinality
Metrics dimensions are a powerful capability that enable drill-in and drill-out at query time via aggregation functions.

When defining a metric with dimensions (labels) you actually define a metrics vector like CounterVec. It is a vector because each unique set of metric name and key-value pairs observed at runtime will define a new time series. For example, for the labels “method” and “statusCode” on the metric “http_requests_total” you might get three time series:
```
{"kvstore_http_requests_total", "method", "POST", "statusCode", "200"}
{"kvstore_http_requests_total", "method", "POST", "statusCode", "500"}
{"kvstore_http_requests_total", "method", "GET", "statusCode", "200"}
```
A new time series will be created when new values `GET` and `429` are later observed.
```
{"kvstore_http_requests_total", "method", "GET", "statusCode", "429"}
```
From the perspective of the service instrumentation this all happens behind the scenes.

Metric cardinality is the number of unique <dimension>=<value> pair sets observed for a metric. For example, `{operationID=”createCollection”, code=”200”}` and `{operationID=”createCollection”, code=”500”}` are two unique sets of dimension-value pairs and will generate two time-series in the metric database.

Cardinality has an impact on aggregation performance and resource consumption of the Prometheus Server. Use these guidelines when choosing dimensions:
* Each unique set of dimension values creates a new time series in the Prometheus Server.
* Multiplying the cardinality of individual dimensions can provide an upper bound for the number of time-series created for a metric. For example, # Tenants x # HTTP Response Codes x # Operation IDs. In practice not all combinations will be observed, for example a 201 for a delete operation would not exist.
* __It is ok to use tenantID as a dimension where there are known use cases.__ Tenant level metrics can be valuable but be mindful of the dimension multiplier effect and the impact to aggregation performance. Consider that alerts will generally not be per-tenant.
* Do not use dimensions that have unbounded cardinality. Do not use requestID as a dimension, if you need to record each data point then use logging.
* Aggregation performance for analysis queries can be improved by using Prometheus recording rules which ‘pre-compute’ a new time series based on an aggregation. This can speed queries for dashboards, for example.
* If the cardinality of using a dimension becomes too high then use logging (with the added dimension) instead of or in addition to metrics.

The infrastructure team will add capacity and partitioning as necessary as Prometheus Server resource consumption increases. This will enable the server to support more time-series and data points but may not improve aggregation performance.

# Defining Custom Metrics
Defining a metric involves choosing the metric type, defining the metric name, and the labels (if any) that will be used to dimension the observations.

Defining histograms requires additional configuration. For histograms you must define the buckets that the observations will be collected in. The default [prometheus.DefBuckets](https://godoc.org/github.com/prometheus/client_golang/prometheus#pkg-variables) provides a good starting point for network service request latencies.

Note that histograms include a counter metric. So, for example, if you observe a request duration for each request you do not need to have a separate counter metric. See the [histogram documentation](https://prometheus.io/docs/practices/histograms/) for more details.

Once defined each metric must be registered once with the prometheus registry. The service should call metrics.RegisterHTTPMetrics() from the service main.
```go
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	DbRequestsDurationsHistogram = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "kvstore",
			Name:    "db_durations_histogram_seconds",
			Help:    "Database latency distributions",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"command", "code"},
	)
)

func Register() {
	prometheus.MustRegister(DbRequestsDurationsHistogram)
}
```

# Observing Custom Metrics
With the metric defined and registered, the service code must be instrumented to observe the metric values to the Prometheus server. Recall that Prometheus is pull based so from the service perspective observing a metric value is a local memory operation. The value observed will be folded into previous observations since the last scrape. For example, if a gauge is observed multiple times between scrapes only the latest value will be ingested into the Prometheus server.

Building on the database metrics defined above one would observe runtime values with code like the following. First, a couple of notes:
* WithLabelValues() is used to get the specific time series for the given set of label values.
* All label values must be converted to strings.

Here is the code to observe the latency histogram and db request counter metrics.
```go
        codeString := strconv.FormatInt(code, 10)
	metrics.DbRequestsDurationsHistogram.WithLabelValues(operation, codeString).Observe(elapsedSeconds)
```

If you want to observe the value on demand when scraping occurs you can use functions like [prometheus.NewGaugeFunc](https://godoc.org/github.com/prometheus/client_golang/prometheus#NewGaugeFunc). This can be useful for metrics based data structure sizes like lengths for channels and maps.

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

# HTTP Middleware
The metrics package provides middleware for serving up the Prometheus metrics endpoint and observing standard http metrics.

See the [ssc-observation README](https://github.com/splunk/ssc-observation) for details on how to configure the middleware pipeline.

Additionally the http metrics must be registered. To accomplish this:
```go
import (
      kvmetrics "github.com/splunk/kvstore-service/kvstore/metrics"
      "github.com/splunk/ssc-observation/metrics"
)

func configureAPI(api *operations.KVStoreAPI) http.Handler {
	// Register http metrics, passing it a string to scope the metrics, for example the service name.
	metrics.RegisterHTTPMetrics(serviceName)
	// Register custom service metrics
	kvmetrics.Register()
}
```

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

# Prometheus Dashboard
Once your service is deployed in the kubernetes environment you can use the (very basic) Prometheus Dashboard to check if your service was properly discovered and metrics are getting observed. This is not the dashboarding solution that will be used in production but is useful for quick validation. You can also run the prometheus server locally without much effort (see the prometheus documention).

The URL is environment URL prefixed with 'prometheus.'.

| Environment | URL
|-------------|----
| Playground1 | https://prometheus.playground1.dev.us-west-2.splunk8s.io.
| Staging     | https://prometheus.s1.stage.us-west-2.splunk8s.io/graph

Here is an example query that shows container CPU usage for kvstore.
```
sum (rate (container_cpu_usage_seconds_total{image!="",namespace="kvstore",pod_name=~"kvservice.*"}[1m])) by (pod_name,namespace)
```

[cumulative]: https://en.wikipedia.org/wiki/Histogram#Cumulative_histogram
[default buckets]: https://github.com/prometheus/client_golang/blob/180b8fdc22b4ea7750bcb43c925277654a1ea2f3/prometheus/histogram.go#L54-L64
[errors of quantile estimation]: https://prometheus.io/docs/practices/histograms/#errors-of-quantile-estimation
[histogram quantile]: https://prometheus.io/docs/prometheus/latest/querying/functions/#histogram_quantile
[histogram]: #correct-use-of-histograms
[Prometheus histogram]: https://prometheus.io/docs/practices/histograms/
[rationale]: https://www.robustperception.io/why-are-prometheus-histograms-cumulative/
[typical histogram]: https://en.wikipedia.org/wiki/Histogram
