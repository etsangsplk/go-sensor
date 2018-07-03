/*
Package logging package provides structured leveled logging in accordance with the SSC standard logging format. The format is defined at http://go/ssc-logging-format. Features include request loggers, component loggers, and http access logging. This logging package wraps a more complicated logging package (zap) and exposes just the APIs needed to instrument your service according to the SSC standard. Your service should not take any dependencies on zap APIs. If you need a zap feature exposed please use the slack channel below for support.

Conceptual documentation can be found in the repository READMEs:
     SSC-Observation Overview: https://github.com/splunk/ssc-observation
     Logging Package: https://github.com/splunk/ssc-observation/tree/master/logging

For support please use the #ssc-observation channel.
*/
package logging
