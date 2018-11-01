# Table of Contents

To initiaze a lightstep tracer properly, you need to set some environment varibles.

# Environment Variables

(Optional) variables will be set to LightStep default values. Refer to lightstep documentation for default values.

| Envrionment variable       |  Format |  Comment  | Example |
| :------------- |:-------------:| :----- | :-----: |
| TRACER_URI_SCHEME |  host:port | (Optional) collector endpoint | http or https |
| TRACER_COLLECTOR_HOST_PORT |  host:port | (Optional, but will use Lightstep public satellite) collector endpoint | 127.0.0.1:8080 |
| TRACER_COLLECTOR_SEND_PLAINTEXT | string   | (Optional) flag to indicate  whether to encrypt data send to the endpoint  | true or false |
| LIGHTSTEP_ACCESSTOKEN |  string | (Required) access toke for lightstep  | abcde |
| LIGHTSTEP_API_HOST_PORT | host:port | (Required) LightStep web API endpoint |  127.0.0.1:8080 |
| LIGHTSTEP_API_SEND_PLAINTEXT |  bool as string | f(Optional) lag to indicate  whether to encrypt data send to the endpoint  | true or false |
| LIGHTSTEP_MAXBUFFERED_SPANS | integer as string  |  (Optional) maximum number of spans that will be buffered  | 10 |
| LIGHTSTEP_MAX_LOGKEY_LEN | integer as string | (Optional) maximum allowable size (in characters) of an OpenTracing logging key. Longer keys are truncated. | 10 |
| LIGHTSTEP_MAX_LOG_VALUE_LEN |  integer as string | (Optional) maximum allowable size (in characters) of an OpenTracing logging value. Longer values are truncated. | 10 |
| LIGHTSTEP_MAX_LOGS_PER_SPAN |  integer as string | (Optional) limits the number of logs in a single span  | 100 |
| LIGHTSTEP_REPORTING_PERIOD | integer as string | (Optional) maximum duration of time (in sec) between sending spans to a collector | 60 |
| LIGHTSTEP_MIN_REPORTING_PERIOD  |  integer as string |  (Optional) minimum duration (in sec) of time between sending spans to a collector | 30 |
| LIGHTSTEP_DROP_SPANLOGS |  bool as string   |  (Optional) turns log events on all Spans into no-ops  | false |
| LIGHTSTEP_TRANSPORT_PROTOCOL | string |  (Optional) transport protocol used by collector to send spans, default to usegrpc  | usehttp |

Note Default settings for Lightstep collector is to use public lightstep satellite with:

* Host: "collector-grpc.lightstep.com"

* Port: 443

* Plaintext: false

* usegrp

For sending Span via HTTPS 

* Host: "collector.lightstep.com"

* Port: 443

* Plaintext: false

* usehttp


Reference:

[lightstep go client](https://github.com/lightstep/lightstep-tracer-go/blob/master/README.md)

[lightstep tracer configuration](https://github.com/lightstep/lightstep-tracer-go/blob/master/options.go)