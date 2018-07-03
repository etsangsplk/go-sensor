package logging_test

import (
	"github.com/splunk/ssc-observation/logging"
)

// This example demonstrates basic logging functions using the global logger.
func Example() {
	// Creating a logger and setting the global logger
	log := logging.New("service1")
	log.Info("Service starting")
	logging.SetGlobalLogger(log)

	// Using the global logger
	log = logging.Global()
	log.Info("message1")
	log.SetLevel(logging.DebugLevel)
	if log.Enabled(logging.DebugLevel) {
		// ... do something expensive here that should only be done for debug level ...
		log.Debug("message2")
	}

	// Call Flush before service exit
	defer log.Flush()

	// Sample output:
	// {"level":"INFO","time":"2018-07-03T00:51:10.722Z","location":"logging/basicexample_test.go:11","message":"Service starting","service":"service1","hostname":"ychristensen-MBP-C56E9"}
	// {"level":"INFO","time":"2018-07-03T00:51:10.722Z","location":"logging/basicexample_test.go:16","message":"message1","service":"service1","hostname":"ychristensen-MBP-C56E9"}
	// {"level":"DEBUG","time":"2018-07-03T00:51:10.722Z","location":"logging/basicexample_test.go:20","message":"message2","service":"service1","hostname":"ychristensen-MBP-C56E9"}
}
