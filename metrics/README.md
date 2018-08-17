# Table of Contents

**1. [Package Metrics](#package-metrics)**
* [Terminology and core concepts](#terminology-and-core-concepts)
* [Non-Golang services](#non-golang-services)
* [Instrument a service with a single metric](#instrument-a-service-with-a-single-metric)
* [Use histograms in Prometheus](#use-histograms-in-prometheus)

**2. [Metrics APIs](#metrics-apis)**
* [Prometheus API features](#prometheus-api-features)
* [SSC Metrics API features](#ssc-metrics-api-features)

**3. [Metrics Dimensions and Cardinality](#metrics-dimensions-and-cardinality)**

**4. [Define Custom Metrics](#define-custom-metrics)**

**5. [Observe Custom Metrics](#observe-custom-metrics)**

**6. [Results of Prometheus Server Scrapes](#results-of-prometheus-server-scrapes)**

**7. [HTTP Middleware](#http-middleware)**

**8. [Prometheus Labels](#prometheus-labels)**

**9. [Metrics Endpoint Discovery](#metrics-endpoint-discovery)**

**10. [Prometheus Dashboard](#prometheus-dashboard)**

**11. [Help and support](#help-and-support)**

**12. [Additional resources](#additional-resources)**


# Package Metrics

Use this package to instrument a custom metric. Metrics externalize measurements that you take within a service into a time series of numeric data. Also, the ssc-observation/metrics package provides additional common components such as http middleware. Instrumenting a service with metrics enables monitoring, alerting, troubleshooting and service capacity planning scenarios.

Metrics instrumentation of SSC services is mostly done with the [Prometheus client APIs](https://godoc.org/github.com/prometheus/client_golang/prometheus). 

At this time, metrics should not be used for usage meters for billing. The recommended approach for usage meters is still under investigation.

## Terminology and core concepts

* __Multi-dimensional data model__: A [model](https://prometheus.io/docs/concepts/data_model/) in which you can label a single metric (such as "http_requests_total") with multiple dimensions such as http method, operation ID, and status code. Queries can use aggregations to drill out, filters to drill in, or a combination of the two. For example, you can query all POST operations with status code 5XX, or just listCollections operation with status code 429 (a throttling error).
* Support for the following rich set of [metric types](https://prometheus.io/docs/concepts/metric_types/):
  * __Counter__: A cumulative metric that represents a value that only ever goes up. Counters support automatic rate calculations such as http requests per second.
  * __Gauge__: Represents a value that can arbitrarily go up and down.
  * __Histogram__: A [histogram] represents the distribution of a set of observations over a defined set of buckets. Histograms can be aggregated and are calculated in the prometheus server.
  * __Summary__: Represents the distribution of a set of observations over a defined set of phi-quantiles over a sliding time window. Summaries can not be aggregated and are calculated in the service itself, not in the prometheus server. At this time we are recomending that services not use summaries. If you have a use case please post it to the ssc-observation channel in slack.
* __ssc-observation__: Common library repository with APIs and middleware for use by SSC services for service instrumentation. Contains the packages metrics, tracing and logging. The tracing package provides functionality that is related to tracing, such as extracting the tenant ID and making it available, creating a request ID, and so on. The metrics and logging libraries make use of some of the functionality provided by the tracing package.
* __Metrics Middleware__: Common library code that provides consistent metrics on http requests and serves up the metrics endpoint through simple configuration.
* __Metrics Endpoint__: Prometheus uses a pull-based model, and each service must publish a metrics endpoint that is provided via common library middleware.

Read the [Prometheus Overview](https://prometheus.io/docs/introduction/overview/) for more details on the full prometheus system and its features.

Read the [Histograms](https://prometheus.io/docs/practices/histograms/) page for a more in depth discussion of this metric.

Counter, Gauge, and Histogram are the most frequently used metric types.

> **IMPORTANT:**
> Please see below for **specific instruction** regarding the use of [histogram] metrics.

## Non-Golang services
Prometheus has client libraries for Golang, Java, Scala and Python. Use these libraries for non-Golang services. We are working on enabling this for components instrumented with dropwizard.

Options for C++ support, such as possibly backporting a 3rd party package, are still under investigation.

A full list is available in the [Prometheus documentation](https://prometheus.io/docs/instrumenting/clientlibs/)

## Instrument a service with a single metric
Follow these steps to instrument a service with a single metric. For example, this metric is a measure of the number of open database connections dimensioned by database server host and database name. Since this value can increase and decrease over time, choose the gauge as the metric type. 

>> NOTE:
>> Database metrics like this are defined in a common database library, but each service also has its own custom metrics.

1. Define and register a metric. To define a metric, define the labels that you want to use to 'dimension' the data into different time series streams.

For example, you can choose two labels for 'host' and 'database'. This choice allows you to monitor the number of open connections to each unique host and database. Later, you can get a count of connections across all hosts or databases by using an aggregation function at query time.

Use the following example to define and register a matric.

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

2. Instrument your runtime code to observe this metric. 

For example, you can use the simplified getDb() function to observe the metric. This function is called by each request to get the proper connection pool (sql.DB) for the target host.

Use the following example to observe your metric.


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

> **NOTE:**
> If your metric has no dimensions then use prometheus.NewGauge(). See this [example](https://godoc.org/github.com/prometheus/client_golang/prometheus#hdr-A_Basic_Example) in the Prometheus docs.

3. Ensure your service serves a /metrics endpoint, and make your service discoverable by the Prometheus Server. Promethues uses a [pull-based model](#observing-custom-metrics) to scrape metrics from each service on a configured interval.

## Use histograms in Prometheus

The [Prometheus histogram] implementation is unique and counter-intuitive as compared to a [typical histogram]. Use the following information to use the Prometheus histogram correctly.

### Cumulative buckets

The bucket implementation is [cumulative], which means that the
buckets include the samples of the buckets below it. The [rationale]
for [cumulative] buckets is that buckets can be deleted
without breaking queries. 

While this is useful, it also makes it difficult to
query for the observation counts within a particular bucket, because
it includes observations from all buckets below it. Obtaining
observation counts for each bucket requires subtraction between the
two.

### Quantile estimation error

The [histogram quantile] calculation **assumes a linear
interpolation** within buckets. The upstream developers have
documented the [errors of quantile estimation], but basically the
[histogram quantile] function is always approximate, based on the
bucket layout. For this reason please honor the following
recommendations:

1. Do not use Prometheus [histogram] metrics for statistical analysis
   if you need an accurate value.  If you need to perform statistical
   analysis, then log each observation and use Splunk to perform analysis.
1. For alerting purposes, use [histogram] metrics,
   but do not use the [histogram quantile] function. Instead, create
   alerts directly against the bucket boundaries, which provide
   accurate measurements.

### Alerting on histogram bucket boundaries

If you' use a histogram for the purpose of alerting, you have two
goals:

1. Alert your team when `n` percentage of requests fall outside an
   acceptable bucket boundary.
1. Alert your team when bucket sizes are potentially incorrect.

#### Example: Latency alert

In the following example, assume the following inputs:

- The name of the histogram metric is: `k8s_demo_rest_api_histogram_seconds`
- The bucket boundaries include `1` and `10` the latter being the
  largest (below `+Inf`), in this case we're using [default buckets]
- The metric has `code`, `method`, and `operation` labels for grouping
- The expected latency is below `1s`
- You want to be alerted if `1%` of traffic falls above this threshold
  given a `5m` average
- You want to be alerted if `1%` of traffic is beyond the largest
  bucket

```
sum(rate(k8s_demo_rest_api_histogram_seconds_bucket{le="1"}[5m])
    / ignoring(le) rate(k8s_demo_rest_api_histogram_seconds_count[5m]))
by (code, method, operation) < 0.99
```

This alert triggers if more than `1%` of the traffic is above `1s`.

> NOTE:
> Because the calculation involves metrics with different labels, the
> `le` label must be `ignored`. The rest are identical.


# Metrics APIs
Instrumenting an SSC service involves using both the metrics package and the Prometheus client APIs. The metrics package provides mostly supporting capability.

## Prometheus API features
* API for defining new metrics and their dimensions (labels)
* API to register defined metrics for exposition to the Prometheus Server
* API to observe data values

## SSC Metrics API features
* Pre-defined middleware handlers.
* Possibly, a facility for pushing service metrics to non-Prometheus targets such as splunk. (future)
* Possibly, customizations of the local prometheus registry that all metrics must be registered with. (future)

The total set of metrics in the Prometheus server will include custom metrics from each SSC service, kubernetes metrics, and metrics pulled from AWS Cloudwatch.

# Metrics Dimensions and Cardinality
Metrics dimensions are a powerful capability that enable drill-in and drill-out at query time via aggregation functions.

To defining a metric with dimensions, also called labels, you actually define a metrics vector like CounterVec. It is a vector because each unique set of metric name and key-value pairs observed at runtime defines a new time series. For example, for the labels “method” and “statusCode” on the metric “http_requests_total” you might get three time series:
```
{"kvstore_http_requests_total", "method", "POST", "statusCode", "200"}
{"kvstore_http_requests_total", "method", "POST", "statusCode", "500"}
{"kvstore_http_requests_total", "method", "GET", "statusCode", "200"}
```
A new time series will be created when new values `GET` and `429` are later observed.
```
{"kvstore_http_requests_total", "method", "GET", "statusCode", "429"}
```
From the perspective of the service instrumentation, this all happens behind the scenes.

Metric cardinality is the number of unique <dimension>=<value> pair sets observed for a metric. For example, `{operationID=”createCollection”, code=”200”}` and `{operationID=”createCollection”, code=”500”}` are two unique sets of dimension-value pairs and will generate two time-series in the metric database.

Cardinality has an impact on aggregation performance and resource consumption of the Prometheus Server. Use these guidelines when choosing dimensions:

* Each unique set of dimension values creates a new time series in the Prometheus Server.
* Multiplying the cardinality of individual dimensions can provide an upper bound for the number of time-series created for a metric. For example, # Tenants x # HTTP Response Codes x # Operation IDs. In practice not all combinations will be observed, for example a 201 for a delete operation would not exist.
* __It is ok to use tenantID as a dimension where there are known use cases.__ Tenant level metrics can be valuable but be mindful of the dimension multiplier effect and the impact to aggregation performance. Consider that alerts will generally not be per-tenant.
* Do not use dimensions that have unbounded cardinality. Do not use requestID as a dimension. Use logging if you need to record each data point.
* Improve aggregation performance for analysis queries with Prometheus recording rules. These rules ‘pre-compute’ a new time series based on an aggregation. This method can speed queries for dashboards, for example.
* If the cardinality of using a dimension becomes too high, then use logging with the added dimension instead of or in addition to metrics.

The infrastructure team will add capacity and partitioning as necessary as Prometheus Server resource consumption increases. This will enable the server to support more time-series and data points but may not improve aggregation performance.

# Define Custom Metrics

To define a custom metric, you need to complete the following tasks:

1. Choose the metric type.
2. Define the metric name.
3. If you are using any labels, define the labels that you need to dimension the observations.
4. Register each metric once with the Prometheus registry. The service should call metrics.RegisterHTTPMetrics() from the service main.

For example:

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

> **NOTE:**
> Defining histograms requires additional configuration. For histograms, define the buckets that the observations are collected in. The default [prometheus.DefBuckets](https://godoc.org/github.com/prometheus/client_golang/prometheus#pkg-variables) provides a good starting point for network service request latencies.
> Histograms include a counter metric. So, for example, if you observe a request duration for each request, then you do not need to have a separate counter metric. See the [histogram documentation](https://prometheus.io/docs/practices/histograms/) for more details.


# Observe Custom Metrics
After you define and register the metric, you need to instrument the service code to observe the metric values to the Prometheus server. Recall that Prometheus is pull based, so from the service perspective observing a metric value is a local memory operation. The value observed is folded into previous observations since the last scrape. 

For example, if a gauge is observed multiple times between scrapes, then only the latest value is ingested into the Prometheus server.

Building on the database metrics defined earlier, observe runtime values with code like the following example. Use this example code to observe the latency histogram and db request counter metrics:

```go
        codeString := strconv.FormatInt(code, 10)
	metrics.DbRequestsDurationsHistogram.WithLabelValues(operation, codeString).Observe(elapsedSeconds)
```

> **NOTE:**
> * WithLabelValues() is used to get the specific time series for the given set of label values.
> * All label values must be converted to strings.

To observe the value on demand when scraping occurs, you can use functions like [prometheus.NewGaugeFunc](https://godoc.org/github.com/prometheus/client_golang/prometheus#NewGaugeFunc). This can be useful for metrics-based data structure sizes, such as lengths for channels and maps.

# Results of Prometheus Server Scrapes
If you were running your service locally, a simple curl to the metrics endpoint retreives text representation of the current metric observations. There is also a protobuf protocol that the prometheus server uses. For example: ```curl http://localhost:8066/service/metrics```.

Here is a small snippet of what that looks like, including some additional examples of go runtime metrics that the Prometheus client API provides:

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
The metrics package provides middleware for serving up the Prometheus metrics endpoint and for observing standard http metrics.

See the [ssc-observation README](https://github.com/splunk/ssc-observation) for details on how to configure the middleware pipeline.

You must use the following code to register the http metrics:

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
Prometheus adds additional labels to the time series stream for the metric job, the group, and the host. 

* The metric group is defined in the prometheus server configuration and typically has values such as "production", "staging", or "development". 
* The job defines a scraping configuration, which includes the location of the endpoints to scrape. 
* The host is simply the hostname of the target server that is scraped.

For example, when viewed in the Prometheus web console, you can see all of the labels that are defined by the service, and those added by prometheus:

```
db_requests_total{code="0",command="none",group="production",instance="localhost:8066",job="prometheus_production1"}
```

# Metrics Endpoint Discovery
The Prometheus Server running in the kubernetes cluster can automatically discover which pods have a metrics endpoint. It does this by annotating the service's kubernetes metadata with the port and path. Note that the port is the pod port, not the service port.

For example, see the .withAnnotationsmixin() call:
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
Once your service is deployed in the kubernetes environment, you can use the (very basic) Prometheus Dashboard to check if your service was properly discovered and metrics are getting observed. This is not the dashboarding solution that will be used in production, but is useful for quick validation. You can also run the prometheus server locally without much effort (see the prometheus documention).

The URL is environment URL prefixed with 'prometheus.'.

| Environment | URL
|-------------|----
| Playground1 | https://prometheus.playground1.dev.us-west-2.splunk8s.io.
| Staging     | https://prometheus.s1.stage.us-west-2.splunk8s.io/graph

Here is an example query that shows container CPU usage for kvstore:
```
sum (rate (container_cpu_usage_seconds_total{image!="",namespace="kvstore",pod_name=~"kvservice.*"}[1m])) by (pod_name,namespace)
```

# Help and support
For help, join the ssc-observation slack channel or contact ychristensen@splunk.com.

# Additional resources
The [Prometheus client API](https://godoc.org/github.com/prometheus/client_golang/prometheus) reference documentation provides more complete coverage than this README.

A more in depth look can be found in the [SSC Metrics and Alerts architecture document](https://docs.google.com/document/d/11AlcILE3S_7XE5t3hgUAYSJCsosbFbzGQW2VALcz-hU/edit).

The [Prometheus documentation](https://prometheus.io/docs/introduction/overview/) also has extensive information about its architecture and best practices.

[cumulative]: https://en.wikipedia.org/wiki/Histogram#Cumulative_histogram
[default buckets]: https://github.com/prometheus/client_golang/blob/180b8fdc22b4ea7750bcb43c925277654a1ea2f3/prometheus/histogram.go#L54-L64
[errors of quantile estimation]: https://prometheus.io/docs/practices/histograms/#errors-of-quantile-estimation
[histogram quantile]: https://prometheus.io/docs/prometheus/latest/querying/functions/#histogram_quantile
[histogram]: #correct-use-of-histograms
[Prometheus histogram]: https://prometheus.io/docs/practices/histograms/
[rationale]: https://www.robustperception.io/why-are-prometheus-histograms-cumulative/
[typical histogram]: https://en.wikipedia.org/wiki/Histogram
