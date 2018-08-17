/*
The tracing package provides API to extend context for use by the metrics and logging packages, specifically for request id, operation id, and tenant id. Since services will have different ways for extracting these values the tracing package acts as an extensibility point to enrich context with these values. Additionally there is http middleware for extracting request ID and tenant ID from http requests.

Conceptual documentation can be found in the repository README.
     SSC-Observation Overview: https://cd.splunkdev.com/libraries/go-observation

More information on golang context can be found at to go blog https://blog.golang.org/context.

For support please use the #ssc-observation channel.
*/
package tracing
